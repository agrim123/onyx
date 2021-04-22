package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/agrim123/onyx/pkg/core/ec2"
	"github.com/agrim123/onyx/pkg/logger"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/spf13/cobra"
)

var securityGroupIngressTypes string
var securityGroupIngressPorts string
var securityGroupEnv string
var securityGroupID string

var ec2Command = &cobra.Command{
	Use:   "ec2",
	Short: "Actions to be performed on EC2 namespace",
}

var ec2SgCommand = &cobra.Command{
	Use:   "sg",
	Short: "Lists, Authorizes or Revokes the security group rules",
}

var ec2sgDescribeCommand = &cobra.Command{
	Use:     "describe {--env environment | --id sg-id}",
	Short:   "Describes security group based on id or filtered by environment. Supports only one group at a time.",
	Args:    cobra.NoArgs,
	Example: "onyx ec2 sg describe --env staging\nonyx ec2 sg describe --id sg-12VJGkhd28iv11",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		ctx := context.Background()

		if securityGroupEnv != "" {
			sg, err := ec2.SelectSecurityGroups(ctx, cfg, securityGroupEnv)
			if err != nil {
				return err
			}
			if len(sg) > 0 {
				(sg)[0].DisplaySecurityGroup(ctx, cfg, nil, false)
			}
			return nil
		}

		if securityGroupID != "" {
			sg, err := ec2.NewSecurityGroup(ctx, cfg, securityGroupID)
			if err != nil {
				return err
			}

			sg.DisplaySecurityGroup(ctx, cfg, nil, false)
			return nil
		}

		return errors.New("Either `--id` or `--env` is required")
	},
}

var ec2sgListCommand = &cobra.Command{
	Use:     "list [--env <environment>]",
	Short:   "Lists all security groups",
	Args:    cobra.NoArgs,
	Example: "onyx ec2 sg list\nonyx ec2 sg list --env staging",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		ctx := context.Background()

		env := strings.Title(strings.ToLower(securityGroupEnv))

		securityGroups, err := ec2.ListSecurityGroupsByEnv(ctx, cfg, env)
		if err != nil {
			return err
		}

		if env != "" {
			fmt.Println("Security groups: (Environment: " + logger.Bold(env) + ")")
		}
		for _, securityGroup := range securityGroups {
			fmt.Println(securityGroup.ID, "(", logger.Italic(securityGroup.Name), ")")
		}

		return nil
	},
}

var ec2sgAuthorizeCommand = &cobra.Command{
	Use:     "authorize [environment | security-group-id] {[--types types] | [--ports ports]}",
	Short:   "Authorizes security group rules",
	Long:    ``,
	Args:    cobra.MinimumNArgs(1),
	Example: "onyx ec2 sg authorize production -t ssh\nonyx ec2 sg authorize staging -t ssh,mongo,redis\nonyx ec2 sg authorize sg-ajvjTUf581ig1 -t ssh,mongo,redis",
	RunE: func(cmd *cobra.Command, args []string) error {
		var ports []int32
		if securityGroupIngressPorts != "" {
			portsStr := strings.Split(securityGroupIngressPorts, ",")
			for _, port := range portsStr {
				portInt64, _ := strconv.ParseInt(strings.TrimSpace(port), 0, 64)
				ports = append(ports, int32(portInt64))
			}
		}

		var types []string
		if securityGroupIngressTypes != "" {
			for _, t := range strings.Split(securityGroupIngressTypes, ",") {
				types = append(types, strings.TrimSpace(t))
			}
		}

		return ec2.AuthorizeOrRevokeRule(args[0], types, ports, true)
	},
}

var ec2sgRevokeCommand = &cobra.Command{
	Use:     "revoke [environment | security-group-id] {[--types types] | [--ports ports]}",
	Short:   "Revokes the security group rules",
	Long:    ``,
	Args:    cobra.MinimumNArgs(1),
	Example: "onyx ec2 sg revoke staging -t redis",
	RunE: func(cmd *cobra.Command, args []string) error {
		var ports []int32
		if securityGroupIngressPorts != "" {
			portsStr := strings.Split(securityGroupIngressPorts, ",")
			for _, port := range portsStr {
				portInt64, _ := strconv.ParseInt(strings.TrimSpace(port), 0, 64)
				ports = append(ports, int32(portInt64))
			}
		}

		var types []string
		if securityGroupIngressTypes != "" {
			for _, t := range strings.Split(securityGroupIngressTypes, ",") {
				types = append(types, strings.TrimSpace(t))
			}
		}

		return ec2.AuthorizeOrRevokeRule(args[0], types, ports, false)
	},
}

func init() {
	ec2Command.AddCommand(ec2SgCommand)

	ec2SgCommand.AddCommand(ec2sgAuthorizeCommand, ec2sgRevokeCommand, ec2sgDescribeCommand, ec2sgListCommand)

	ec2sgListCommand.Flags().StringVarP(&securityGroupEnv, "env", "e", "", "Environment for which to list. Allowed values production|staging")

	ec2sgDescribeCommand.Flags().StringVarP(&securityGroupEnv, "env", "e", "", "Environment for which to describe. Allowed values production|staging")
	ec2sgDescribeCommand.Flags().StringVarP(&securityGroupID, "id", "i", "", "Security group ID to describe")

	ec2sgAuthorizeCommand.Flags().StringVarP(&securityGroupIngressTypes, "types", "t", "", "Types of rule to authorize. Allowed ssh|redis|mongo|mysql (required)")
	ec2sgAuthorizeCommand.Flags().StringVarP(&securityGroupIngressPorts, "ports", "p", "", "Ports to authorize. Allowed values 0-65536")
	ec2sgAuthorizeCommand.Flags().StringVarP(&securityGroupID, "id", "i", "", "Security group ID to change")

	ec2sgRevokeCommand.Flags().StringVarP(&securityGroupIngressTypes, "types", "t", "", "Types of rule to authorize. Allowed ssh|redis|mongo|mysql (required)")
	ec2sgRevokeCommand.Flags().StringVarP(&securityGroupIngressPorts, "ports", "p", "", "Ports to authorize. Allowed values 0-65536")
	ec2sgRevokeCommand.Flags().StringVarP(&securityGroupID, "id", "i", "", "Security group ID to change")
}
