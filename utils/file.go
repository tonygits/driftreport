package utils

import (
	"encoding/json"
	"io"
	"log"
	"os"

	"github.com/driftreport/entities"
)

func ParseTerraformState(filePath string) (*entities.TerraformState, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Error reading Terraform state file: %v", err)
		return nil, err
	}

	// Parse JSON
	var state entities.TerraformState
	err = json.Unmarshal(data, &state)
	if err != nil {
		log.Fatalf("Error parsing Terraform state file: %v", err)
		return nil, err
	}

	return &state, nil
}

func ReadFile(reader io.Reader) ([]byte, error) {
	all, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return all, nil
}
