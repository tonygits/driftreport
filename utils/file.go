package utils

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/driftreport/entities"
)

//ParseTerraformState parses data from the terraform tfstate json file to TerraformState struct
func ParseTerraformState(filePath string) (*entities.TerraformState, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		Logger.Sugar().Errorf("error reading terraform state file: %v", err)
		return nil, &entities.CustomError{
			StatusCode: http.StatusInternalServerError,
			Err:        err,
		}
	}

	//check if file is empty
	if len(data) == 0 {
		Logger.Sugar().Error(".tfstate is empty")
		return nil, &entities.CustomError{
			StatusCode: http.StatusBadRequest,
			Err:        errors.New(".tfstate is empty"),
		}
	}

	// Parse JSON
	var state entities.TerraformState
	err = json.Unmarshal(data, &state)
	if err != nil {
		log.Printf("Error parsing Terraform state file: %v", err)
		return nil, &entities.CustomError{
			StatusCode: http.StatusInternalServerError,
			Err:        err,
		}
	}

	return &state, nil
}
