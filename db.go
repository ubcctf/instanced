package main

import (
	"database/sql"
	"time"
)

var initStatement = `
CREATE TABLE instances(id INTEGER PRIMARY KEY, challenge TEXT, expiry INTEGER);
DELETE FROM instances;
`

type InstanceRecord struct {
	id        int
	expiry    time.Time
	challenge string
}

func (in *Instancer) InitDB(file string) error {
	db, err := sql.Open("sqlite3", file)
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
	expiry := time.Now().Add(ttl)

	stmt, err := in.db.Prepare("INSERT INTO instances(challenge, expiry) values(?, ?)")
	if err != nil {
		return time.Now(), err
	}
	defer stmt.Close()

	_, err = stmt.Exec(challenge, expiry)
	if err != nil {
		return time.Now(), err
	}

	return expiry, nil
}

func (in *Instancer) DeleteInstanceRecord(id int) error {
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
	rows, err := in.db.Query("SELECT id, challenge, expiry FROM instances")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := make([]InstanceRecord, 15)
	for rows.Next() {
		record := InstanceRecord{}
		err = rows.Scan(&record.id, &record.challenge, &record.expiry)
		if err != nil {
			return records, err
		}
	}
	err = rows.Err()
	return records, err
}
