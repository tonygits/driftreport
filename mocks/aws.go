package mocks

import (
	"context"
	"log"

	"github.com/driftreport/entities"
	"github.com/driftreport/providers"
)

type (
	MockAWSProvider struct {
	}
)

func NewAWSProvider() providers.AWSProvider {
	return &MockAWSProvider{}
}

func (s *MockAWSProvider) GetEC2Instances(ctx context.Context, instanceIds []string) (map[string]*entities.EC2Instance, error) {
	log.Printf("get ec2 instances by %s", instanceIds)
	return map[string]*entities.EC2Instance{}, nil
}
