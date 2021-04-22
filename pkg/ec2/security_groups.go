package ec2

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/agrim123/onyx/pkg/iam"
	"github.com/agrim123/onyx/pkg/logger"
	"github.com/agrim123/onyx/pkg/utils"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
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
	ID          string
	Name        string
	Description string
	Tags        []types.Tag
	rules       []SecurityGroupRule
}

type SecurityGroupRule struct {
	user        string
	port        int32
	cidr        string
	protocol    string
	description string
}

func NewSecurityGroup(ctx context.Context, cfg aws.Config, id string) (*SecurityGroup, error) {
	securityGroup := SecurityGroup{
		ID:    id,
		rules: make([]SecurityGroupRule, 0),
	}

	ec2Handler := ec2Lib.NewFromConfig(cfg)
	output, err := ec2Handler.DescribeSecurityGroups(ctx, &ec2Lib.DescribeSecurityGroupsInput{
		GroupIds: []string{id},
	})
	if err != nil {
		return &securityGroup, err
	}

	securityGroups := convertSecurityGroups(&output.SecurityGroups)

	return &(securityGroups[0]), nil
}

func NewSecurityGroupRule(port int32, user string) (*SecurityGroupRule, error) {
	return &SecurityGroupRule{
		port: port,
		user: strings.ToLower(user),
	}, nil
}

// DisplaySecurityGroup prints the security group details
func (sg *SecurityGroup) DisplaySecurityGroup(ctx context.Context, cfg aws.Config, changedRules *[]SecurityGroupRule, toRemove bool) {
	changedRulesMap := make(map[string]bool)
	if changedRules != nil {
		for _, rule := range *changedRules {
			changedRulesMap[fmt.Sprintf("%s_%d", rule.description, rule.port)] = true
		}
	}

	if toRemove {
		logger.Warn("Proposed changes to security group: " + logger.Bold(sg.ID))
	} else {
		logger.Info("Current state of security group: " + logger.Bold(sg.ID))
	}

	fmt.Println("|-----------------------------------------------------")
	fmt.Println(fmt.Sprintf("| %s (%s)", logger.Bold(sg.Name), sg.ID))
	fmt.Println("| Description:", sg.Description)
	fmt.Println("| Rules:")
	fmt.Println("|  |------------------------------------------")
	for _, rule := range sg.rules {
		if _, ok := changedRulesMap[fmt.Sprintf("%s_%d", rule.description, rule.port)]; ok {
			if toRemove {
				fmt.Print(logger.Red(fmt.Sprintf("|  | %d -> %d (%s): %s - %s", rule.port, rule.port, rule.protocol, rule.cidr, rule.description)))
				fmt.Println(logger.Bold("    <------- This rule will be removed/updated"))
			} else {
				fmt.Println(logger.Green(fmt.Sprintf("|  | %d -> %d (%s): %s - %s", rule.port, rule.port, rule.protocol, rule.cidr, rule.description)))
			}
		} else {
			fmt.Println(fmt.Sprintf("|  | %d -> %d (%s): %s - %s", rule.port, rule.port, rule.protocol, rule.cidr, rule.description))
		}

		// for _, a := range ipPermission.UserIdGroupPairs {
		// 	fmt.Println(fmt.Sprintf("|  | %d -> %d (%s): %s - %v", ipPermission.FromPort, ipPermission.ToPort, *ipPermission.IpProtocol, *a.GroupId, aws.ToString(a.Description)))
		// }
	}
	fmt.Println("|  |------------------------------------------")
	fmt.Println("|-----------------------------------------------------")
}

func (sgRule *SecurityGroupRule) enrichRuleDescription() string {
	return fmt.Sprintf("[Onyx approved] User: %s", sgRule.user)
}

func (sgRule *SecurityGroupRule) attachNewIP(ip string) {
	sgRule.cidr = ip
}

