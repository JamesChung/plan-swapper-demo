package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

type State struct {
	Values map[string]*Module `json:"values"`
}

type Module struct {
	Resources    []StateResource `json:"resources"`
	ChildModules []*Module       `json:"child_modules"`
}

type StateResource struct {
	Address string           `json:"address"`
	Values  *json.RawMessage `json:"values"`
}

type ResourceMap map[string]*json.RawMessage

func (r ResourceMap) AddModuleResources(resources []StateResource) {
	for _, resource := range resources {
		r[resource.Address] = resource.Values
	}
}

func (r ResourceMap) AddModule(module *Module) {
	if module.Resources != nil {
		r.AddModuleResources(module.Resources)
	}
	if module.ChildModules != nil {
		for _, mod := range module.ChildModules {
			r.AddModule(mod)
		}
	}
}

func GetStateFileResourceMapping(stateFilePath string) (ResourceMap, error) {
	resourceMap := make(ResourceMap)

	stateFile, err := os.Open(stateFilePath)
	if err != nil {
		return nil, err
	}

	stateFileBytes, err := io.ReadAll(stateFile)
	if err != nil {
		return nil, err
	}

	state := State{}
	err = json.Unmarshal(stateFileBytes, &state)
	if err != nil {
		return nil, err
	}

	for _, module := range state.Values {
		resourceMap.AddModule(module)
	}

	return resourceMap, nil
}

func (r ResourceMap) GenerateUpdatedPlanJSONFile(planFilePath string) ([]byte, error) {
	planFile, err := os.Open(planFilePath)
	if err != nil {
		log.Fatalln(err)
	}

	planFileBytes, err := io.ReadAll(planFile)
	if err != nil {
		log.Fatalln(err)
	}

	plan := make(map[string]any)

	err = json.Unmarshal(planFileBytes, &plan)
	if err != nil {
		log.Fatalln(err)
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

func main() {
	resourceMap, err := GetStateFileResourceMapping("state.json")
	if err != nil {
		log.Fatalln(err)
	}

	data, err := resourceMap.GenerateUpdatedPlanJSONFile("plan.json")
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(string(data))
}
