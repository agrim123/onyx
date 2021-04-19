package ecs

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/agrim123/onyx/pkg/ec2"
	"github.com/agrim123/onyx/pkg/logger"
	"github.com/aws/aws-sdk-go-v2/aws"
	ecsLib "github.com/aws/aws-sdk-go-v2/service/ecs"
)

type Cluster struct {
	Arn                *string
	Name               string
	Services           []Service
	ContainerInstances int
}

type ContainerInstance struct {
	Arn      *string
	Instance ec2.Instance
}

type Service struct {
	Arn               *string
	Name              string
	TaskDefinitionArn string
	Tasks             []Task
}

type Task struct {
	Arn               *string
	TaskDefinitionArn string
	ContainerInstance *ContainerInstance
	Service           *Service
}

func Describe(ctx context.Context, cfg aws.Config, clusterName, serviceName string) {
	if serviceName == "" {
		logger.Info("Service name is not provided. This results in large query, please consider narrowing your search.")
	}

	cluster := Cluster{
		Name: clusterName,
	}

	ecsHandler := ecsLib.NewFromConfig(cfg)

	// Get all services of the cluster
	allServicesOutput, err := ecsHandler.ListServices(ctx, &ecsLib.ListServicesInput{
		Cluster: &clusterName,
	})
	if err != nil {
		panic(err)
	}

	requiredServiceArns := make([]string, 0)
	if serviceName != "" {
		for _, serviceArn := range allServicesOutput.ServiceArns {
			if strings.Contains(serviceArn, serviceName) {
				requiredServiceArns = append(requiredServiceArns, serviceArn)
			}
		}
	} else {
		requiredServiceArns = allServicesOutput.ServiceArns
	}

	// describe all services fetched above
	servicesOutput, err := ecsHandler.DescribeServices(ctx, &ecsLib.DescribeServicesInput{
		Cluster:  &clusterName,
		Services: requiredServiceArns,
	})
	if err != nil {
		panic(err)
	}

	// filter required services
	allServices := make([]Service, 0)
	for _, service := range servicesOutput.Services {
		allServices = append(allServices, Service{
			Arn:               service.ServiceArn,
			Name:              *service.ServiceName,
			TaskDefinitionArn: *service.TaskDefinition,
		})
	}

	// Get all tasks of cluster or filtered if service name is provided
	var allTasksOutput *ecsLib.ListTasksOutput
	if serviceName != "" {
		allTasksOutput, err = ecsHandler.ListTasks(ctx, &ecsLib.ListTasksInput{
			Cluster:     &clusterName,
			ServiceName: aws.String(serviceName),
		})
	} else {
		allTasksOutput, err = ecsHandler.ListTasks(ctx, &ecsLib.ListTasksInput{
			Cluster: &clusterName,
		})
	}

	// fetch required tasks details
	detailedTasks, err := ecsHandler.DescribeTasks(ctx, &ecsLib.DescribeTasksInput{
		Cluster: &clusterName,
		Tasks:   allTasksOutput.TaskArns,
	})

	allTasks := make(map[string]Task)
	for _, task := range detailedTasks.Tasks {
		allTasks[*task.TaskArn] = Task{
			Arn:               task.TaskArn,
			TaskDefinitionArn: *task.TaskDefinitionArn,
			ContainerInstance: &ContainerInstance{
				Arn: task.ContainerInstanceArn,
			},
			Service: &Service{},
		}
	}

	containerInstancesMap := make(map[string]*ContainerInstance)
	for _, task := range detailedTasks.Tasks {
		containerInstancesMap[*task.ContainerInstanceArn] = &ContainerInstance{
			Arn: task.ContainerInstanceArn,
		}
	}

	cluster.ContainerInstances = len(containerInstancesMap)

	containerInstancesArns := make([]string, 0)
	for containerInstanceArn := range containerInstancesMap {
		containerInstancesArns = append(containerInstancesArns, containerInstanceArn)
	}

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

	instancesDetails := ec2.DescribeInstances(ctx, cfg, instanceIDs)

	for _, instancesDetail := range *instancesDetails {
		instanceIDsMap[instancesDetail.ID] = instancesDetail
	}

	for containerInstanceArn, containerInstance := range containerInstancesMap {
		instanceID := containerInstance.Instance.ID

		containerInstance.Instance = instanceIDsMap[instanceID]
		containerInstancesMap[containerInstanceArn] = containerInstance
	}

	for taskArn, task := range allTasks {
		task.ContainerInstance = containerInstancesMap[*task.ContainerInstance.Arn]
		allTasks[taskArn] = task
	}

	taskPerTaskDefinition := make(map[string][]Task)
	for _, task := range allTasks {
		if tasks, ok := taskPerTaskDefinition[task.TaskDefinitionArn]; ok {
			taskPerTaskDefinition[task.TaskDefinitionArn] = append(tasks, task)
		} else {
			taskPerTaskDefinition[task.TaskDefinitionArn] = []Task{task}
		}
	}

	for i, service := range allServices {
		service.Tasks = taskPerTaskDefinition[service.TaskDefinitionArn]
		allServices[i] = service
	}

	cluster.Services = allServices

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
}

func RedeployService(ctx context.Context, cfg aws.Config, clusterName, serviceName string) error {
	services := make([]string, 0)

	if serviceName == "" {
		services = GetClusterServices(ctx, cfg, clusterName)

		fmt.Println("Cluster Name:", clusterName)
		fmt.Println("Select service(s) to restart:")
		for i, service := range services {
			fmt.Println(i, ":", service)
		}

		var indexes string

		fmt.Print("Enter choice: ")
		fmt.Scanf("%s", &indexes)

		if len(indexes) == 0 {
			return errors.New("Invalid choice")
		}

		servicesToRestart := make([]string, 0)
		for _, index := range strings.Split(indexes, ",") {
			i, _ := strconv.ParseInt(index, 0, 32)
			servicesToRestart = append(servicesToRestart, services[int(i)])
		}

		services = servicesToRestart
	} else {
		services = append(services, serviceName)
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
