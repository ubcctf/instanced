package db

import (
	"encoding/json"
	"time"
)

// InstanceRecord is a record used to keep track of an active instance
type InstanceRecord struct {
	Id        int64     `json:"id"`
	Expiry    time.Time `json:"expiry"`
	Challenge string    `json:"challenge"`
	TeamID    string    `json:"team"`
	UUID      string    `json:"uuid"`
	Url       string    `json:"url"`
}

func (r *InstanceRecord) MarshalJSON() ([]byte, error) {
	type Alias InstanceRecord
	data := &struct {
		*Alias
		Expiry string `json:"expiry"`
	}{
		Alias:  (*Alias)(r),
		Expiry: r.Expiry.Format(time.TimeOnly) + " UTC",
	}
	if r.Expiry.Before(time.Now()) {
		data.Expiry = "Expired"
	}
	return json.Marshal(data)
}
