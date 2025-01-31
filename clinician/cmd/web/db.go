package main

import (
	"database/sql"

	"github.com/moh/clinician/internals/utilities" // Import the required package

	_ "github.com/lib/pq"
)

func openDB(connStr string) (db *sql.DB, err error) {

	db, err = sql.Open("postgres", connStr)

	if err != nil {
		utilities.Danger(err, "Cannot connect to db")
		return nil, err
	}

	if err = db.Ping(); err != nil {
		utilities.Danger(err, "Cannot reach db")
		return nil, err
	}

	return
}
