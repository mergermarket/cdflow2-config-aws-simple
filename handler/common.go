package handler

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	common "github.com/mergermarket/cdflow2-config-common"
)

const tflocksTableName = "cdflow2-tflocks"

var datadogAPIKeyArn = aws.String("arn:aws:secretsmanager:eu-west-1:109201950569:secret:cdflow2/datadog/datadog-api-key-ARHVef")

// Exit represents a planned exit without the need for further output.
type Exit bool

// Error outputs and empty string - the reason for failure will have already been output to the user.
func (Exit) Error() string {
	return ""
}

func (h *Handler) getAWSCredentials(inputEnv map[string]string) (bool, string, string, string) {
	if inputEnv["AWS_ACCESS_KEY_ID"] == "" || inputEnv["AWS_SECRET_ACCESS_KEY"] == "" {
		return false, "", "", ""
	}
	return true, inputEnv["AWS_ACCESS_KEY_ID"], inputEnv["AWS_SECRET_ACCESS_KEY"], inputEnv["AWS_SESSION_TOKEN"]
}

func (h *Handler) createAWSSession(accessKeyID, secretAccessKey, sessionToken string) {
	creds := credentials.NewStaticCredentials(accessKeyID, secretAccessKey, sessionToken)
	h.awsSession = session.Must(session.NewSession(&aws.Config{Credentials: creds, Region: &h.defaultRegion}))
}

func (h *Handler) printAWSCredentialsStatusMessage(ok bool) {
	if ok {
		fmt.Fprintf(h.ErrorStream, "  %s found AWS credentials in environment\n", h.styles.tick)
	} else {
		fmt.Fprintf(h.ErrorStream, "  %s missing AWS credentials in environment (AWS_ACCESS_KEY_ID & AWS_SECRET_ACCESS_KEY)\n", h.styles.cross)
	}
}

func (h *Handler) getDefaultRegion(config map[string]interface{}) string {
	region, _ := config["default_region"].(string)
	h.defaultRegion = region
	return region
}

func (h *Handler) printDefaultRegionStatusMessage(region string) {
	if region == "" {
		fmt.Fprintf(h.ErrorStream, "  %s missing config.params.default_region in cdflow.yaml\n", h.styles.cross)
	} else {
		fmt.Fprintf(h.ErrorStream, "  %s config.params.default_region in cdflow.yaml: %v\n", h.styles.tick, region)
	}
}

// CheckInputConfiguration checks config from cdflow.yaml and the input environment
func (h *Handler) CheckInputConfiguration(config map[string]interface{}, inputEnv map[string]string) bool {
	problems := 0

	fmt.Fprintf(h.ErrorStream, "\n%s\n\n", h.styles.au.Underline("Checking AWS configuration..."))
	if !h.handleDefaultRegion(config) {
		problems++
	}
	if !h.handleAWSCredentials(inputEnv) {
		problems++
	}
	fmt.Fprintln(h.ErrorStream, "")
	if problems > 0 {
		s := ""
		if problems > 1 {
			s = "s"
		}
		fmt.Fprintf(h.ErrorStream, "Please resolve the above problem%s and rerun the command.\n", s)
	}
	return problems == 0
}

func (h *Handler) handleDefaultRegion(config map[string]interface{}) bool {
	region := h.getDefaultRegion(config)
	h.printDefaultRegionStatusMessage(region)
	return region != ""
}

func (h *Handler) handleAWSCredentials(inputEnv map[string]string) bool {
	ok, accessKeyID, secretAccessKey, sessionToken := h.getAWSCredentials(inputEnv)
	h.printAWSCredentialsStatusMessage(ok)
	if !ok {
		return false
	}
	h.createAWSSession(accessKeyID, secretAccessKey, sessionToken)
	return true
}

func listBuckets(s3Client s3iface.S3API) ([]string, error) {
	response, err := s3Client.ListBuckets(&s3.ListBucketsInput{})
	if err != nil {
		return []string{}, err
	}
	var result []string
	for _, bucket := range response.Buckets {
		result = append(result, *bucket.Name)
	}
	return result, nil
}

