package providers

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/driftreport/entities"
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
		log.Printf("failed to load default config: %v", err)
		return nil, err
	}

	client := ec2.NewFromConfig(cfg)
	return &AppAWSProvider{
		awsRegion: awsRegion,
		client:    client,
	}, nil
}

func (a *AppAWSProvider) GetEC2Instance(ctx context.Context, instanceID string) (*entities.EC2Instance, error) {
	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}
	result, err := a.client.DescribeInstances(ctx, input)
	if err != nil {
		log.Printf("failed to describe instances: %v", err)
		return nil, err
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		log.Println("instance not found")
		return nil, fmt.Errorf("instance not found")
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
