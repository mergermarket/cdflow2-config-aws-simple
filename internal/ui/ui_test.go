package ui_test

import (
	"bytes"
	"testing"

	"github.com/mergermarket/cdflow2-config-simple-aws/internal/ui"
)

func TestConfirm(t *testing.T) {
	// Given
	userInput := &bytes.Buffer{}
	userInput.WriteString("yes\n")

	stdout := &bytes.Buffer{}

	// When
	confirmed := ui.Confirm("do you want to continue?", userInput, stdout)

	// Then
	if !confirmed {
		t.Fatal("expected confirmed")
	}

}
