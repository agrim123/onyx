package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/agrim123/onyx/pkg/ec2"
	"github.com/agrim123/onyx/pkg/iam"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/spf13/cobra"
)

var securityGroupIngressType string

var ec2Command = &cobra.Command{
	Use:   "ec2",
	Short: "Actions to be performed on EC2 namespace",
}

var ec2SgCommand = &cobra.Command{
	Use:   "sg",
	Short: "Lists, Authorizes or Revokes the security group rules",
	Long:  ``,
}

var ec2sgDescribeCommand = &cobra.Command{
	Use:     "describe <security-group-id>",
	Short:   "Describes security group",
	Long:    ``,
	Args:    cobra.MinimumNArgs(1),
	Example: "onyx ec2 sg describe sg-04ab0d31cc6fe57ca",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		ctx := context.Background()

		ec2.GetSecurityGroup(ctx, cfg, args[0], ec2.SecurityGroupRule{})

		return nil
	},
}

var ec2sgListCommand = &cobra.Command{
	Use:     "list",
	Short:   "Lists security groups by environment",
	Long:    ``,
	Args:    cobra.MinimumNArgs(1),
	Example: "onyx ec2 sg list staging",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		ctx := context.Background()

		securityGroups, err := ec2.ListSecurityGroup(ctx, cfg, strings.Title(strings.ToLower(args[0])))
		if err != nil {
			return err
		}

		fmt.Println("Security groups: (Environment: " + strings.Title(strings.ToLower(args[0])) + ")")
		for _, securityGroup := range *securityGroups {
			fmt.Println(securityGroup.ID, "(", securityGroup.Name, ")")
		}

		return nil
	},
}

var ec2sgAuthorizeCommand = &cobra.Command{
	Use:     "authorize <environment> [-t type]",
	Short:   "Authorizes security group rules",
	Long:    ``,
	Args:    cobra.MinimumNArgs(1),
	Example: "onyx ec2 sg authorize production -t ssh",
	RunE: func(cmd *cobra.Command, args []string) error {
		securityGroupUser, err := iam.Whoami()
		if err != nil {
			return errors.New("Unable to derive username. Error: " + err.Error())
		}

		if securityGroupUser == "" || len(securityGroupUser) < 3 {
			return errors.New("Invalid user")
		}

		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		ctx := context.Background()

		securityGroups, err := ec2.SelectSecurityGroups(ctx, cfg, strings.Title(strings.ToLower(args[0])))
		if err != nil {
			return err
		}

		for _, securityGroup := range *securityGroups {
			sg, err := ec2.NewSecurityGroupRule(securityGroup.ID, securityGroupIngressType, securityGroupUser)
			if err != nil {
				return err
			}

			rules := sg.GetRules(ctx, cfg)
			if len(rules) > 0 {
				if err := sg.Revoke(ctx, cfg, rules); err != nil {
					return err
				}
			}

			sg.Authorize(ctx, cfg)
		}

		return nil
	},
}

var ec2sgRevokeCommand = &cobra.Command{
	Use:     "revoke <environment> [-t type]",
	Short:   "Revokes the security group rules",
	Long:    ``,
	Args:    cobra.MinimumNArgs(1),
	Example: "onyx ec2 sg revoke staging -t redis",
	RunE: func(cmd *cobra.Command, args []string) error {
		securityGroupUser, err := iam.Whoami()
		if err != nil {
			return errors.New("Unable to derive username. Error: " + err.Error())
		}

		if securityGroupUser == "" || len(securityGroupUser) < 3 {
			return errors.New("Invalid user")
		}

		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
		if err != nil {
			log.Fatalf("unable to load SDK config, %v", err)
		}
		ctx := context.Background()

		securityGroups, err := ec2.SelectSecurityGroups(ctx, cfg, strings.Title(strings.ToLower(args[0])))
		if err != nil {
			return err
		}

		for _, securityGroup := range *securityGroups {
			sg, err := ec2.NewSecurityGroupRule(securityGroup.ID, securityGroupIngressType, securityGroupUser)
			if err != nil {
				return err
			}

			rules := sg.GetRules(ctx, cfg)
			if len(rules) > 0 {
				if err := sg.Revoke(ctx, cfg, rules); err != nil {
					return err
				}
			}
		}

		return nil
	},
}

func init() {
	ec2Command.AddCommand(ec2SgCommand)
	ec2SgCommand.AddCommand(ec2sgAuthorizeCommand, ec2sgRevokeCommand, ec2sgDescribeCommand, ec2sgListCommand)
	ec2sgAuthorizeCommand.Flags().StringVarP(&securityGroupIngressType, "type", "t", "", "Type of rule to authorize. Allowed ssh|redis|mongo (required)")
	ec2sgAuthorizeCommand.MarkFlagRequired("type")
	ec2sgRevokeCommand.Flags().StringVarP(&securityGroupIngressType, "type", "t", "", "Type of rule to authorize. Allowed ssh|redis|mongo (required)")
	ec2sgRevokeCommand.MarkFlagRequired("type")
}
