package handler_test

import (
	"bytes"

	"testing"

	"github.com/mergermarket/cdflow2-config-simple-aws/handler"

	common "github.com/mergermarket/cdflow2-config-common"
)

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
