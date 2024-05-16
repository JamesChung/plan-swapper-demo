package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

type StateV4 struct {
	Version          uint64            `json:"version"`
	TerraformVersion string            `json:"terraform_version"`
	Serial           uint64            `json:"serial"`
	Lineage          string            `json:"lineage"`
	Resources        []ResourceStateV4 `json:"resources"`
}

type ResourceStateV4 struct {
	Module         string                  `json:"module,omitempty"`
	Mode           string                  `json:"mode"`
	Type           string                  `json:"type"`
	Name           string                  `json:"name"`
	EachMode       string                  `json:"each,omitempty"`
	ProviderConfig string                  `json:"provider"`
	Instances      []InstanceObjectStateV4 `json:"instances"`
}

type InstanceObjectStateV4 struct {
	IndexKey any    `json:"index_key,omitempty"`
	Status   string `json:"status,omitempty"`
	Deposed  string `json:"deposed,omitempty"`

	SchemaVersion           uint64            `json:"schema_version"`
	AttributesRaw           json.RawMessage   `json:"attributes,omitempty"`
	AttributesFlat          map[string]string `json:"attributes_flat,omitempty"`
	AttributeSensitivePaths json.RawMessage   `json:"sensitive_attributes,omitempty"`

	PrivateRaw []byte `json:"private,omitempty"`

	Dependencies []string `json:"dependencies,omitempty"`

	CreateBeforeDestroy bool `json:"create_before_destroy,omitempty"`
}

type ResourceMap map[string]*json.RawMessage

func (r ResourceMap) GenerateUpdatedPlanJSONFile(planFilePath string) ([]byte, error) {
	planFile, err := os.Open(planFilePath)
	if err != nil {
		return nil, err
	}

	planFileBytes, err := io.ReadAll(planFile)
	if err != nil {
		return nil, err
	}

	plan := make(map[string]any)

	err = json.Unmarshal(planFileBytes, &plan)
	if err != nil {
		return nil, err
	}

	if plannedValues, ok := plan["planned_values"].(map[string]any); ok {
		for _, module := range plannedValues {
			r.traversePlannedValuesModule(module)
		}
	}

	if resourceChanges, ok := plan["resource_changes"].([]any); ok {
		for _, resource := range resourceChanges {
			if res, ok := resource.(map[string]any); ok {
				address := res["address"].(string)
				if c, ok := res["change"].(map[string]any); ok {
					if values, ok := r[address]; ok {
						c["after"] = values
						c["after_unknown"] = struct{}{}
					}
				}
			}
		}
	}

	return json.Marshal(&plan)
}

func (r ResourceMap) traversePlannedValuesModule(module any) {
	if mod, ok := module.(map[string]any); ok {
		if resources, ok := mod["resources"].([]any); ok {
			for _, resource := range resources {
				res := resource.(map[string]any)
				if address, ok := res["address"].(string); ok {
					if values, ok := r[address]; ok {
						res["values"] = values
					}
				}
			}
		}
		if children, ok := mod["child_modules"].([]any); ok {
			for _, child := range children {
				r.traversePlannedValuesModule(child)
			}
		}
	}
}

func GetStateV4ResourceMapping(stateFilePath string) (ResourceMap, error) {
	resourceMap := make(ResourceMap)

	stateFile, err := os.Open(stateFilePath)
	if err != nil {
		return nil, err
	}

	stateFileBytes, err := io.ReadAll(stateFile)
	if err != nil {
		return nil, err
	}

	state := StateV4{}
	err = json.Unmarshal(stateFileBytes, &state)
	if err != nil {
		return nil, err
	}

	for _, resource := range state.Resources {
		var address string
		if resource.Module != "" {
			address += fmt.Sprintf("%s.", resource.Module)
		}
		address += fmt.Sprintf("%s.%s", resource.Type, resource.Name)
		for _, instance := range resource.Instances {
			var addr string
			switch val := instance.IndexKey.(type) {
			case float64:
				addr = fmt.Sprintf("%s[%d]", address, int(val))
			case string:
				addr = fmt.Sprintf("%s[\"%s\"]", address, val)
			default:
				addr = address
			}
			resourceMap[addr] = &instance.AttributesRaw
		}
	}

	return resourceMap, nil
}

func main() {
	res, err := GetStateV4ResourceMapping("state.json")
	if err != nil {
		log.Fatal(err)
	}
	data, err := res.GenerateUpdatedPlanJSONFile("plan.json")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
