package handler

import (
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/s3"
	common "github.com/mergermarket/cdflow2-config-common"
)

// Setup handles a setup request in order to pipeline setup.
func (handler *Handler) Setup(request *common.SetupRequest, response *common.SetupResponse) error {

	if !handler.checkInputConfiguration(request.Config, request.Env, nil) {
		response.Success = false
		return nil
	}

	fmt.Fprintf(handler.errorStream, "%s\n\n", handler.styles.au.Underline("Checking AWS resources..."))

	buckets, err := listBuckets(handler.getS3Client())
	if err != nil {
		fmt.Fprintf(handler.errorStream, "%v\n\n", err)
		return nil
	}

	if err := handler.checkOrCreateReleaseBucket(buckets); err != nil {
		if success, ok := err.(Exit); ok {
			response.Success = bool(success)
			return nil
		}
		return err
	}

	if err := handler.checkOrCreateTfstateBucket(buckets); err != nil {
		if success, ok := err.(Exit); ok {
			response.Success = bool(success)
			return nil
		}
		return err
	}

	if err := handler.checkOrCreateTflocksTable(); err != nil {
		if success, ok := err.(Exit); ok {
			response.Success = bool(success)
			return nil
		}
		return err
	}

	if err := handler.checkOrCreateLambdaBucket(buckets); err != nil {
		if success, ok := err.(Exit); ok {
			response.Success = bool(success)
			return nil
		}
		return err
	}

	if err := handler.checkOrCreateECRRepository(request.Component); err != nil {
		if success, ok := err.(Exit); ok {
			response.Success = bool(success)
			return nil
		}
		return err
	}

	fmt.Fprintf(handler.errorStream, "\n")

	return nil
}

func (handler *Handler) checkOrCreateReleaseBucket(buckets []string) error {
	ok, recoverable := handler.handleReleaseBucket(buckets)
	if !ok && !recoverable {
		fmt.Fprintf(handler.errorStream, "\nUnable to resolve automatically.\n\n")
		return Exit(false)
	}
	if !ok {
		fmt.Fprintf(handler.errorStream, "\n")
		switch handler.choice("create release bucket", []string{"yes", "skip", "exit"}) {
		case "yes":
			name, err := handler.createReleaseBucket()
			if err != nil {
				return err
			}
			fmt.Fprintf(handler.errorStream, "\n  %s created release bucket: %v\n", handler.styles.tick, name)
		case "skip":
		default: // exit
			return Exit(true)
		}
	}
	return nil
}

func (handler *Handler) createReleaseBucket() (string, error) {
	name := "cdflow2-release-" + randHexPostfix()
	if _, err := handler.getS3Client().CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(name),
	}); err != nil {
		return "", err
	}
	return name, nil
}

func (handler *Handler) checkOrCreateTfstateBucket(buckets []string) error {
	ok, recoverable := handler.handleTfstateBucket(buckets)
	if !ok && !recoverable {
		fmt.Fprintf(handler.errorStream, "\nUnable to resolve automatically.\n\n")
		return Exit(false)
	}
	if !ok {
		fmt.Fprintf(handler.errorStream, "\n")
		switch handler.choice("create tfstate bucket", []string{"yes", "skip", "exit"}) {
		case "yes":
			name, err := handler.createTfstateBucket()
			if err != nil {
				return err
			}
			fmt.Fprintf(handler.errorStream, "\n  %s created tfstate bucket: %v\n", handler.styles.tick, name)
		case "skip":
		default: // exit
			return Exit(true)
		}
	}
	return nil
}

func (handler *Handler) createTfstateBucket() (string, error) {
	name := "cdflow2-tfstate-" + randHexPostfix()
	s3Client := handler.getS3Client()
	if _, err := s3Client.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(name),
	}); err != nil {
		return "", err
	}
	if _, err := s3Client.PutBucketVersioning(&s3.PutBucketVersioningInput{
		Bucket: aws.String(name),
		VersioningConfiguration: &s3.VersioningConfiguration{
			Status: aws.String("Enabled"),
		},
	}); err != nil {
		return "", err
	}
	return name, nil
}

