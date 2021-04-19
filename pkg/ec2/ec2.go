package ec2

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2Lib "github.com/aws/aws-sdk-go-v2/service/ec2"
)

type Instance struct {
	ID          string
	PublicIPv4  string
	PrivateIPv4 string
}

func DescribeInstances(ctx context.Context, cfg aws.Config, instanceIDs []string) *[]Instance {
	ec2Handler := ec2Lib.NewFromConfig(cfg)

	ec2DetailsOutput, err := ec2Handler.DescribeInstances(ctx, &ec2Lib.DescribeInstancesInput{
		InstanceIds: instanceIDs,
	})
	if err != nil {
		panic(err)
	}

	instances := make([]Instance, 0)
	for _, reservation := range ec2DetailsOutput.Reservations {
		publicIpv4 := ""
		if reservation.Instances[0].PublicIpAddress != nil {
			publicIpv4 = *reservation.Instances[0].PublicIpAddress
		}

		instances = append(instances, Instance{
			ID:          *reservation.Instances[0].InstanceId,
			PrivateIPv4: *reservation.Instances[0].PrivateIpAddress,
			PublicIPv4:  publicIpv4,
		})
	}

	return &instances
}
