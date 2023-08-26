package main

import (
	"database/sql"
	"errors"
	"time"

	_ "modernc.org/sqlite"
)

var initStatement = `
CREATE TABLE IF NOT EXISTS instances(id INTEGER PRIMARY KEY, challenge TEXT, expiry INTEGER);
DELETE FROM instances;
`

type InstanceRecord struct {
	id        int
	expiry    time.Time
	challenge string
}

func (in *Instancer) InitDB(file string) error {
	db, err := sql.Open("sqlite", file)
	if err != nil {
		return err
	}
	_, err = db.Exec(initStatement)
	if err != nil {
		return err
	}
	in.db = db
	return nil
}

func (in *Instancer) InsertInstanceRecord(ttl time.Duration, challenge string) (time.Time, error) {
	if in.db == nil {
		return time.Time{}, errors.New("db not initialized")
	}
	expiry := time.Now().Add(ttl)

	stmt, err := in.db.Prepare("INSERT INTO instances(challenge, expiry) values(?, ?)")
	if err != nil {
		return time.Now(), err
	}
	defer stmt.Close()

	_, err = stmt.Exec(challenge, expiry.Unix())
	if err != nil {
		return time.Now(), err
	}

	return expiry, nil
}

func (in *Instancer) DeleteInstanceRecord(id int) error {
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

func (in *Instancer) ReadInstanceRecords() ([]InstanceRecord, error) {
	if in.db == nil {
		return nil, errors.New("db not initialized")
	}
	rows, err := in.db.Query("SELECT id, challenge, expiry FROM instances")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := make([]InstanceRecord, 0)
	for rows.Next() {
		record := InstanceRecord{}
		var t int64
		err = rows.Scan(&record.id, &record.challenge, &t)
		if err != nil {
			return records, err
		}
		record.expiry = time.Unix(t, 0)
		records = append(records, record)
	}
	err = rows.Err()
	return records, err
}
