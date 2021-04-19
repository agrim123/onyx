package cmd

import (
	"context"
	"log"

	"github.com/agrim123/onyx/pkg/ecs"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/spf13/cobra"
)

var ecsCommand = &cobra.Command{
	Use:       "ecs",
	Short:     "Actions to be performed on ECS clusters",
	ValidArgs: []string{"describe"},
}

var ecsDescribeCommand = &cobra.Command{
	Use:     "describe",
	Short:   "Describes the given ECS cluster and service if name is provided",
	Long:    ``,
	Args:    cobra.MinimumNArgs(1),
	Example: "onyx ecs describe staging-api-cluster \nonyx ecs describe staging-api-cluster some-service",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		ctx := context.Background()

		var clusterName string
		var serviceName string

		if len(args) > 0 {
			clusterName = args[0]
		}

		if len(args) > 1 {
			serviceName = args[1]
		}

		ecs.Describe(ctx, cfg, clusterName, serviceName)
	},
}

var ecsClusterName string
var ecsServiceName string

var ecsRestartServiceCommand = &cobra.Command{
	Use:     "restart",
	Short:   "Forces new deployment of provided ECS services",
	Long:    ``,
	Example: "onyx ecs restart --cluster staging-api-cluster\nonyx ecs restart --cluster staging-api-cluster --service backtest_services",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		ctx := context.Background()
		return ecs.RedeployService(ctx, cfg, ecsClusterName, ecsServiceName)
	},
}

func init() {
	ecsCommand.AddCommand(ecsDescribeCommand, ecsRestartServiceCommand)
	ecsRestartServiceCommand.Flags().StringVarP(&ecsClusterName, "cluster", "c", "", "Cluster Name (required)")
	ecsRestartServiceCommand.MarkFlagRequired("cluster")
	ecsRestartServiceCommand.Flags().StringVarP(&ecsServiceName, "service", "s", "", "Service Name")
}
