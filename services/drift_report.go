package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/driftreport/entities"
	"github.com/driftreport/providers"
	"github.com/driftreport/utils"
)

type (
	DriftReport interface {
		GenerateDriftReport(ctx context.Context) error
		DetectDrift(context.Context, string, *entities.Instance, map[string]bool) (*entities.DriftReport, error)
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

func (s *AppDriftReport) GenerateDriftReport(ctx context.Context) error {
	instanceIds, tfInstances, err := loadTerraformInstances("terraform.tfstate.json")
	if err != nil {
		utils.Logger.Sugar().Errorf("error loading Terraform state: %v", err)
		return err
	}

	tfInstanceMap := make(map[string]*entities.Instance)
	if len(tfInstances) > 0 {
		for _, instance := range tfInstances {
			tfInstanceMap[instance.Attributes.InstanceID] = instance
		}
	}
	attributesList := "instance_type,security_groups,tags"
	if len(instanceIds) == 0 {
		utils.Logger.Sugar().Error("Error: No instances specified")
		return fmt.Errorf("no instances specified")
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
			report, err := s.DetectDrift(ctx, id, instance, attributes)
			if err != nil {
				log.Printf("Error detecting drift for instance %s: %v", id, err)
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

// LoadTerraformState reads and parses the terraform.tfstate file
func loadTerraformInstances(filePath string) ([]string, []*entities.Instance, error) {
	instanceIds := make([]string, 0)
	tfInstances := make([]*entities.Instance, 0)
	terraformState, err := utils.ParseTerraformState(filePath)
	if err != nil {
		utils.Logger.Sugar().Errorf("failed to parse terraform state file: %v", err)
		return instanceIds, tfInstances, err
	}

	// Look for an EC2 instance in the state file
	for _, resource := range terraformState.Resources {
		if resource.Type == "aws_instance" {
			if len(resource.Instances) > 0 {
				for i, resourceInstance := range resource.Instances {
					tfInstances = append(tfInstances, resourceInstance)
					instanceIds = append(instanceIds, resource.Instances[i].Attributes.InstanceID)
				}
			}
		}
	}
	return instanceIds, tfInstances, nil
}

func (s *AppDriftReport) DetectDrift(ctx context.Context, instanceID string, instance *entities.Instance, attributes map[string]bool) (*entities.DriftReport, error) {
	if !attributes["instance_type"] || !attributes["security_groups"] || !attributes["tags"] {
		log.Println("no attributes for instance")
		return nil, errors.New("no attributes for instance")
	}

	awsProvider, err := providers.NewAWSProvider()
	if err != nil {
		log.Printf("failed to initialize aws provider %v", err)
		return nil, err
	}
	awsConfig, err := awsProvider.GetEC2Instance(ctx, instanceID)
	if err != nil {
		log.Printf("failed to get ec2 instance %v", err)
		return nil, err
	}

	var tfInstantType string
	tfSecurityGroups := make([]string, 0)
	var tfTags map[string]string
	if instance != nil {
		tfInstantType = instance.Attributes.Type
		tfSecurityGroups = instance.Attributes.SecurityGroups
		tfTags = instance.Attributes.Tags
	}

	differences := make(map[string]string)
	if attributes["instance_type"] && tfInstantType != "" && awsConfig.InstanceType != tfInstantType {
		differences["instance_type"] = fmt.Sprintf("AWS: %s, Terraform: %s", awsConfig.InstanceType, tfInstantType)
	}

	if attributes["security_groups"] && len(tfSecurityGroups) > 0 {
		if fmt.Sprintf("%v", awsConfig.SecurityGroups) != fmt.Sprintf("%v", tfSecurityGroups) {
			differences["security_groups"] = fmt.Sprintf("AWS: %v, Terraform: %v", awsConfig.SecurityGroups, tfSecurityGroups)
		}
	}

	if attributes["tags"] {
		if fmt.Sprintf("%v", awsConfig.Tags) != fmt.Sprintf("%v", tfTags) {
			differences["tags"] = fmt.Sprintf("AWS: %v, Terraform: %v", awsConfig.Tags, tfTags)
		}
	}

	return &entities.DriftReport{
		InstanceID:  instanceID,
		Drifted:     len(differences) > 0,
		Differences: differences,
	}, nil
}
