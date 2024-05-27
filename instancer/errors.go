package instancer

import "fmt"

type ChallengeNotFoundError struct {
	ChallengeID string
}

func (e *ChallengeNotFoundError) Error() string {
	return fmt.Sprintf("challenge not found %v", e.ChallengeID)
}

type InstanceExistsError struct {
	ChallengeID string
	TeamID      string
}

func (e *InstanceExistsError) Error() string {
	return fmt.Sprintf("team %v already has instance of challenge %v", e.TeamID, e.ChallengeID)
}
