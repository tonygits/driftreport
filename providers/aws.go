package providers

import (
	"context"
	"errors"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/driftreport/entities"
	"github.com/driftreport/utils"
)

type (
	AWSProvider interface {
		GetEC2Instances(ctx context.Context, instanceIDs []string) (map[string]*entities.EC2Instance, error)
	}

	AppAWSProvider struct {
		awsRegion string
		client    *ec2.Client
	}
)

func NewAWSProvider(
	awsRegion string,
) (AWSProvider, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(awsRegion))
	if err != nil {
		utils.Logger.Sugar().Errorf("failed to load default config: %v", err)
		return nil, &entities.CustomError{
			StatusCode: http.StatusUnauthorized,
			Err:        err,
		}
	}

	client := ec2.NewFromConfig(cfg)
	return &AppAWSProvider{
		awsRegion: awsRegion,
		client:    client,
	}, nil
}

//GetEC2Instances get EC2 instance from AWS account
func (a *AppAWSProvider) GetEC2Instances(ctx context.Context, instanceIDs []string) (map[string]*entities.EC2Instance, error) {
	input := &ec2.DescribeInstancesInput{
		InstanceIds: instanceIDs,
	}

	result, err := a.client.DescribeInstances(ctx, input)
	if err != nil {
		utils.Logger.Sugar().Errorf("failed to describe instances: %v", err)
		return nil, &entities.CustomError{
			StatusCode: http.StatusBadRequest,
			Err:        err,
		}
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		utils.Logger.Sugar().Warn("no instances found")
		return nil, &entities.CustomError{
			StatusCode: http.StatusNotFound,
			Err:        errors.New("no instances found"),
		}
	}

	instanceMap := make(map[string]*entities.EC2Instance)
	typeInstances := make([]types.Instance, 0)
	for _, res := range result.Reservations {
		typeInstances = append(typeInstances, res.Instances...)
	}

	for _, instance := range typeInstances {
		id := *instance.InstanceId

		var sgs []string
		for _, sg := range instance.SecurityGroups {
			sgs = append(sgs, *sg.GroupId)
		}

		tags := make(map[string]string)
		for _, tag := range instance.Tags {
			tags[*tag.Key] = *tag.Value
		}

		instanceMap[id] = &entities.EC2Instance{
			InstanceType:   string(instance.InstanceType),
			SecurityGroups: sgs,
			Tags:           tags,
		}
	}

	return instanceMap, nil
}
