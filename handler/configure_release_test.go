package handler_test

import (
	"bytes"
	"strings"

	"testing"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/mergermarket/cdflow2-config-simple-aws/handler"

	common "github.com/mergermarket/cdflow2-config-common"
)

type mockedS3 struct {
	s3iface.S3API
}

func (s3Client mockedS3) ListBuckets(*s3.ListBucketsInput) (*s3.ListBucketsOutput, error) {
	return &s3.ListBucketsOutput{}, nil
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
	// Given
	var outputBuffer bytes.Buffer
	var errorBuffer bytes.Buffer

	handler, _ := handler.New(&handler.Opts{
		S3Client:     mockedS3{},
		OutputStream: &outputBuffer,
		ErrorStream:  &errorBuffer,
	}).(*handler.Handler)

	// When
	success := handler.CheckAWSResources()
	// Then
	if success {
		t.Fatal("unexpected success, output:", errorBuffer.String())
	}
	if !strings.Contains(errorBuffer.String(), "no release bucket found") {
		t.Fatal("no release bucket message, got output:", errorBuffer.String())
	}
}
