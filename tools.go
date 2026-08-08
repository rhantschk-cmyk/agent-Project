package main

import "github.com/ollama/ollama/api"

//Structs to fill up when creating tool
type Property struct {
	name string
	category string
	description string
}

type Tool struct {
	name string 
	description string 
}

func build_tool(all_properties []Property, target_tool Tool) api.Tool {
	// Standard Tool stuff for tool function
	props := api.NewToolPropertiesMap()
	for _, prop := range all_properties {
		props.Set(prop.name, api.ToolProperty{
			Type: api.PropertyType{prop.category},
			Description: prop.description,
		})
	}

	tool := api.Tool {
		Type: "function",
		Function: api.ToolFunction {
			Name: target_tool.name,
			Description: target_tool.description,
			Parameters: api.ToolFunctionParameters{
				Type: "object",
				Properties: props,
			},
		},
	}

	return tool
}
