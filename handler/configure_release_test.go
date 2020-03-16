package handler_test

import (
	"bytes"
	"fmt"
	"strings"

	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/mergermarket/cdflow2-config-simple-aws/handler"

	common "github.com/mergermarket/cdflow2-config-common"
)

type mockedS3 struct {
	s3iface.S3API
	buckets []string
}

type mockedDynamoDB struct {
	dynamodbiface.DynamoDBAPI
}

func (m mockedDynamoDB) DescribeTable(*dynamodb.DescribeTableInput) (*dynamodb.DescribeTableOutput, error) {
	return &dynamodb.DescribeTableOutput{}, nil
}

type failingDynamoDB struct {
	dynamodbiface.DynamoDBAPI
}

func (m failingDynamoDB) DescribeTable(*dynamodb.DescribeTableInput) (*dynamodb.DescribeTableOutput, error) {
	return nil, fmt.Errorf("table not found")
}

func (s3Client mockedS3) ListBuckets(*s3.ListBucketsInput) (*s3.ListBucketsOutput, error) {
	var buckets []*s3.Bucket
	for _, bucket := range s3Client.buckets {
		buckets = append(buckets, &s3.Bucket{Name: aws.String(bucket)})
	}
	return &s3.ListBucketsOutput{
		Buckets: buckets,
	}, nil
}

func TestConfigureRelease(t *testing.T) {
	// Given
	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	handler := handler.New(&handler.Opts{OutputStream: &outputBuffer, ErrorStream: &errorBuffer})
	request := common.CreateConfigureReleaseRequest()
	response := common.CreateConfigureReleaseResponse()

	// When
	handler.ConfigureRelease(request, response)

	// Then
	if response.Success {
		t.Fatal("unexpected success, no config provided")
	}
}

func TestCheckAWSResources(t *testing.T) {
	t.Run("no buckets supplied", func(t *testing.T) {
		// Given
		var outputBuffer bytes.Buffer
		var errorBuffer bytes.Buffer

		handler, _ := handler.New(&handler.Opts{
			S3Client:       mockedS3{},
			OutputStream:   &outputBuffer,
			ErrorStream:    &errorBuffer,
			DynamoDBClient: &mockedDynamoDB{},
		}).(*handler.Handler)

		// When
		success := handler.CheckAWSResources()
		// Then
		if success {
			t.Fatal("unexpected success, output:", errorBuffer.String())
		}
		if !strings.Contains(errorBuffer.String(), "no release bucket found") {
			t.Fatal("expected 'no release bucket' message, got output:", errorBuffer.String())
		}
		if !strings.Contains(errorBuffer.String(), "no terraform state bucket found") {
			t.Fatal("expected 'no release bucket' message, got output:", errorBuffer.String())
		}
	})

	t.Run("multiple release buckets supplied", func(t *testing.T) {
		// Given
		var outputBuffer bytes.Buffer
		var errorBuffer bytes.Buffer

		handler, _ := handler.New(&handler.Opts{
			S3Client: mockedS3{buckets: []string{
				"cdflow2-release-bucket-1",
				"cdflow2-release-bucket-2",
			}},
			OutputStream:   &outputBuffer,
			ErrorStream:    &errorBuffer,
			DynamoDBClient: &mockedDynamoDB{},
		}).(*handler.Handler)

		// When
		success := handler.CheckAWSResources()
		// Then
		if success {
			t.Fatal("unexpected success, output:", errorBuffer.String())
		}
		if !strings.Contains(errorBuffer.String(), "multiple release buckets found") {
			t.Fatal("expected 'multiple release buckets' message, got output:", errorBuffer.String())
		}
	})

	t.Run("multiple tfstate buckets supplied", func(t *testing.T) {
		// Given
		var outputBuffer bytes.Buffer
		var errorBuffer bytes.Buffer

		handler, _ := handler.New(&handler.Opts{
			S3Client: mockedS3{buckets: []string{
				"cdflow2-release-bucket-1",
				"cdflow2-tfstate-bucket-1",
				"cdflow2-tfstate-bucket-2",
			}},
			OutputStream:   &outputBuffer,
			ErrorStream:    &errorBuffer,
			DynamoDBClient: &mockedDynamoDB{},
		}).(*handler.Handler)

		// When
		success := handler.CheckAWSResources()
		// Then
		if success {
			t.Fatal("unexpected success, output:", errorBuffer.String())
		}
		if !strings.Contains(errorBuffer.String(), "multiple terraform state buckets found") {
			t.Fatal("expected 'multiple terraform state buckets' message, got output:", errorBuffer.String())
		}
	})

	t.Run("correct buckets supplied - happy path", func(t *testing.T) {
		// Given
		var outputBuffer bytes.Buffer
		var errorBuffer bytes.Buffer

		handler, _ := handler.New(&handler.Opts{
			S3Client: mockedS3{buckets: []string{
				"cdflow2-release-bucket-1",
				"cdflow2-tfstate-bucket-1",
			}},
			OutputStream:   &outputBuffer,
			ErrorStream:    &errorBuffer,
			DynamoDBClient: &mockedDynamoDB{},
		}).(*handler.Handler)

		// When
		success := handler.CheckAWSResources()
		// Then
		if !success {
			t.Fatal("unexpected failure, output:", errorBuffer.String())
		}
		if !strings.Contains(errorBuffer.String(), "release bucket found") {
			t.Fatal("expected 'release bucket found' message, got output:", errorBuffer.String())
		}
		if !strings.Contains(errorBuffer.String(), "terraform state bucket found") {
			t.Fatal("expected 'terraform state bucket found' message, got output:", errorBuffer.String())
		}
	})

	t.Run("test tflock", func(t *testing.T) {
		// Given
		var outputBuffer bytes.Buffer
		var errorBuffer bytes.Buffer

		handler, _ := handler.New(&handler.Opts{
			S3Client: mockedS3{buckets: []string{
				"cdflow2-release-bucket-1",
				"cdflow2-tfstate-bucket-1",
			}},
			OutputStream:   &outputBuffer,
			ErrorStream:    &errorBuffer,
			DynamoDBClient: &failingDynamoDB{},
		}).(*handler.Handler)

		// When
		success := handler.CheckAWSResources()
		// Then
		if success {
			t.Fatal("unexpected success, output:", errorBuffer.String())
		}
	})

}
