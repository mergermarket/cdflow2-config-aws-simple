package handler

import (
	"io"

	"github.com/mergermarket/cdflow2-config-simple-aws/command"
)

type handler struct{}

// New returns a new handler.
func New() command.Handler {
	return &handler{}
}

func (*handler) ConfigureRelease(request *command.ConfigureReleaseRequest, response *command.ConfigureReleaseResponse, errorStream io.Writer) error {

	return nil
}

func (*handler) UploadRelease(request *command.UploadReleaseRequest, response *command.UploadReleaseResponse, errorStream io.Writer, version string) error {

	return nil
}

func (*handler) PrepareTerraform(request *command.PrepareTerraformRequest, response *command.PrepareTerraformResponse, errorStream io.Writer) error {

	return nil
}
