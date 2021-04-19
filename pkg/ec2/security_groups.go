package ec2

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/agrim123/onyx/pkg/logger"
	"github.com/agrim123/onyx/pkg/utils"
	"github.com/aws/aws-sdk-go-v2/aws"
	ec2Lib "github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

var allowedRules = map[string]int32{
	"ssh":   22,
	"redis": 6379,
	"mongo": 27017,
	"mysql": 3306,
}

type SecurityGroup struct {
	ID   string
	Name string
}

type SecurityGroupRule struct {
	ID     string
	Type   string
	User   string
	port   int32
	DryRun bool
}

func NewSecurityGroupRule(id, typeStr, user string) (*SecurityGroupRule, error) {
	if _, ok := allowedRules[strings.ToLower(typeStr)]; !ok {
		return nil, errors.New("Invalid type. Allowed values: ssh|redis|mongo|mysql")
	}

	return &SecurityGroupRule{
		ID:   id,
		Type: strings.ToLower(typeStr),
		port: allowedRules[strings.ToLower(typeStr)],
		User: strings.ToLower(user),
	}, nil
}

func (sg *SecurityGroupRule) enrichRuleDescription() string {
	return fmt.Sprintf("[Onyx approved] (%s) User: %s", sg.Type, sg.User)
}

func (sg *SecurityGroupRule) getIpPermissions(cidrs []*string, port int32, user string) *[]types.IpPermission {
	perms := make([]types.IpPermission, 0)

	for _, cidr := range cidrs {
		perms = append(perms, types.IpPermission{
			FromPort:   port,
			IpProtocol: aws.String("tcp"),
			ToPort:     port,
			IpRanges: []types.IpRange{
				{
					CidrIp:      cidr,
					Description: aws.String(sg.enrichRuleDescription()),
				},
			},
		})
	}

	return &perms
}

func (sg *SecurityGroupRule) Describe(ctx context.Context, cfg aws.Config) {
	GetSecurityGroup(ctx, cfg, sg.ID, *sg)
}

func SelectSecurityGroups(ctx context.Context, cfg aws.Config, env string) (*[]SecurityGroup, error) {
	securityGroups, err := ListSecurityGroup(ctx, cfg, env)
	if err != nil {
		return nil, err
	}

	fmt.Println("Select security groups:")
	for i, securityGroup := range *securityGroups {
		fmt.Println(logger.Bold(i), ":", securityGroup.ID, "(", logger.Italic(securityGroup.Name), ")")
	}

	indexes := logger.InfoScan("Enter choice: ")

	if len(indexes) == 0 {
		return nil, errors.New(logger.Bold("Invalid choice"))
	}

	selectedSecurityGroups := make([]SecurityGroup, 0)
	for _, index := range strings.Split(indexes, ",") {
		i, _ := strconv.ParseInt(index, 0, 32)
		selectedSecurityGroups = append(selectedSecurityGroups, (*securityGroups)[int(i)])
	}

	return &selectedSecurityGroups, nil
}

