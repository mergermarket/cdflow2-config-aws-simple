package handler

import (
	"fmt"
	"io"
	"log"

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

func (handler *handler) ConfigureRelease(request *common.ConfigureReleaseRequest, response *common.ConfigureReleaseResponse, errorStream io.Writer) error {
	region, _ := request.Config["default_region"].(string)
	if region == "" {
		fmt.Fprintln(errorStream, "cdflow.yaml: config.default_region is required")
		response.Success = false
		return nil
	}

	if request.Env["AWS_ACCESS_KEY_ID"] == "" || request.Env["AWS_SECRET_ACCESS_KEY"] == "" {
		log.Fatalln("no AWS credentials set in environment (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, [AWS_SESSION_TOKEN])")
	}

	response.Env["AWS_ACCESS_KEY_ID"] = request.Env["AWS_ACCESS_KEY_ID"]
	response.Env["AWS_SECRET_ACCESS_KEY"] = request.Env["AWS_SECRET_ACCESS_KEY"]
	if request.Env["AWS_SESSION_TOKEN"] != "" {
		response.Env["AWS_SESSION_TOKEN"] = request.Env["AWS_SESSION_TOKEN"]
	}

	return nil
}

func (handler *handler) UploadRelease(request *common.UploadReleaseRequest, response *common.UploadReleaseResponse, errorStream io.Writer, version string) error {

	return nil
}

func (handler *handler) PrepareTerraform(request *common.PrepareTerraformRequest, response *common.PrepareTerraformResponse, errorStream io.Writer) error {

	return nil
}
