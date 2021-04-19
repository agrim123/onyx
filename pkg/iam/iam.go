package iam

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
)

func Whoami() (string, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-1"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	ctx := context.Background()

	iamHandler := iam.NewFromConfig(cfg)
	output, err := iamHandler.GetUser(ctx, &iam.GetUserInput{})
	if err != nil {
		return "", err
	}

	return *output.User.UserName, nil
}
