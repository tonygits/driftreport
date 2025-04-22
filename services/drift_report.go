package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/driftreport/entities"
	"github.com/driftreport/providers"
	"github.com/driftreport/utils"
)

type (
	DriftReportService interface {
		PrintDriftReport(ctx context.Context) error
	}

	AppDriftReportService struct {
		awsProvider providers.AWSProvider
	}
)

func NewDriftReportService(awsProvider providers.AWSProvider) DriftReportService {
	return &AppDriftReportService{
		awsProvider: awsProvider,
	}
}

// PrintDriftReport gets the instance map from Terraform state and AWS EC2 instance and parses both to drift checker
// and prints (in JSON) the reports added on the buffered channel
func (s *AppDriftReportService) PrintDriftReport(ctx context.Context) error {
	attributes := make(map[string]bool)
	attributesList := "instance_type,security_groups,tags"
	for _, attr := range strings.Split(attributesList, ",") {
		attributes[attr] = true
	}

	// Load the terraform instance map and instances ids from  the terraform file
	tfInstanceMap, instanceIds, err := loadTerraformStateInstances("../terraform.tfstate.json")
	if err != nil {
		utils.Logger.Sugar().Errorf("error loading Terraform state: %v", err)
		return &entities.CustomError{
			StatusCode: http.StatusInternalServerError,
			Err:        err,
		}
	}

	// Check if any instances were found in the terraform state
	if len(instanceIds) == 0 {
		utils.Logger.Sugar().Error("Error: No tfInstances specified")
		return &entities.CustomError{
			StatusCode: http.StatusBadRequest,
			Err:        errors.New("no instances specified"),
		}
	}

	// Get the AWS EC2 instance map with instance ids
	awsEC2InstanceMap, err := s.awsProvider.GetEC2Instances(ctx, instanceIds)
	if err != nil {
		utils.Logger.Sugar().Errorf("error retrieving AWS config with err %v", err)
		return &entities.CustomError{
			StatusCode: http.StatusInternalServerError,
			Err:        err,
		}
	}

	var wg sync.WaitGroup
	reports := make(chan *entities.DriftReport, len(instanceIds))
	for _, id := range instanceIds {
		wg.Add(1)
		go func(instanceID string) {
			defer wg.Done()
			var awsEC2Instance *entities.EC2Instance
			var tfInstance *entities.EC2Instance
			if _, ok := awsEC2InstanceMap[instanceID]; ok {
				awsEC2Instance = awsEC2InstanceMap[instanceID]
			}
			if _, ok := tfInstanceMap[instanceID]; ok {
				tfInstance = tfInstanceMap[instanceID]
			}
			select {
			case <-ctx.Done():
				log.Printf("drift check for instance %v failed with reason - %v", instanceID, ctx.Err())
				return
			default:
				report, err := driftChecker(instanceID, awsEC2Instance, tfInstance, attributes)
				if err != nil {
					utils.Logger.Sugar().Errorf("error checking drift for instance %s: %v", instanceID, err)
					return
				}
				reports <- report
			}
		}(id)
	}
	wg.Wait()
	close(reports)

	var allReports []*entities.DriftReport
	fmt.Println("\nPrint drift reports in JSON")
	for report := range reports {
		output, _ := json.MarshalIndent(report, "", "  ")
		allReports = append(allReports, report)
		fmt.Println(string(output))
	}
	fmt.Println("\nPrint drift reports in tabular format")
	printDriftTable(allReports)
	return nil
}

