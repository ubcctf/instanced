package main

import (
	"database/sql"
	"errors"
	"time"

	_ "modernc.org/sqlite"
)

type InstanceRecord struct {
	id        int64
	expiry    time.Time
	challenge string
}

func (in *Instancer) InitDB(file string) error {
	db, err := sql.Open("sqlite", file)
	if err != nil {
		return err
	}
	in.db = db
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS instances(id INTEGER PRIMARY KEY, challenge TEXT, expiry INTEGER);")
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

func (in *Instancer) InsertInstanceRecord(ttl time.Duration, challenge string) (InstanceRecord, error) {
	if in.db == nil {
		return InstanceRecord{}, errors.New("db not initialized")
	}
	expiry := time.Now().Add(ttl)

	stmt, err := in.db.Prepare("INSERT INTO instances(challenge, expiry) values(?, ?)")
	if err != nil {
		return InstanceRecord{}, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(challenge, expiry.Unix())
	if err != nil {
		return InstanceRecord{}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return InstanceRecord{}, err
	}

	return InstanceRecord{
		id:        id,
		expiry:    expiry,
		challenge: challenge,
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
