package ecs

import (
	"context"
	"strings"

	"github.com/agrim123/onyx/pkg/utils"
	"github.com/aws/aws-sdk-go-v2/aws"
	ecsLib "github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

type Cluster struct {
	Arn                *string
	Name               string
	Services           []Service
	ContainerInstances int
}

func (c *Cluster) GetServices(ctx context.Context, cfg aws.Config, serviceName string) error {
	ecsHandler := ecsLib.NewFromConfig(cfg)
	allServices := make([]Service, 0)

	allServicesArns := make([]string, 0)

	var nextToken *string
	for {
		allServicesOutput, _ := ecsHandler.ListServices(ctx, &ecsLib.ListServicesInput{
			Cluster:            aws.String(c.Name),
			NextToken:          nextToken,
			SchedulingStrategy: types.SchedulingStrategyReplica,
		})

		allServicesArns = append(allServicesArns, allServicesOutput.ServiceArns...)

		if allServicesOutput.NextToken == nil {
			break
		}

		nextToken = allServicesOutput.NextToken
	}

	requiredServiceArns := make([]string, 0)
	if serviceName != "" {
		for _, serviceArn := range allServicesArns {
			if strings.Contains(serviceArn, serviceName) {
				requiredServiceArns = append(requiredServiceArns, serviceArn)
			}
		}
	} else {
		requiredServiceArns = allServicesArns
	}

	servicesFromAWS := make([]types.Service, 0)
	for _, chunk := range utils.GetChunks(requiredServiceArns, 9) {
		servicesOutput, err := ecsHandler.DescribeServices(ctx, &ecsLib.DescribeServicesInput{
			Cluster:  aws.String(c.Name),
			Services: chunk,
		})
		if err == nil {
			servicesFromAWS = append(servicesFromAWS, servicesOutput.Services...)
		}
	}

	for _, service := range servicesFromAWS {
		allServices = append(allServices, Service{
			Arn:               service.ServiceArn,
			Name:              *service.ServiceName,
			TaskDefinitionArn: *service.TaskDefinition,
		})
	}

	c.Services = allServices

	return nil
}

func (c *Cluster) FilterServicesByName(serviceName string) {
	if serviceName != "" {
		filteredServices := make([]Service, 0)
		for _, service := range c.Services {
			if strings.Contains(service.Name, serviceName) {
				filteredServices = append(filteredServices, service)
			}
		}

		c.Services = filteredServices
	}
}
