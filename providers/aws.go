package providers

import (
	"context"
	"errors"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/driftreport/entities"
	"github.com/driftreport/utils"
)

type (
	AWSProvider interface {
		GetEC2Instance(ctx context.Context, instanceID string) (*entities.EC2Instance, error)
	}

	AppAWSProvider struct {
		awsRegion string
		client    *ec2.Client
	}
)

func NewAWSProvider() (AWSProvider, error) {
	return NewAWSProviderWithCredentials(
		os.Getenv("AWS_REGION"),
	)
}

func NewAWSProviderWithCredentials(
	awsRegion string,
) (AWSProvider, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(os.Getenv("AWS_REGION")))
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

//GetEC2Instance get EC2 instance from AWS account
func (a *AppAWSProvider) GetEC2Instance(ctx context.Context, instanceID string) (*entities.EC2Instance, error) {
	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
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

	instance := result.Reservations[0].Instances[0]
	var securityGroups []string
	for _, sg := range instance.SecurityGroups {
		securityGroups = append(securityGroups, *sg.GroupId)
	}

	tags := make(map[string]string)
	for _, tag := range instance.Tags {
		tags[*tag.Key] = *tag.Value
	}

	return &entities.EC2Instance{
		InstanceType:   string(instance.InstanceType),
		SecurityGroups: securityGroups,
		Tags:           tags,
	}, nil
}