func (sg *SecurityGroup) FilterIngressRules(securityGroupRules *[]SecurityGroupRule) (filteredSecurityGroupRules []SecurityGroupRule) {
	for _, sgRule := range sg.rules {
		for _, securityGroupRule := range *securityGroupRules {
			if sgRule.port == securityGroupRule.port && (strings.Contains(sgRule.description, securityGroupRule.user) || sgRule.description == securityGroupRule.enrichRuleDescription()) {
				filteredSecurityGroupRules = append(filteredSecurityGroupRules, sgRule)
			}
		}
	}
	return
}

func (sg *SecurityGroup) Authorize(ctx context.Context, cfg aws.Config, rules *[]SecurityGroupRule) {
	logger.Info("Authorizing new rules for %s", logger.Bold(sg.ID))
	ec2Handler := ec2Lib.NewFromConfig(cfg)

	publicIP := utils.GetPublicIP()

	ipPermissions := make([]types.IpPermission, 0)
	for i, rule := range *rules {
		// Update rule ip and description
		rule.description = rule.enrichRuleDescription()
		rule.attachNewIP(publicIP)

		ipPermissions = append(ipPermissions, types.IpPermission{
			FromPort:   rule.port,
			IpProtocol: aws.String("tcp"),
			ToPort:     rule.port,
			IpRanges: []types.IpRange{
				{
					CidrIp:      &rule.cidr,
					Description: &rule.description,
				},
			},
		})

		(*rules)[i] = rule
	}

	_, err := ec2Handler.AuthorizeSecurityGroupIngress(ctx, &ec2Lib.AuthorizeSecurityGroupIngressInput{
		GroupId:       aws.String(sg.ID),
		IpPermissions: ipPermissions,
	})
	if err != nil {
		panic(err)
	}

	logger.Success("Authorized new rules for %s", logger.Bold(sg.ID))
	// Force refresh
	sg, _ = NewSecurityGroup(ctx, cfg, sg.ID)
	sg.DisplaySecurityGroup(ctx, cfg, rules, false)
}

func (sg *SecurityGroup) Revoke(ctx context.Context, cfg aws.Config, rules []SecurityGroupRule) error {
	sg.DisplaySecurityGroup(ctx, cfg, &rules, true)

	logger.Info("Revoking old rules for %s", logger.Bold(sg.ID))

	ipPermissions := make([]types.IpPermission, 0)
	for _, rule := range rules {
		ipPermissions = append(ipPermissions, types.IpPermission{
			FromPort:   rule.port,
			IpProtocol: aws.String("tcp"),
			ToPort:     rule.port,
			IpRanges: []types.IpRange{
				{
					CidrIp:      &rule.cidr,
					Description: &rule.description,
				},
			},
		})
	}

	ec2Handler := ec2Lib.NewFromConfig(cfg)
	output, err := ec2Handler.RevokeSecurityGroupIngress(ctx, &ec2Lib.RevokeSecurityGroupIngressInput{
		GroupId:       aws.String(sg.ID),
		IpPermissions: ipPermissions,
	})
	if err != nil {
		return err
	}

	// Force refresh
	sg, _ = NewSecurityGroup(ctx, cfg, sg.ID)

	if output.Return {
		logger.Success("Revoked old rules for %s", logger.Bold(sg.ID))
		sg.DisplaySecurityGroup(ctx, cfg, nil, false)
		return nil
	}

	sg.DisplaySecurityGroup(ctx, cfg, nil, false)
	return errors.New("Unable to revoke old rules")
}

func SelectSecurityGroups(ctx context.Context, cfg aws.Config, env string) ([]SecurityGroup, error) {
	securityGroups, err := ListSecurityGroupsByEnv(ctx, cfg, env)
	if err != nil {
		return nil, err
	}

	logger.Info("Select security groups:")
	for i, securityGroup := range securityGroups {
		fmt.Println(logger.Bold(i), ":", securityGroup.ID, "(", logger.Italic(securityGroup.Name), ")")
	}

	indexes := utils.GetUserInput("Enter Choice: ")

	if len(indexes) == 0 {
		return nil, errors.New(logger.Bold("Invalid choice"))
	}

	selectedSecurityGroups := make([]SecurityGroup, 0)
	for _, index := range strings.Split(indexes, ",") {
		i, _ := strconv.ParseInt(strings.TrimSpace(index), 0, 32)
		selectedSecurityGroups = append(selectedSecurityGroups, securityGroups[int(i)])
	}

	return selectedSecurityGroups, nil
}

