package handler_test

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	common "github.com/mergermarket/cdflow2-config-common"
	"github.com/mergermarket/cdflow2-config-simple-aws/handler"
)

type mockS3Client struct {
	s3iface.S3API
}

func (s3Client mockS3Client) ListBuckets(input *s3.ListBucketsInput) (*s3.ListBucketsOutput, error) {
	return &s3.ListBucketsOutput{
		Buckets: []*s3.Bucket{},
	}, nil
}

func TestConfigureReleaseNoDefaultRegion(t *testing.T) {
	request := common.CreateConfigureReleaseRequest()
	response := common.CreateConfigureReleaseResponse()
	var errors bytes.Buffer

	s3Client := &mockS3Client{}

	handler := handler.New(&handler.Opts{
		S3Client:    s3Client,
		ErrorStream: &errors,
	})

	if err := handler.ConfigureRelease(request, response); err != nil {
		log.Fatalln("unexpected error in configure release")
	}

	if response.Success {
		log.Fatal("unexpected success with no default region configured")
	}
	if !strings.Contains(errors.String(), "config.default_region is required") {
		log.Fatal("stderr output doesn't mention default region:", errors.String())
	}
}

func TestConfigureReleaseAWSCredentialsPassedThrough(t *testing.T) {
	request := common.CreateConfigureReleaseRequest()
	response := common.CreateConfigureReleaseResponse()

	handler := handler.New(&handler.Opts{
		S3Client: mockS3Client{},
	})

	request.Config["default_region"] = "test-region"

	request.Env["AWS_ACCESS_KEY_ID"] = "foo"
	request.Env["AWS_SECRET_ACCESS_KEY"] = "bar"
	request.Env["AWS_SESSION_TOKEN"] = "baz"

	if err := handler.ConfigureRelease(request, response); err != nil {
		log.Fatalln("unexpected error in configure release")
	}

	if response.Env["AWS_DEFAULT_REGION"] != "test-region" {
		log.Fatalln("AWS_DEFAULT_REGION not passed through, got:", response.Env["AWS_DEFAULT_REGION"])
	}
	if response.Env["AWS_ACCESS_KEY_ID"] != "foo" {
		log.Fatalln("AWS_ACCESS_KEY_ID not passed through, got:", response.Env["AWS_ACCESS_KEY_ID"])
	}
	if response.Env["AWS_SECRET_ACCESS_KEY"] != "bar" {
		log.Fatalln("AWS_SECRET_ACCESS_KEY not passed through, got:", response.Env["AWS_SECRET_ACCESS_KEY"])
	}
	if response.Env["AWS_SESSION_TOKEN"] != "baz" {
		log.Fatalln("AWS_SESSION_TOKEN not passed through, got:", response.Env["AWS_SESSION_TOKEN"])
	}
}

func TestReleaseBucketNotConfigured(t *testing.T) {
	request := common.CreateConfigureReleaseRequest()
	response := common.CreateConfigureReleaseResponse()

	var errors bytes.Buffer

	handler := handler.New(&handler.Opts{
		ErrorStream: &errors,
	})

	request.Config["default_region"] = "test-region"

	request.Env["AWS_ACCESS_KEY_ID"] = "foo"
	request.Env["AWS_SECRET_ACCESS_KEY"] = "bar"
	request.Env["AWS_SESSION_TOKEN"] = "baz"

	if err := handler.ConfigureRelease(request, response); err != nil {
		log.Fatalln("unexpected error in configure release")
	}

	if response.Success {
		log.Fatalln("unexpected success, should have failed due to missing config.release_bucket")
	}
	if !strings.Contains(errors.String(), "config.release_bucket is required") {
		log.Fatal("stderr output doesn't mention release bucket:", errors.String())
	}
}