// loadTerraformStateInstances reads and parses instances from the terraform.tfstate file
func loadTerraformStateInstances(filePath string) (map[string]*entities.EC2Instance, []string, error) {
	tfInstanceIds := make([]string, 0)
	tfInstanceMap := make(map[string]*entities.EC2Instance)
	terraformState, err := utils.ParseTerraformState(filePath)
	if err != nil {
		utils.Logger.Sugar().Errorf("failed to parse terraform state file: %v", err)
		return tfInstanceMap, tfInstanceIds, err
	}

	// Filter out EC2 instances from the terraform state
	tfInstances := make([]*entities.Instance, 0)
	for _, resource := range terraformState.Resources {
		if resource.Type == "aws_instance" {
			tfInstances = append(tfInstances, resource.Instances...)
		}
	}

	// Check if any EC2 instances were found
	if len(tfInstances) == 0 {
		utils.Logger.Sugar().Error("no instances found in terraform state")
		return tfInstanceMap, tfInstanceIds, &entities.CustomError{
			StatusCode: http.StatusBadRequest,
			Err:        errors.New("no instances found in terraform state"),
		}
	}

	// Iterate over the instances and populate the tfInstanceMap
	for _, instance := range tfInstances {
		tfInstanceIds = append(tfInstanceIds, instance.Attributes.InstanceID)
		attrs := instance.Attributes
		tfInstanceMap[instance.Attributes.InstanceID] = &entities.EC2Instance{
			InstanceType:   attrs.Type,
			SecurityGroups: attrs.SecurityGroups,
			Tags:           attrs.Tags,
		}
	}

	return tfInstanceMap, tfInstanceIds, nil
}

//DriftChecker compares instance from AWS EC2 and terraform tfstate json file and creates a drift report
func driftChecker(instanceId string, ec2Instance, tfInstance *entities.EC2Instance, attributes map[string]bool) (*entities.DriftReport, error) {
	if !attributes["instance_type"] && !attributes["security_groups"] && !attributes["tags"] {
		log.Println("no attributes for instance")
		return nil, &entities.CustomError{
			StatusCode: http.StatusBadRequest,
			Err:        errors.New("no attributes for instance"),
		}
	}

	if tfInstance == nil {
		utils.Logger.Sugar().Errorf("error retrieving terraform config for instance %s", instanceId)
		return nil, &entities.CustomError{
			StatusCode: http.StatusBadRequest,
			Err:        errors.New("terraform instance not set"),
		}
	}

	if ec2Instance == nil {
		utils.Logger.Sugar().Errorf("error retrieving AWS config for instance %s", instanceId)
		return nil, &entities.CustomError{
			StatusCode: http.StatusBadRequest,
			Err:        errors.New("ec2 instance not set"),
		}
	}

	differences := make(map[string]string)
	if attributes["instance_type"] && ec2Instance.InstanceType != tfInstance.InstanceType {
		differences["instance_type"] = fmt.Sprintf("AWS: %s, Terraform: %s", ec2Instance.InstanceType, tfInstance.InstanceType)
	}
	if attributes["security_groups"] && fmt.Sprintf("%v", ec2Instance.SecurityGroups) != fmt.Sprintf("%v", tfInstance.SecurityGroups) {
		differences["security_groups"] = fmt.Sprintf("AWS: %v, Terraform: %v", ec2Instance.SecurityGroups, tfInstance.SecurityGroups)
	}
	fmt.Println("tftags", tfInstance.Tags)
	if attributes["tags"] && fmt.Sprintf("%v", ec2Instance.Tags) != fmt.Sprintf("%v", tfInstance.Tags) {
		differences["tags"] = fmt.Sprintf("AWS: %v, Terraform: %v", ec2Instance.Tags, tfInstance.Tags)
	}

	return &entities.DriftReport{
		InstanceID:  instanceId,
		Drifted:     len(differences) > 0,
		Differences: differences,
	}, nil
}

//printDriftTable prints drift report in a tabular format
func printDriftTable(reports []*entities.DriftReport) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.Debug)
	fmt.Fprintln(writer, "INSTANCE ID\tDRIFTED\tATTRIBUTES WITH DIFFERENCES")
	for _, r := range reports {
		if r.Drifted {
			detailLines := make([]string, 0, len(r.Differences))
			for attr, detail := range r.Differences {
				detailLines = append(detailLines, fmt.Sprintf("%s: %s", attr, detail))
			}
			fmt.Fprintf(writer, "%s\t%t\t%s\n", r.InstanceID, r.Drifted, strings.Join(detailLines, ",\n "))
		} else {
			fmt.Fprintf(writer, "%s\t%t\t%s\n", r.InstanceID, r.Drifted, "No differences")
		}
	}
	writer.Flush()
}
