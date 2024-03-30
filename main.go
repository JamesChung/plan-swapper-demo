package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

type State struct {
	Values map[string]Module `json:"values"`
}

type Module struct {
	Resources []StateResource `json:"resources"`
}

type StateResource struct {
	Address string           `json:"address"`
	Values  *json.RawMessage `json:"values"`
}

func GetStateFileResourceMapping(filePath string) (map[string]*json.RawMessage, error) {
	resourceMap := make(map[string]*json.RawMessage)

	stateFile, err := os.Open(filePath)
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

	for _, v := range state.Values {
		for _, resource := range v.Resources {
			resourceMap[resource.Address] = resource.Values
		}
	}

	return resourceMap, nil
}

func GenerateUpdatedPlanJSONFile(planFilePath string, resourceMap map[string]*json.RawMessage) ([]byte, error) {
	planFile, err := os.Open(planFilePath)
	if err != nil {
		log.Fatalln(err)
	}

	planFileBytes, err := io.ReadAll(planFile)
	if err != nil {
		log.Fatalln(err)
	}

	m := make(map[string]any)

	err = json.Unmarshal(planFileBytes, &m)
	if err != nil {
		log.Fatalln(err)
	}

	if resourceChanges, ok := m["resource_changes"].([]any); ok {
		for _, resource := range resourceChanges {
			if r, ok := resource.(map[string]any); ok {
				address := r["address"].(string)
				if c, ok := r["change"].(map[string]any); ok {
					if values, ok := resourceMap[address]; ok {
						c["after"] = values
						c["after_unknown"] = struct{}{}
					}
				}
			}
		}
	}

	return json.Marshal(&m)
}

func main() {
	resourceMap, err := GetStateFileResourceMapping("state.json")
	if err != nil {
		log.Fatalln(err)
	}

	data, err := GenerateUpdatedPlanJSONFile("plan.json", resourceMap)
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(string(data))
}
