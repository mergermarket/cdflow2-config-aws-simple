package ui_test

import (
	"io"
	"log"
	"testing"
)

func TestConfirm(t *testing.T) {
	// Given
	inputReader, inputWriter = io.Pipe()
	outputReader, outputWriter = io.Pipe()

	ui := ui.New(inputReader, outputWriter)

	go func() {
		data := make([]byte, 1024)
		outputReader.Read(&data)
		log.Println("finished reading")
	}()

	// When
	confirmed := ui.Confirm("test")

	// Then
	if !confirmed {
		t.Fatal("expected confirmed")
	}
}
