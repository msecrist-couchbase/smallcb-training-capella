package main

import "fmt"

func SessionExit(sessionId string) error {
	return nil // TODO.
}

func SessionCreate(fullName, email string) (sessionId string, err error) {
	if fullName == "test-bad" {
		return "", fmt.Errorf("ErrTestBad")
	}

	return "123456", nil
}
