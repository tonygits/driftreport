package mock

import (
	"context"
	"github.com/driftreport/entities"
	"github.com/driftreport/providers"
	"log"
)

type (
	MockAWSProvider struct {
	}
)

func NewAWSProvider() providers.AWSProvider {
	return &MockAWSProvider{}
}

func (s *MockAWSProvider) GetEC2Instance(ctx context.Context, instanceId string) (*entities.EC2Instance, error) {
	log.Printf("get ec2 instance by %s", instanceId)
	return &entities.EC2Instance{}, nil
}
