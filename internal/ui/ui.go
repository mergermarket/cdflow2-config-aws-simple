package ui

import (
	"bufio"
	"bytes"
)

// Confirm Asks a question and gets a yes or no answer
func Confirm(message string, userInput *bytes.Buffer, output *bytes.Buffer) bool {
	scanner := bufio.NewScanner(userInput)
	scanner.Scan()
	userText := scanner.Text()

	if userText == "yes" {
		return true
	}
	return false
}