// ListSecurityGroupsByEnv returns all security groups filtered by Tag:Environment if provided
func ListSecurityGroupsByEnv(ctx context.Context, cfg aws.Config, env string) ([]SecurityGroup, error) {
	ec2Handler := ec2Lib.NewFromConfig(cfg)
	filters := make([]types.Filter, 0)

	if env != "" {
		filters = append(filters, types.Filter{
			Name:   aws.String("tag:Environment"),
			Values: []string{strings.Title(strings.ToLower(env))},
		})
	} else {
		logger.Warn("Please use `--env` to narrow down search.")
	}

	output, err := ec2Handler.DescribeSecurityGroups(ctx, &ec2Lib.DescribeSecurityGroupsInput{
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	return convertSecurityGroups(&output.SecurityGroups), nil
}

func convertSecurityGroups(libSecurityGroups *[]types.SecurityGroup) (securityGroups []SecurityGroup) {
	for _, securityGroup := range *libSecurityGroups {
		rules := make([]SecurityGroupRule, 0)
		for _, ipp := range securityGroup.IpPermissions {
			for _, iprange := range ipp.IpRanges {
				rules = append(rules, SecurityGroupRule{
					cidr:        *iprange.CidrIp,
					description: *iprange.Description,
					port:        ipp.FromPort,
					protocol:    aws.ToString(ipp.IpProtocol),
				})
			}

			for _, group := range ipp.UserIdGroupPairs {
				rules = append(rules, SecurityGroupRule{
					cidr:        *group.GroupId,
					description: aws.ToString(group.Description),
					port:        ipp.FromPort,
					protocol:    aws.ToString(ipp.IpProtocol),
				})
			}
		}

		securityGroups = append(securityGroups, SecurityGroup{
			ID:          *securityGroup.GroupId,
			Name:        aws.ToString(securityGroup.GroupName),
			Tags:        securityGroup.Tags,
			Description: aws.ToString(securityGroup.Description),
			rules:       rules,
		})
	}

	return
}

func AuthorizeOrRevokeRule(envOrID string, types []string, ports []int32, authorize bool) error {
	portsToUpdate := make([]int32, 0)
	for _, t := range types {
		if value, ok := allowedRules[strings.ToLower(t)]; !ok {
			return errors.New("Invalid type. Allowed values: ssh|redis|mongo|mysql")
		} else {
			portsToUpdate = append(portsToUpdate, value)
		}
	}

	for _, port := range ports {
		portsToUpdate = append(portsToUpdate, port)
	}

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

	var securityGroups []SecurityGroup
	if strings.HasPrefix(envOrID, "sg-") {
		securityGroup, err := NewSecurityGroup(ctx, cfg, envOrID)
		if err != nil {
			return errors.New("Invalid security group id. Error: " + err.Error())
		}

		logger.Success("Detected security group: %s (%s)", logger.Bold(securityGroup.ID), logger.Italic(securityGroup.Name))

		securityGroups = append(securityGroups, *securityGroup)
	} else {
		selectedSecurityGroups, err := SelectSecurityGroups(ctx, cfg, strings.Title(strings.ToLower(envOrID)))
		if err != nil {
			return err
		}

		securityGroups = append(securityGroups, selectedSecurityGroups...)
	}

	sgRules := make([]SecurityGroupRule, 0)
	for _, port := range portsToUpdate {
		sgRule, _ := NewSecurityGroupRule(port, securityGroupUser)
		sgRules = append(sgRules, *sgRule)
	}

	for _, securityGroup := range securityGroups {
		rules := securityGroup.FilterIngressRules(&sgRules)
		if len(rules) > 0 {
			if err := securityGroup.Revoke(ctx, cfg, rules); err != nil {
				return err
			}
		}

		if authorize {
			securityGroup.Authorize(ctx, cfg, &sgRules)
		}
	}

	return nil
}