func (h *Handler) handleReleaseBucket(buckets []string) (ok bool, recoverable bool) {
	buckets = filterPrefix(buckets, "cdflow2-release-")
	if len(buckets) == 0 {
		fmt.Fprintf(h.ErrorStream, "  %s no release bucket found with prefix 'cdflow2-release-'\n", h.styles.cross)
		return false, true
	} else if len(buckets) > 1 {
		fmt.Fprintf(h.ErrorStream, "  %s multiple release buckets found with prefix 'cdflow2-release-', there should be exactly one\n", h.styles.cross)
		return false, false
	}
	fmt.Fprintf(h.ErrorStream, "  %s release bucket found: %v\n", h.styles.tick, buckets[0])
	h.releaseBucket = buckets[0]
	return true, false
}

func (h *Handler) handleTfstateBucket(buckets []string) (bool, bool) {
	buckets = filterPrefix(buckets, "cdflow2-tfstate-")
	if len(buckets) == 0 {
		fmt.Fprintf(h.ErrorStream, "  %s no terraform state bucket found with prefix 'cdflow2-tfstate-'\n", h.styles.cross)
		return false, true
	} else if len(buckets) > 1 {
		fmt.Fprintf(h.ErrorStream, "  %s multiple terraform state buckets found with prefix 'cdflow2-tfstate-', there should be exactly one\n", h.styles.cross)
		return false, false
	}
	fmt.Fprintf(h.ErrorStream, "  %s terraform state bucket found: %v\n", h.styles.tick, buckets[0])
	h.tfstateBucket = buckets[0]
	return true, false
}

func (h *Handler) handleTflocksTable() bool {
	_, err := h.getDynamoDBClient().DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(tflocksTableName),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == dynamodb.ErrCodeResourceNotFoundException {
			fmt.Fprintf(h.ErrorStream, "  %s dynamodb table not found: %s\n", h.styles.cross, tflocksTableName)
			return false
		}
		log.Panic(err)
	}
	fmt.Fprintf(h.ErrorStream, "  %s dynamodb table found: %s\n", h.styles.tick, tflocksTableName)
	h.tflocksTable = tflocksTableName
	return true
}

func (h *Handler) handleLambdaBucket(outputEnv map[string]string, buckets []string) (bool, bool) {
	buckets = filterPrefix(buckets, "cdflow2-lambda-")
	if len(buckets) == 0 {
		fmt.Fprintf(h.ErrorStream, "  %s no cdflow2-lambda-... S3 bucket found (required only if building a lambda)\n", h.styles.warningCross)
		return false, true
	} else if len(buckets) > 1 {
		fmt.Fprintf(h.ErrorStream, "  %s multiple cdflow2-lambda-... S3 buckets found - there should be at most one\n", h.styles.warningCross)
		return false, false
	}
	fmt.Fprintf(h.ErrorStream, "  %s lambda bucket found: %v\n", h.styles.tick, buckets[0])
	h.lambdaBucket = buckets[0]
	if outputEnv != nil {
		outputEnv["LAMBDA_BUCKET"] = buckets[0]
	}
	return true, false
}

func (h *Handler) handleECRRepository(component string, outputEnv map[string]string) (bool, error) {
	response, err := h.getECRClient().DescribeRepositories(&ecr.DescribeRepositoriesInput{
		RepositoryNames: []*string{aws.String(component)},
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == ecr.ErrCodeRepositoryNotFoundException {
			fmt.Fprintf(h.ErrorStream, "  %s no %s ECR repository (required only for docker images)\n", h.styles.warningCross, component)
			return false, nil
		}
		return false, err
	}
	fmt.Fprintf(h.ErrorStream, "  %s ECR repository found: %s\n", h.styles.tick, component)
	if outputEnv != nil {
		outputEnv["ECR_REPOSITORY"] = *response.Repositories[0].RepositoryUri
	}
	return true, nil
}

func (h *Handler) requiresLambdaBucket(releaseRequiredEnv map[string]*common.ReleaseRequirements) bool {
	for _, requiredEnv := range releaseRequiredEnv {
		for _, envVar := range requiredEnv.Needs {
			if envVar == "LAMBDA_BUCKET" {
				return true
			}
		}
	}
	return false
}

func (h *Handler) getDatadogAPIKey() string {
	client := h.getSecretManagerClient()

	value, err := client.GetSecretValue(&secretsmanager.GetSecretValueInput{SecretId: datadogAPIKeyArn})
	if err != nil {
		fmt.Fprintf(h.ErrorStream, "Unable to fetch Datadog API key: %v.\n\n", err)
		return ""
	}

	return *value.SecretString
}
