package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"os"
	"strings"
)

func main() {
	var (
		instanceId string
		err        error
	)
	ctx := context.Background()
	profile := "go-iam"
	instanceId, err = createEC2(ctx, profile)
	if err != nil {
		fmt.Printf("createEC2 error: %s", err)
		os.Exit(1)
	}
	fmt.Printf("Instance id: %s", instanceId)
}

func createEC2(ctx context.Context, profile string) (string, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(profile))
	if err != nil {
		return "", fmt.Errorf("unable to load SDK config, %s", err)
	}
	ec2Client := ec2.NewFromConfig(cfg)

	keyPairName, err := createKeyPair("go-aws-demo", ctx, ec2Client)
	if err != nil {
		return "", err
	}

	imageId, err := describeImages(ctx, ec2Client)
	if err != nil {
		return "", err
	}

	instance, err := ec2Client.RunInstances(ctx, &ec2.RunInstancesInput{
		ImageId:      aws.String(imageId),
		KeyName:      aws.String(keyPairName),
		InstanceType: types.InstanceTypeT2Micro,
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	})

	if err != nil {
		return "", fmt.Errorf("RunInstances error: %s", err)
	}

	if len(instance.Instances) == 0 {
		return "", fmt.Errorf("instance.Instances is of 0 length")
	}

	return *instance.Instances[0].InstanceId, nil
}

// Create key pair for ec2 instance
func createKeyPair(name string, ctx context.Context, client *ec2.Client) (string, error) {
	existKeyPairs, err := client.DescribeKeyPairs(ctx, &ec2.DescribeKeyPairsInput{
		KeyNames: []string{name},
	})
	// KeyPair can exist
	if err != nil && !strings.Contains(err.Error(), "InvalidKeyPair.NotFound") {
		return "", fmt.Errorf("CreateKeyPair error: %s", err)
	}

	if existKeyPairs == nil || len(existKeyPairs.KeyPairs) == 0 {
		output, err := client.CreateKeyPair(ctx, &ec2.CreateKeyPairInput{
			KeyName: aws.String(name),
		})

		if err != nil {
			return "", fmt.Errorf("CreateKeyPair error: %s", err)
		}
		err = os.WriteFile("go-aws-ec2.pem", []byte(*output.KeyMaterial), 0600)
		if err != nil {
			return "", fmt.Errorf("WriteFile error: %s", err)
		}
		return *output.KeyName, nil
	}

	return *existKeyPairs.KeyPairs[0].KeyName, nil

}

// Get EC2 Image Information
func describeImages(ctx context.Context, client *ec2.Client) (string, error) {
	imageOutput, err := client.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("name"),
				Values: []string{"ubuntu/images/hvm-ssd/ubuntu-focal-20.04-amd64-server-*"},
			},
			{
				Name:   aws.String("virtualization-type"),
				Values: []string{"hvm"},
			},
		},
		Owners: []string{"099720109477"}, // https://ubuntu.com/server/docs/cloud-images/amazon-ec2, https://cloud-images.ubuntu.com/locator
	})

	if err != nil {
		return "", fmt.Errorf("DescribeImages error: %s", err)
	}

	if len(imageOutput.Images) == 0 {
		return "", fmt.Errorf("imageOutput.Images is of 0 length")
	}

	return *imageOutput.Images[0].ImageId, nil
}
