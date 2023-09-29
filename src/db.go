package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
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

func (in *Instancer) InitDB(file string) error {
	db, err := sql.Open("sqlite", file)
	if err != nil {
		return err
	}
	in.db = db
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS instances(id INTEGER PRIMARY KEY, challenge TEXT, team TEXT, expiry INTEGER, uuid TEXT);")
	if err != nil {
		return err
	}

	if !in.config.ResetDB {
		return nil
	}

	_, err = db.Exec("DELETE FROM instances;")
	if err != nil {
		return err
	}

	return nil
}

func (in *Instancer) InsertInstanceRecord(ttl time.Duration, team string, challenge string, cuuid string) (InstanceRecord, error) {
	if in.db == nil {
		return InstanceRecord{}, errors.New("db not initialized")
	}
	expiry := time.Now().Add(ttl)

	stmt, err := in.db.Prepare("INSERT INTO instances(challenge, team, expiry, uuid) values(?, ?, ?, ?)")
	if err != nil {
		return InstanceRecord{}, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(challenge, team, expiry.Unix(), cuuid)
	if err != nil {
		return InstanceRecord{}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return InstanceRecord{}, err
	}

	return InstanceRecord{
		Id:        id,
		Expiry:    expiry,
		Challenge: challenge,
		UUID:      cuuid,
	}, nil
}

func (in *Instancer) DeleteInstanceRecord(id int64) error {
	if in.db == nil {
		return errors.New("db not initialized")
	}
	stmt, err := in.db.Prepare("DELETE FROM instances WHERE id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(id)
	if err != nil {
		return err
	}

	return nil
}

func (in *Instancer) ReadInstanceRecord(id int64) (InstanceRecord, error) {
	if in.db == nil {
		return InstanceRecord{}, errors.New("db not initialized")
	}
	rows, err := in.db.Query("SELECT id, challenge, team, expiry, uuid FROM instances WHERE id = ?", id)
	if err != nil {
		return InstanceRecord{}, err
	}
	defer rows.Close()
	records := make([]InstanceRecord, 0)
	for rows.Next() {
		record := InstanceRecord{}
		var t int64
		err = rows.Scan(&record.Id, &record.Challenge, &record.TeamID, &t, &record.UUID)
		if err != nil {
			return InstanceRecord{}, err
		}
		record.Expiry = time.Unix(t, 0)
		records = append(records, record)
	}
	if len(records) != 1 {
		return InstanceRecord{}, fmt.Errorf("unique record not found with id %v", id)
	}
	err = rows.Err()
	return records[0], err
}

func (in *Instancer) ReadInstanceRecords() ([]InstanceRecord, error) {
	if in.db == nil {
		return nil, errors.New("db not initialized")
	}
	rows, err := in.db.Query("SELECT id, challenge, team, expiry, uuid FROM instances")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := make([]InstanceRecord, 0)
	for rows.Next() {
		record := InstanceRecord{}
		var t int64
		err = rows.Scan(&record.Id, &record.Challenge, &record.TeamID, &t, &record.UUID)
		if err != nil {
			return records, err
		}
		record.Expiry = time.Unix(t, 0)
		records = append(records, record)
	}
	err = rows.Err()
	return records, err
}

func (in *Instancer) ReadInstanceRecordsTeam(teamID string) ([]InstanceRecord, error) {
	if in.db == nil {
		return nil, errors.New("db not initialized")
	}
	stmt, err := in.db.Prepare("SELECT id, challenge, team, expiry, uuid FROM instances WHERE team = ?")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := make([]InstanceRecord, 0)
	for rows.Next() {
		record := InstanceRecord{}
		var t int64
		err = rows.Scan(&record.Id, &record.Challenge, &record.TeamID, &t, &record.UUID)
		if err != nil {
			return records, err
		}
		record.Expiry = time.Unix(t, 0)
		record.Url = fmt.Sprintf("http://%v.%v.ctf.maplebacon.org", record.UUID, record.Challenge)
		records = append(records, record)
	}
	err = rows.Err()
	return records, err
}
