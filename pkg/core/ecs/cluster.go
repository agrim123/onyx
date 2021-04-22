package ecs

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	ecsLib "github.com/aws/aws-sdk-go-v2/service/ecs"
)

func GetClusterServices(ctx context.Context, cfg aws.Config, clusterName string) (serviceArns []string) {
	ecsHandler := ecsLib.NewFromConfig(cfg)

	var nextToken *string
	for {
		// Get all services of the cluster
		allServicesOutput, err := ecsHandler.ListServices(ctx, &ecsLib.ListServicesInput{
			Cluster:   &clusterName,
			NextToken: nextToken,
		})
		if err != nil {
			fmt.Println("Unable to describe cluster", err)
		}

		serviceArns = append(serviceArns, allServicesOutput.ServiceArns...)

		if allServicesOutput.NextToken == nil {
			break
		}

		nextToken = allServicesOutput.NextToken
	}

	return extractServiceNameFromServiceArns(clusterName, serviceArns)
}
