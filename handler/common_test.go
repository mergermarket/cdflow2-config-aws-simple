package handler_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mergermarket/cdflow2-config-simple-aws/handler"
)

func TestCheckInputConfiguration(t *testing.T) {
	t.Run("when there is no default region", func(t *testing.T) {
		// Given
		var outputBuffer bytes.Buffer
		var errorBuffer bytes.Buffer
		myHandler, _ := handler.New(&handler.Opts{OutputStream: &outputBuffer, ErrorStream: &errorBuffer}).(*handler.Handler)

		config := map[string]interface{}{}
		inputEnv := map[string]string{}
		outputEnv := map[string]string{}

		// When
		success := myHandler.CheckInputConfiguration(config, inputEnv, outputEnv)

		// Then
		if success {
			t.Fatal("unexpected success")
		}
		if !strings.Contains(errorBuffer.String(), "missing config.params.default_region") {
			t.Fatal("didn't output message about missing region", errorBuffer.String())
		}
	})
}
