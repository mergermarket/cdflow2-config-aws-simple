package ui

import (
	"bufio"
	"bytes"
	"fmt"
)

// Confirm Asks a question and gets a yes or no answer
func Confirm(message string, userInput *bytes.Buffer, output *bytes.Buffer) bool {
	fmt.Fprint(output, message)
	scanner := bufio.NewScanner(userInput)
	scanner.Scan()
	userText := scanner.Text()

	if userText == "yes" {
		return true
	}
	return false
}
