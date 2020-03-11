package handler_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mergermarket/cdflow2-config-simple-aws/handler"
)

func TestCheckInputConfiguration(t *testing.T) {
	t.Run("errors in input configuration", func(t *testing.T) {
		// Given
		var outputBuffer bytes.Buffer
		var errorBuffer bytes.Buffer
		myHandler, _ := handler.New(&handler.Opts{OutputStream: &outputBuffer, ErrorStream: &errorBuffer}).(*handler.Handler)

		config := map[string]interface{}{}
		inputEnv := map[string]string{}

		// When
		success := myHandler.CheckInputConfiguration(config, inputEnv)

		// Then
		if success {
			t.Fatal("unexpected success")
		}
		if !strings.Contains(errorBuffer.String(), "missing config.params.default_region") {
			t.Fatal("didn't output message about missing region", errorBuffer.String())
		}
		if !strings.Contains(errorBuffer.String(), "missing AWS credentials") {
			t.Fatal("didn't output message about missing AWS credentials, output was:", errorBuffer.String())
		}
	})

	t.Run("successful input configuration", func(t *testing.T) {
		// Given
		var outputBuffer bytes.Buffer
		var errorBuffer bytes.Buffer
		myHandler, _ := handler.New(&handler.Opts{OutputStream: &outputBuffer, ErrorStream: &errorBuffer}).(*handler.Handler)

		config := map[string]interface{}{
			"default_region": "eu-west-1",
		}
		inputEnv := map[string]string{
			"AWS_ACCESS_KEY_ID":     "test-access-key",
			"AWS_SECRET_ACCESS_KEY": "test-secret-access-key",
		}

		// When
		success := myHandler.CheckInputConfiguration(config, inputEnv)

		// Then
		if !success {
			t.Fatal("unexpected failure")
		}
	})

}
