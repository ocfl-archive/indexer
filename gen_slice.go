package main

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
)

type Schema struct {
	Items       *Schema            `json:"items"`
	Properties  map[string]*Schema `json:"properties"`
	Definitions map[string]*Schema `json:"definitions"`
	Ref         string             `json:"$ref"`
	AnyOf       []*Schema          `json:"anyOf"`
	Required    []string           `json:"required"`
}

func main() {
	data, err := os.ReadFile("data/csl-data.json")
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	var s Schema
	if err := json.Unmarshal(data, &s); err != nil {
		fmt.Printf("Error unmarshaling json: %v\n", err)
		return
	}

	mandatoryFields := make(map[string]struct{})
	optionalFields := make(map[string]struct{})

	// Top-level is an array, so we look into its items
	if s.Items != nil {
		extractFields("", s.Items, &s, mandatoryFields, optionalFields)
	}

	var sortedMandatoryFields []string
	for f := range mandatoryFields {
		sortedMandatoryFields = append(sortedMandatoryFields, f)
	}
	slices.Sort(sortedMandatoryFields)

	var sortedOptionalFields []string
	for f := range optionalFields {
		sortedOptionalFields = append(sortedOptionalFields, f)
	}
	slices.Sort(sortedOptionalFields)

	fmt.Println("var CSLFieldsMandatory = []string{")
	for _, f := range sortedMandatoryFields {
		fmt.Printf("\t\"%s\",\n", f)
	}
	fmt.Println("}")

	fmt.Println("\nvar CSLFieldsOptional = []string{")
	for _, f := range sortedOptionalFields {
		fmt.Printf("\t\"%s\",\n", f)
	}
	fmt.Println("}")
}

func extractFields(prefix string, s *Schema, root *Schema, mandatoryFields, optionalFields map[string]struct{}) {
	if s == nil {
		return
	}

	// Resolve $ref if present
	if s.Ref != "" {
		refParts := strings.Split(s.Ref, "/")
		if len(refParts) == 3 && refParts[0] == "#" && refParts[1] == "definitions" {
			defName := refParts[2]
			if def, ok := root.Definitions[defName]; ok {
				extractFields(prefix, def, root, mandatoryFields, optionalFields)
			}
		}
		return
	}

	// Check for anyOf (common in the provided schema for definitions)
	if s.AnyOf != nil {
		for _, sub := range s.AnyOf {
			extractFields(prefix, sub, root, mandatoryFields, optionalFields)
		}
	}

	// Determine mandatory fields for this level
	requiredMap := make(map[string]struct{})
	for _, r := range s.Required {
		requiredMap[r] = struct{}{}
	}

	// Properties
	for name, prop := range s.Properties {
		fieldName := name
		if prefix != "" {
			fieldName = prefix + "/" + name
		}

		if _, ok := requiredMap[name]; ok {
			mandatoryFields[fieldName] = struct{}{}
		} else {
			optionalFields[fieldName] = struct{}{}
		}

		// If it's an array, look into its items
		if prop.Items != nil {
			extractFields(fieldName, prop.Items, root, mandatoryFields, optionalFields)
		} else {
			// If it's an object (or potentially an object), look into its properties
			extractFields(fieldName, prop, root, mandatoryFields, optionalFields)
		}
	}
}