func (handler *Handler) checkOrCreateTflocksTable() error {
	ok, err := handler.handleTflocksTable()
	if err != nil {
		return err
	}
	if !ok {
		fmt.Fprintf(handler.errorStream, "\n")
		switch handler.choice("create "+tflocksTableName+" table (optional)", []string{"yes", "skip", "exit"}) {
		case "yes":
			if err := handler.createTflocksTable(); err != nil {
				return err
			}
			fmt.Fprintf(handler.errorStream, "\n  %s created %s dynamodb table\n", handler.styles.tick, tflocksTableName)
		case "skip":
		default: // exit
			return Exit(true)
		}
	}
	return nil
}

func (handler *Handler) createTflocksTable() error {
	dynamodbClient := handler.getDynamoDBClient()
	lockIDAttribute := aws.String("LockID")
	if _, err := dynamodbClient.CreateTable(&dynamodb.CreateTableInput{
		TableName: aws.String(tflocksTableName),
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: lockIDAttribute,
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: lockIDAttribute,
				KeyType:       aws.String(dynamodb.KeyTypeHash),
			},
		},
		BillingMode: aws.String(dynamodb.BillingModePayPerRequest),
	}); err != nil {
		return err
	}
	return nil
}

func (handler *Handler) checkOrCreateLambdaBucket(buckets []string) error {
	ok, recoverable := handler.handleLambdaBucket(nil, buckets)
	if !ok && !recoverable {
		fmt.Fprintf(handler.errorStream, "\nUnable to resolve automatically.\n\n")
		return Exit(false)
	}
	if !ok {
		fmt.Fprintf(handler.errorStream, "\n")
		switch handler.choice("create lambda bucket", []string{"yes", "skip", "exit"}) {
		case "yes":
			name, err := handler.createLambdaBucket()
			if err != nil {
				return err
			}
			fmt.Fprintf(handler.errorStream, "\n  %s created lambda bucket: %v\n", handler.styles.tick, name)
		case "skip":
		default: // exit
			return Exit(true)
		}
	}
	return nil
}

func (handler *Handler) createLambdaBucket() (string, error) {
	name := "cdflow2-lambda-" + randHexPostfix()
	if _, err := handler.getS3Client().CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(name),
	}); err != nil {
		return "", err
	}
	return name, nil
}

func (handler *Handler) checkOrCreateECRRepository(component string) error {
	ok, err := handler.handleECRRepository(component, nil)
	if err != nil {
		return err
	}
	if !ok {
		fmt.Fprintf(handler.errorStream, "\n")
		switch handler.choice("create ECR registry", []string{"yes", "skip", "exit"}) {
		case "yes":
			if err := handler.createECRRepository(component); err != nil {
				return err
			}
			fmt.Fprintf(handler.errorStream, "\n  %s created ECR registry: %v\n", handler.styles.tick, component)
		case "skip":
		default: // exit
			return Exit(true)
		}
	}
	return nil
}

func (handler *Handler) createECRRepository(component string) error {
	ecrClient := handler.getECRClient()
	if _, err := ecrClient.CreateRepository(&ecr.CreateRepositoryInput{
		ImageScanningConfiguration: &ecr.ImageScanningConfiguration{
			ScanOnPush: aws.Bool(true),
		},
		ImageTagMutability: aws.String(ecr.ImageTagMutabilityImmutable),
		RepositoryName:     aws.String(component),
	}); err != nil {
		return err
	}
	if _, err := ecrClient.PutLifecyclePolicy(&ecr.PutLifecyclePolicyInput{
		LifecyclePolicyText: aws.String(`
		    {
				"rules": [
					{
						"rulePriority": 1,
						"description": "Keep most recent 100 images",
						"selection": {
							"tagStatus": "tagged",
							"tagPrefixList": ["v"],
							"countType": "imageCountMoreThan",
							"countNumber": 100
						},
						"action": {
							"type": "expire"
						}
					}
				]
			}
		`),
		RepositoryName: aws.String(component),
	}); err != nil {
		return err
	}
	return nil
}

func contains(list []string, item string) bool {
	for _, subject := range list {
		if item == subject {
			return true
		}
	}
	return false
}

func (handler *Handler) choice(prompt string, choices []string) string {
	for {
		fmt.Fprintf(handler.errorStream, "%s [%s]? ", prompt, strings.Join(choices, ", "))
		var answer string
		_, err := fmt.Fscanln(handler.inputStream, &answer)
		if err != nil {
			if err == io.EOF {
				return "exit"
			} else {
				continue
			}
		}
		if contains(choices, answer) {
			return answer
		}
	}
}
