package main

import "fmt"

type ChallengeNotFoundError struct {
	chal string
}

func (e *ChallengeNotFoundError) Error() string {
	return fmt.Sprintf("challenge not found: %q", e.chal)
}
