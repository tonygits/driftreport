package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/driftreport/entities"
	"github.com/driftreport/providers"
	"github.com/driftreport/utils"
)

type (
	DriftReport interface {
		PrintDriftReport(ctx context.Context) error
		DriftChecker(context.Context, string, *entities.Instance, map[string]bool) (*entities.DriftReport, error)
	}

	AppDriftReport struct {
		awsProvider providers.AWSProvider
	}
)

func NewDriftReport(awsProvider providers.AWSProvider) DriftReport {
	return &AppDriftReport{
		awsProvider: awsProvider,
	}
}

//PrintDriftReport prints (in JSON) the reports added on the buffered channel
func (s *AppDriftReport) PrintDriftReport(ctx context.Context) error {
	tfInstances, err := loadTerraformStateInstances("../terraform.tfstate.json")
	if err != nil {
		utils.Logger.Sugar().Errorf("error loading Terraform state: %v", err)
		return &entities.CustomError{
			StatusCode: http.StatusInternalServerError,
			Err:        err,
		}
	}

	tfInstanceMap := make(map[string]*entities.Instance)
	instanceIds := make([]string, 0)
	if len(tfInstances) > 0 {
		for _, instance := range tfInstances {
			instanceIds = append(instanceIds, instance.Attributes.InstanceID)
			tfInstanceMap[instance.Attributes.InstanceID] = instance
		}
	}

	attributesList := "instance_type,security_groups,tags"
	if len(instanceIds) == 0 {
		utils.Logger.Sugar().Error("Error: No instances specified")
		return &entities.CustomError{
			StatusCode: http.StatusBadRequest,
			Err:        errors.New("no instances specified"),
		}
	}

	attributes := make(map[string]bool)
	for _, attr := range strings.Split(attributesList, ",") {
		attributes[attr] = true
	}

	var wg sync.WaitGroup
	reports := make(chan *entities.DriftReport, len(instanceIds))

	for _, instanceId := range instanceIds {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			var instance *entities.Instance
			if _, ok := tfInstanceMap[id]; ok {
				instance = tfInstanceMap[id]
			}
			report, err := s.DriftChecker(ctx, id, instance, attributes)
			if err != nil {
				utils.Logger.Sugar().Errorf("error detecting drift for instance %s: %v", id, err)
				return
			}
			reports <- report
		}(instanceId)
	}

	wg.Wait()
	close(reports)

	for report := range reports {
		output, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(output))
	}
	return nil
}

// loadTerraformStateInstances reads and parses instances from the terraform.tfstate file
func loadTerraformStateInstances(filePath string) ([]*entities.Instance, error) {
	tfInstances := make([]*entities.Instance, 0)
	terraformState, err := utils.ParseTerraformState(filePath)
	if err != nil {
		utils.Logger.Sugar().Errorf("failed to parse terraform state file: %v", err)
		return tfInstances, err
	}

	// Look for an EC2 instance in the state file
	for _, resource := range terraformState.Resources {
		if resource.Type == "aws_instance" {
			tfInstances = append(tfInstances, resource.Instances...)
		}
	}

	return tfInstances, nil
}

//DriftChecker compares instance from AWS EC2 and terraform tfstate json file and creates a drift report
func (s *AppDriftReport) DriftChecker(ctx context.Context, instanceID string, instance *entities.Instance, attributes map[string]bool) (*entities.DriftReport, error) {
	if !attributes["instance_type"] || !attributes["security_groups"] || !attributes["tags"] {
		log.Println("no attributes for instance")
		return nil, &entities.CustomError{
			StatusCode: http.StatusBadRequest,
			Err:        errors.New("no attributes for instance"),
		}
	}

	//get instance from AWS with instance ID
	awsEC2Instance, err := s.awsProvider.GetEC2Instance(ctx, instanceID)
	if err != nil {
		log.Printf("failed to get ec2 instance %v", err)
		return nil, &entities.CustomError{
			StatusCode: http.StatusFailedDependency,
			Err:        err,
		}
	}

	var tfInstantType string
	tfSecurityGroups := make([]string, 0)
	var tfTags map[string]string
	if instance != nil {
		tfInstantType = instance.Attributes.Type
		tfSecurityGroups = instance.Attributes.SecurityGroups
		tfTags = instance.Attributes.Tags
	}

	//compare the instance from terraform state and AWS EC2 instance and add the differences in a map
	differences := make(map[string]string)
	if attributes["instance_type"] && tfInstantType != "" && awsEC2Instance.InstanceType != tfInstantType {
		differences["instance_type"] = fmt.Sprintf("AWS: %s, Terraform: %s", awsEC2Instance.InstanceType, tfInstantType)
	}

	if attributes["security_groups"] && len(tfSecurityGroups) > 0 {
		if fmt.Sprintf("%v", awsEC2Instance.SecurityGroups) != fmt.Sprintf("%v", tfSecurityGroups) {
			differences["security_groups"] = fmt.Sprintf("AWS: %v, Terraform: %v", awsEC2Instance.SecurityGroups, tfSecurityGroups)
		}
	}

	if attributes["tags"] {
		if fmt.Sprintf("%v", awsEC2Instance.Tags) != fmt.Sprintf("%v", tfTags) {
			differences["tags"] = fmt.Sprintf("AWS: %v, Terraform: %v", awsEC2Instance.Tags, tfTags)
		}
	}

	return &entities.DriftReport{
		InstanceID:  instanceID,
		Drifted:     len(differences) > 0,
		Differences: differences,
	}, nil
}
