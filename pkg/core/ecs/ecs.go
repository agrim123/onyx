package ecs

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/agrim123/onyx/pkg/core/ec2"
	"github.com/agrim123/onyx/pkg/logger"
	"github.com/agrim123/onyx/pkg/utils"
	"github.com/aws/aws-sdk-go-v2/aws"
	ecsLib "github.com/aws/aws-sdk-go-v2/service/ecs"
)

type ContainerInstance struct {
	Arn      *string
	Instance ec2.Instance
}

func Describe(ctx context.Context, cfg aws.Config, clusterName, serviceName string) error {
	if serviceName == "" {
		logger.Warn("Service name is not provided. This results in large query, please consider narrowing your search.")
	}

	cluster := Cluster{
		Name: clusterName,
	}

	ecsHandler := ecsLib.NewFromConfig(cfg)
	// Fetch all services of the cluster
	err := cluster.GetServices(ctx, cfg, serviceName)
	if err != nil {
		return err
	}

	// Fetch tasks details of the required services
	allTasks := DescribeTasks(ctx, cfg, clusterName, &cluster.Services)

	// Filter only required container instances
	containerInstancesMap := make(map[string]*ContainerInstance)
	for _, task := range *allTasks {
		containerInstancesMap[*task.ContainerInstance.Arn] = &ContainerInstance{
			Arn: task.ContainerInstance.Arn,
		}
	}

	cluster.ContainerInstances = len(containerInstancesMap)

	containerInstancesArns := make([]string, 0)
	for containerInstanceArn := range containerInstancesMap {
		containerInstancesArns = append(containerInstancesArns, containerInstanceArn)
	}

	// Get the required container instances filteres from tasks in a cluster
	containerInstances, err := ecsHandler.DescribeContainerInstances(ctx, &ecsLib.DescribeContainerInstancesInput{
		ContainerInstances: containerInstancesArns,
		Cluster:            &clusterName,
	})

	instanceIDsMap := make(map[string]ec2.Instance)
	for _, containerInstance := range containerInstances.ContainerInstances {
		instanceIDsMap[*containerInstance.Ec2InstanceId] = ec2.Instance{
			ID: *containerInstance.Ec2InstanceId,
		}

		containerInstancesMap[*containerInstance.ContainerInstanceArn] = &ContainerInstance{
			Arn:      containerInstance.ContainerInstanceArn,
			Instance: instanceIDsMap[*containerInstance.Ec2InstanceId],
		}
	}

	instanceIDs := make([]string, 0)
	for instanceID := range instanceIDsMap {
		instanceIDs = append(instanceIDs, instanceID)
	}

	instancesDetails, err := ec2.DescribeInstances(ctx, cfg, instanceIDs)
	if err != nil {
		return err
	}

	for _, instancesDetail := range *instancesDetails {
		instanceIDsMap[instancesDetail.ID] = instancesDetail
	}

	for containerInstanceArn, containerInstance := range containerInstancesMap {
		instanceID := containerInstance.Instance.ID

		containerInstance.Instance = instanceIDsMap[instanceID]
		containerInstancesMap[containerInstanceArn] = containerInstance
	}

	for taskArn, task := range *allTasks {
		task.ContainerInstance = containerInstancesMap[*task.ContainerInstance.Arn]
		(*allTasks)[taskArn] = task
	}

	taskPerTaskDefinition := make(map[string][]Task)
	for _, task := range *allTasks {
		if tasks, ok := taskPerTaskDefinition[task.TaskDefinitionArn]; ok {
			taskPerTaskDefinition[task.TaskDefinitionArn] = append(tasks, task)
		} else {
			taskPerTaskDefinition[task.TaskDefinitionArn] = []Task{task}
		}
	}

	allServices := &cluster.Services
	for i, service := range *allServices {
		service.Tasks = taskPerTaskDefinition[service.TaskDefinitionArn]
		(*allServices)[i] = service
	}

	cluster.Services = *allServices

	fmt.Println("Cluster name:", cluster.Name)
	// fmt.Println("Registered container instances:", cluster.ContainerInstances)

	for _, service := range cluster.Services {
		fmt.Println("Service Name:", service.Name)
		// fmt.Println("  Task Definition:", service.TaskDefinitionArn)
		fmt.Println("  Tasks:")
		for _, task := range service.Tasks {
			// fmt.Println("    Arn:", *task.Arn)
			fmt.Println("    IP:", task.ContainerInstance.Instance.PrivateIPv4)
		}
	}

	return nil
}

func RedeployService(ctx context.Context, cfg aws.Config, clusterName, serviceName string) error {
	cluster := Cluster{
		Name: clusterName,
	}

	serviceMap := make(map[string]bool)
	err := cluster.GetServices(ctx, cfg, serviceName)
	if err != nil {
		return err
	}

	cluster.FilterServicesByName(serviceName)

	fmt.Println("Cluster Name:", clusterName)
	fmt.Println("Select service(s) to restart:")
	for i, service := range cluster.Services {
		fmt.Println(logger.Bold(i), ":", service.Name)
	}

	indexes := utils.GetUserInput("Enter choice: ")
	if len(indexes) == 0 {
		return errors.New("Invalid choice")
	}

	for _, index := range strings.Split(indexes, ",") {
		i, _ := strconv.ParseInt(strings.TrimSpace(index), 0, 32)
		serviceMap[cluster.Services[int(i)].Name] = true
	}

	services := make([]string, 0)
	for service := range serviceMap {
		services = append(services, service)
	}

	if len(services) == 0 {
		return fmt.Errorf("No services to restart.")
	}

	for _, service := range services {
		ecsHandler := ecsLib.NewFromConfig(cfg)
		_, err := ecsHandler.UpdateService(ctx, &ecsLib.UpdateServiceInput{
			Cluster:            aws.String(clusterName),
			Service:            aws.String(service),
			ForceNewDeployment: true,
		})

		if err != nil {
			fmt.Println("Unable to restart " + service + ". Error: " + err.Error())
		} else {
			fmt.Println("Restarted " + service)
		}
	}

	return nil
}
