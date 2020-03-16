package ui_test

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/mergermarket/cdflow2-config-simple-aws/internal/ui"
)

func TestConfirm(t *testing.T) {
	// Given
	userInput := &bytes.Buffer{}
	userInput.WriteString("yes\n")
	stdout := &bytes.Buffer{}
	question := "do you want to continue?"

	// When
	confirmed := ui.Confirm(question, userInput, stdout)

	// Then
	questionScanner := bufio.NewScanner(stdout)
	questionScanner.Scan()

	got := questionScanner.Text()
	if got != question {
		t.Fatalf("expected: %q, got: %q", question, got)
	}

	if !confirmed {
		t.Fatal("expected confirmed")
	}
}