func ListSecurityGroup(ctx context.Context, cfg aws.Config, env string) (*[]SecurityGroup, error) {
	ec2Handler := ec2Lib.NewFromConfig(cfg)
	output, err := ec2Handler.DescribeSecurityGroups(ctx, &ec2Lib.DescribeSecurityGroupsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:Environment"),
				Values: []string{env},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	securityGroups := make([]SecurityGroup, len(output.SecurityGroups))
	for i, securityGroup := range output.SecurityGroups {
		securityGroups[i] = SecurityGroup{
			ID:   *securityGroup.GroupId,
			Name: *securityGroup.GroupName,
		}
	}

	return &securityGroups, nil
}

func GetSecurityGroup(ctx context.Context, cfg aws.Config, sgID string, sgRule SecurityGroupRule) {
	sg := DescribeSecurityGroup(ctx, cfg, sgID)
	logger.Info("Current state of security group: " + logger.Bold(sgID))
	fmt.Println("|-----------------------------------------------------")
	fmt.Println(fmt.Sprintf("| %s (%s)", logger.Bold(*sg.GroupName), *sg.GroupId))
	fmt.Println("| Description:", *sg.Description)
	fmt.Println("| Rules:")
	fmt.Println("|  |------------------------------------------")
	for _, ipPermission := range sg.IpPermissions {
		for _, ipRange := range ipPermission.IpRanges {
			if *ipRange.Description == sgRule.enrichRuleDescription() {
				fmt.Println(logger.Bold(fmt.Sprintf("|  | %d -> %d (%s): %s - %s", ipPermission.FromPort, ipPermission.ToPort, *ipPermission.IpProtocol, *ipRange.CidrIp, aws.ToString(ipRange.Description))))
			} else {
				fmt.Println(fmt.Sprintf("|  | %d -> %d (%s): %s - %s", ipPermission.FromPort, ipPermission.ToPort, *ipPermission.IpProtocol, *ipRange.CidrIp, aws.ToString(ipRange.Description)))
			}
		}

		for _, a := range ipPermission.UserIdGroupPairs {
			fmt.Println(fmt.Sprintf("|  | %d -> %d (%s): %s - %v", ipPermission.FromPort, ipPermission.ToPort, *ipPermission.IpProtocol, *a.GroupId, aws.ToString(a.Description)))
		}
	}
	fmt.Println("|  |------------------------------------------")
	fmt.Println("|-----------------------------------------------------")
}

func DescribeSecurityGroup(ctx context.Context, cfg aws.Config, sgID string) *types.SecurityGroup {
	ec2Handler := ec2Lib.NewFromConfig(cfg)
	output, err := ec2Handler.DescribeSecurityGroups(ctx, &ec2Lib.DescribeSecurityGroupsInput{
		GroupIds: []string{sgID},
	})
	if err != nil {
		panic(err)
	}

	if len(output.SecurityGroups) > 0 {
		return &output.SecurityGroups[0]
	}

	return &types.SecurityGroup{}
}

func (sg *SecurityGroupRule) GetRules(ctx context.Context, cfg aws.Config) (rules []types.IpRange) {
	ec2Handler := ec2Lib.NewFromConfig(cfg)
	output, err := ec2Handler.DescribeSecurityGroups(ctx, &ec2Lib.DescribeSecurityGroupsInput{
		GroupIds: []string{sg.ID},
	})
	if err != nil {
		panic(err)
	}

	for _, ipp := range output.SecurityGroups[0].IpPermissions {
		if ipp.FromPort == sg.port {
			for _, iprange := range ipp.IpRanges {
				if *iprange.Description == sg.enrichRuleDescription() || strings.Contains(*iprange.Description, sg.User) {
					rules = append(rules, types.IpRange{
						CidrIp:      iprange.CidrIp,
						Description: iprange.Description,
					})
				}
			}
		}
	}

	return
}

func (sg *SecurityGroupRule) Authorize(ctx context.Context, cfg aws.Config) {
	ec2Handler := ec2Lib.NewFromConfig(cfg)
	_, err := ec2Handler.AuthorizeSecurityGroupIngress(ctx, &ec2Lib.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(sg.ID),
		IpPermissions: []types.IpPermission{
			{
				FromPort:   sg.port,
				IpProtocol: aws.String("tcp"),
				ToPort:     sg.port,
				IpRanges: []types.IpRange{
					{
						CidrIp:      aws.String(utils.GetPublicIP()),
						Description: aws.String(sg.enrichRuleDescription()),
					},
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	logger.Info("Authorized new rules for %s", logger.Bold(sg.ID))
	sg.Describe(ctx, cfg)
}

func (sg *SecurityGroupRule) Revoke(ctx context.Context, cfg aws.Config, rules []types.IpRange) error {
	logger.Info("Revoking old rules for %s", logger.Bold(sg.ID))

	ec2Handler := ec2Lib.NewFromConfig(cfg)
	output, err := ec2Handler.RevokeSecurityGroupIngress(ctx, &ec2Lib.RevokeSecurityGroupIngressInput{
		DryRun:  sg.DryRun,
		GroupId: aws.String(sg.ID),
		IpPermissions: []types.IpPermission{
			{
				FromPort:   sg.port,
				IpProtocol: aws.String("tcp"),
				ToPort:     sg.port,
				IpRanges:   rules,
			},
		},
	})
	if err != nil {
		return err
	}

	if output.Return {
		logger.Info("Revoked old rules for %s", logger.Bold(sg.ID))
		sg.Describe(ctx, cfg)
		return nil
	}

	sg.Describe(ctx, cfg)
	return errors.New("Unable to revoke old rules")
}
