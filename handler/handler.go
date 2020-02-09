package handler

import (
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	common "github.com/mergermarket/cdflow2-config-common"
)

type handler struct {
	s3 s3iface.S3API
}

// New returns a new handler.
func New(s3 s3iface.S3API) common.Handler {
	return &handler{
		s3: s3,
	}
}

func handleDefaultRegion(config map[string]interface{}, env map[string]string, errorStream io.Writer) bool {
	region, _ := config["default_region"].(string)
	if region == "" {
		fmt.Fprintln(errorStream, "cdflow.yaml: config.default_region is required")
		return false
	}
	env["AWS_DEFAULT_REGION"] = region
	return true
}

func handleAWSCredentials(inputEnv map[string]string, outputEnv map[string]string, errorStream io.Writer) bool {
	if inputEnv["AWS_ACCESS_KEY_ID"] == "" || inputEnv["AWS_SECRET_ACCESS_KEY"] == "" {
		fmt.Fprintln(errorStream, "no AWS credentials set in environment (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, [AWS_SESSION_TOKEN])")
		return false
	}
	outputEnv["AWS_ACCESS_KEY_ID"] = inputEnv["AWS_ACCESS_KEY_ID"]
	outputEnv["AWS_SECRET_ACCESS_KEY"] = inputEnv["AWS_SECRET_ACCESS_KEY"]
	if inputEnv["AWS_SESSION_TOKEN"] != "" {
		outputEnv["AWS_SESSION_TOKEN"] = inputEnv["AWS_SESSION_TOKEN"]
	}
	return true
}

func (handler *handler) ConfigureRelease(request *common.ConfigureReleaseRequest, response *common.ConfigureReleaseResponse, errorStream io.Writer) error {
	if !handleDefaultRegion(request.Config, response.Env, errorStream) {
		response.Success = false
		return nil
	}

	if !handleAWSCredentials(request.Env, response.Env, errorStream) {
		response.Success = false
		return nil
	}

	return nil
}

func (handler *handler) UploadRelease(request *common.UploadReleaseRequest, response *common.UploadReleaseResponse, errorStream io.Writer, version string) error {

	return nil
}

func (handler *handler) PrepareTerraform(request *common.PrepareTerraformRequest, response *common.PrepareTerraformResponse, errorStream io.Writer) error {

	return nil
}
