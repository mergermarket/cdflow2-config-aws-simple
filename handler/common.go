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
	common "github.com/mergermarket/cdflow2-config-common"
)

const tflocksTableName = "cdflow2-tflocks"

// Exit represents a planned exit without the need for further output.
type Exit bool

// Error outputs and empty string - the reason for failure will have already been output to the user.
func (Exit) Error() string {
	return ""
}

func (handler *Handler) getAWSCredentials(inputEnv map[string]string) (bool, string, string, string) {
	if inputEnv["AWS_ACCESS_KEY_ID"] == "" || inputEnv["AWS_SECRET_ACCESS_KEY"] == "" {
		return false, "", "", ""
	}
	return true, inputEnv["AWS_ACCESS_KEY_ID"], inputEnv["AWS_SECRET_ACCESS_KEY"], inputEnv["AWS_SESSION_TOKEN"]
}

func (handler *Handler) createAWSSession(accessKeyID, secretAccessKey, sessionToken string) {
	creds := credentials.NewStaticCredentials(accessKeyID, secretAccessKey, sessionToken)
	handler.awsSession = session.Must(session.NewSession(&aws.Config{Credentials: creds, Region: &handler.defaultRegion}))
}

func (handler *Handler) printAWSCredentialsStatusMessage(ok bool) {
	if ok {
		fmt.Fprintf(handler.ErrorStream, "  %s found AWS credentials in environment\n", handler.styles.tick)
	} else {
		fmt.Fprintf(handler.ErrorStream, "  %s missing AWS credentials in environment (AWS_ACCESS_KEY_ID & AWS_SECRET_ACCESS_KEY)\n", handler.styles.cross)
	}
}

func (handler *Handler) getDefaultRegion(config map[string]interface{}) string {
	region, _ := config["default_region"].(string)
	handler.defaultRegion = region
	return region
}

func (handler *Handler) printDefaultRegionStatusMessage(region string) {
	if region == "" {
		fmt.Fprintf(handler.ErrorStream, "  %s missing config.params.default_region in cdflow.yaml\n", handler.styles.cross)
	} else {
		fmt.Fprintf(handler.ErrorStream, "  %s config.params.default_region in cdflow.yaml: %v\n", handler.styles.tick, region)
	}
}

// CheckInputConfiguration checks config from cdflow.yaml and the input environment
func (handler *Handler) CheckInputConfiguration(config map[string]interface{}, inputEnv map[string]string) bool {
	problems := 0

	fmt.Fprintf(handler.ErrorStream, "\n%s\n\n", handler.styles.au.Underline("Checking AWS configuration..."))
	if !handler.handleDefaultRegion(config) {
		problems++
	}
	if !handler.handleAWSCredentials(inputEnv) {
		problems++
	}
	fmt.Fprintln(handler.ErrorStream, "")
	if problems > 0 {
		s := ""
		if problems > 1 {
			s = "s"
		}
		fmt.Fprintf(handler.ErrorStream, "Please resolve the above problem%s and rerun the command.\n", s)
	}
	return problems == 0
}

func (handler *Handler) handleDefaultRegion(config map[string]interface{}) bool {
	region := handler.getDefaultRegion(config)
	handler.printDefaultRegionStatusMessage(region)
	return region != ""
}

