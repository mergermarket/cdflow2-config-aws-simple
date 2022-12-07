package handler

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/s3"
	common "github.com/mergermarket/cdflow2-config-common"
)

// Setup handles a setup request in order to pipeline setup.
func (h *Handler) Setup(request *common.SetupRequest, response *common.SetupResponse) error {
	if !h.CheckInputConfiguration(request.Config, request.Env) {
		response.Success = false
		return nil
	}

	response.Monitoring.APIKey = h.getDatadogAPIKey()

	team, err := h.getTeam(request.Config["team"])
	if err == nil {
		response.Monitoring.Data["team"] = team
	}

	fmt.Fprintf(h.ErrorStream, "%s\n\n", h.styles.au.Underline("Checking AWS resources..."))

	buckets, err := listBuckets(h.getS3Client())
	if err != nil {
		fmt.Fprintf(h.ErrorStream, "%v\n\n", err)
		return nil
	}

	if err := h.checkOrCreateReleaseBucket(buckets); err != nil {
		if success, ok := err.(Exit); ok {
			response.Success = bool(success)
			return nil
		}
		return err
	}

	if err := h.checkOrCreateTfstateBucket(buckets); err != nil {
		if success, ok := err.(Exit); ok {
			response.Success = bool(success)
			return nil
		}
		return err
	}

	if err := h.checkOrCreateTflocksTable(); err != nil {
		if success, ok := err.(Exit); ok {
			response.Success = bool(success)
			return nil
		}
		return err
	}

	if h.requiresLambdaBucket(request.ReleaseRequirements) {
		if err := h.checkOrCreateLambdaBucket(buckets); err != nil {
			if success, ok := err.(Exit); ok {
				response.Success = bool(success)
				return nil
			}
			return err
		}
	}

	if err := h.checkOrCreateECRRepository(request.Component); err != nil {
		if success, ok := err.(Exit); ok {
			response.Success = bool(success)
			return nil
		}
		return err
	}

	fmt.Fprintf(h.ErrorStream, "\n")

	return nil
}

func (h *Handler) checkOrCreateReleaseBucket(buckets []string) error {
	ok, recoverable := h.handleReleaseBucket(buckets)
	if !ok && !recoverable {
		fmt.Fprintf(h.ErrorStream, "\nUnable to resolve automatically.\n\n")
		return Exit(false)
	}
	if !ok {
		fmt.Fprintf(h.ErrorStream, "\n")

		name, err := h.createReleaseBucket()
		if err != nil {
			return err
		}
		fmt.Fprintf(h.ErrorStream, "\n  %s created release bucket: %v\n", h.styles.tick, name)

	}
	return nil
}

func (h *Handler) createReleaseBucket() (string, error) {
	name := "cdflow2-release-" + randHexPostfix()
	if _, err := h.getS3Client().CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(name),
	}); err != nil {
		return "", err
	}
	return name, nil
}

func (h *Handler) checkOrCreateTfstateBucket(buckets []string) error {
	ok, recoverable := h.handleTfstateBucket(buckets)
	if !ok && !recoverable {
		fmt.Fprintf(h.ErrorStream, "\nUnable to resolve automatically.\n\n")
		return Exit(false)
	}
	if !ok {
		fmt.Fprintf(h.ErrorStream, "\n")

		name, err := h.createTfstateBucket()
		if err != nil {
			return err
		}
		fmt.Fprintf(h.ErrorStream, "\n  %s created tfstate bucket: %v\n", h.styles.tick, name)

	}
	return nil
}

func (h *Handler) createTfstateBucket() (string, error) {
	name := "cdflow2-tfstate-" + randHexPostfix()
	s3Client := h.getS3Client()
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

func (h *Handler) checkOrCreateTflocksTable() error {
	ok := h.handleTflocksTable()
	if !ok {
		fmt.Fprintf(h.ErrorStream, "\n")

		if err := h.createTflocksTable(); err != nil {
			return err
		}
		fmt.Fprintf(h.ErrorStream, "\n  %s created %s dynamodb table\n", h.styles.tick, tflocksTableName)

	}
	return nil
}

func (h *Handler) createTflocksTable() error {
	dynamodbClient := h.getDynamoDBClient()
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

func (h *Handler) checkOrCreateLambdaBucket(buckets []string) error {
	ok, recoverable := h.handleLambdaBucket(nil, buckets)
	if !ok && !recoverable {
		fmt.Fprintf(h.ErrorStream, "\nUnable to resolve automatically.\n\n")
		return Exit(false)
	}
	if !ok {

		name, err := h.createLambdaBucket()
		if err != nil {
			return err
		}
		fmt.Fprintf(h.ErrorStream, "\n  %s created lambda bucket: %v\n", h.styles.tick, name)

	}
	return nil
}

func (h *Handler) createLambdaBucket() (string, error) {
	name := "cdflow2-lambda-" + randHexPostfix()
	if _, err := h.getS3Client().CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(name),
	}); err != nil {
		return "", err
	}
	return name, nil
}

func (h *Handler) checkOrCreateECRRepository(component string) error {
	repoURI, err := h.getECRRepository(component)
	if err != nil {
		return err
	}
	if repoURI == "" {
		fmt.Fprintf(h.ErrorStream, "\n")

		if err := h.createECRRepository(component); err != nil {
			return err
		}
		fmt.Fprintf(h.ErrorStream, "\n  %s created ECR registry: %v\n", h.styles.tick, component)

	}
	return nil
}

func (h *Handler) createECRRepository(component string) error {
	ecrClient := h.getECRClient()
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
