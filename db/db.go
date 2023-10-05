package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type DBClient struct {
	*sql.DB
}

func InitDB(file string) (DBClient, error) {
	db, err := sql.Open("sqlite", file)
	if err != nil {
		return DBClient{}, err
	}
	// SQLite should only have a single connection
	db.SetMaxOpenConns(1)

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS instances(id INTEGER PRIMARY KEY, challenge TEXT, team TEXT, expiry INTEGER, uuid TEXT);")
	if err != nil {
		return DBClient{}, err
	}

	return DBClient{
		DB: db,
	}, nil
}

func (db *DBClient) InsertInstanceRecord(ttl time.Duration, team string, challenge string, cuuid string) (InstanceRecord, error) {
	expiry := time.Now().Add(ttl)

	stmt, err := db.Prepare("INSERT INTO instances(challenge, team, expiry, uuid) values(?, ?, ?, ?)")
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

func (db *DBClient) DeleteInstanceRecord(id int64) error {
	stmt, err := db.Prepare("DELETE FROM instances WHERE id = ?")
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

func (db *DBClient) ReadInstanceRecord(id int64) (InstanceRecord, error) {
	rows, err := db.Query("SELECT id, challenge, team, expiry, uuid FROM instances WHERE id = ?", id)
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

func (db *DBClient) ReadInstanceRecords() ([]InstanceRecord, error) {
	rows, err := db.Query("SELECT id, challenge, team, expiry, uuid FROM instances")
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

func (db *DBClient) ReadInstanceRecordsTeam(teamID string) ([]InstanceRecord, error) {
	stmt, err := db.Prepare("SELECT id, challenge, team, expiry, uuid FROM instances WHERE team = ?")
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