func (handler *Handler) handleAWSCredentials(inputEnv map[string]string) bool {
	ok, accessKeyID, secretAccessKey, sessionToken := handler.getAWSCredentials(inputEnv)
	handler.printAWSCredentialsStatusMessage(ok)
	if !ok {
		return false
	}
	handler.createAWSSession(accessKeyID, secretAccessKey, sessionToken)
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

func (handler *Handler) handleReleaseBucket(buckets []string) (ok bool, recoverable bool) {
	buckets = filterPrefix(buckets, "cdflow2-release-")
	if len(buckets) == 0 {
		fmt.Fprintf(handler.ErrorStream, "  %s no release bucket found with prefix 'cdflow2-release-'\n", handler.styles.cross)
		return false, true
	} else if len(buckets) > 1 {
		fmt.Fprintf(handler.ErrorStream, "  %s multiple release buckets found with prefix 'cdflow2-release-', there should be exactly one\n", handler.styles.cross)
		return false, false
	}
	fmt.Fprintf(handler.ErrorStream, "  %s release bucket found: %v\n", handler.styles.tick, buckets[0])
	handler.releaseBucket = buckets[0]
	return true, false
}

func (handler *Handler) handleTfstateBucket(buckets []string) (bool, bool) {
	buckets = filterPrefix(buckets, "cdflow2-tfstate-")
	if len(buckets) == 0 {
		fmt.Fprintf(handler.ErrorStream, "  %s no terraform state bucket found with prefix 'cdflow2-tfstate-'\n", handler.styles.cross)
		return false, true
	} else if len(buckets) > 1 {
		fmt.Fprintf(handler.ErrorStream, "  %s multiple terraform state buckets found with prefix 'cdflow2-tfstate-', there should be exactly one\n", handler.styles.cross)
		return false, false
	}
	fmt.Fprintf(handler.ErrorStream, "  %s terraform state bucket found: %v\n", handler.styles.tick, buckets[0])
	handler.tfstateBucket = buckets[0]
	return true, false
}

func (handler *Handler) handleTflocksTable() bool {
	_, err := handler.getDynamoDBClient().DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(tflocksTableName),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == dynamodb.ErrCodeResourceNotFoundException {
			fmt.Fprintf(handler.ErrorStream, "  %s dynamodb table not found: %s\n", handler.styles.cross, tflocksTableName)
			return false
		}
		log.Panic(err)
	}
	fmt.Fprintf(handler.ErrorStream, "  %s dynamodb table found: %s\n", handler.styles.tick, tflocksTableName)
	handler.tflocksTable = tflocksTableName
	return true
}

func (handler *Handler) handleLambdaBucket(outputEnv map[string]string, buckets []string) (bool, bool) {
	buckets = filterPrefix(buckets, "cdflow2-lambda-")
	if len(buckets) == 0 {
		fmt.Fprintf(handler.ErrorStream, "  %s no cdflow2-lambda-... S3 bucket found (required only if building a lambda)\n", handler.styles.warningCross)
		return false, true
	} else if len(buckets) > 1 {
		fmt.Fprintf(handler.ErrorStream, "  %s multiple cdflow2-lambda-... S3 buckets found - there should be at most one\n", handler.styles.warningCross)
		return false, false
	}
	fmt.Fprintf(handler.ErrorStream, "  %s lambda bucket found: %v\n", handler.styles.tick, buckets[0])
	handler.lambdaBucket = buckets[0]
	if outputEnv != nil {
		outputEnv["LAMBDA_BUCKET"] = buckets[0]
	}
	return true, false
}

func (handler *Handler) getECRRepository(component string) (string, error) {
	response, err := handler.getECRClient().DescribeRepositories(&ecr.DescribeRepositoriesInput{
		RepositoryNames: []*string{aws.String(component)},
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == ecr.ErrCodeRepositoryNotFoundException {
			fmt.Fprintf(handler.ErrorStream, "  %s no %s ECR repository (required only for docker images)\n", handler.styles.warningCross, component)
			return "", nil
		}
		return "", err
	}
	fmt.Fprintf(handler.ErrorStream, "  %s ECR repository found: %s\n", handler.styles.tick, component)

	return *response.Repositories[0].RepositoryUri, nil
}

func (handler *Handler) requiresLambdaBucket(releaseRequiredEnv map[string]*common.ReleaseRequirements) bool {
	for _, requiredEnv := range releaseRequiredEnv {
		for _, envVar := range requiredEnv.Needs {
			if envVar == "LAMBDA_BUCKET" {
				return true
			}
		}
	}
	return false
}
