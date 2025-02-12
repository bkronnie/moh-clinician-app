package models

// Code generated by xo. DO NOT EDIT.

import (
	"context"
	"database/sql"
	"strconv"
	"fmt"
)

// User represents a row from 'public.users'.
type User struct {
	ID        int           `json:"id"`         // id
	Username  string        `json:"username"`   // username
	Pssword   string        `json:"pssword"`    // pssword
	Employees sql.NullInt64 `json:"employees"`  // employees
	Rights    string        `json:"rights"`     // rights
	CreatedBy sql.NullInt64 `json:"created_by"` // created_by
	CreatedOn sql.NullTime  `json:"created_on"` // created_on
	// xo fields
	_exists, _deleted bool
}

// Exists returns true when the [User] exists in the database.
func (u *User) Exists() bool {
	return u._exists
}

// Deleted returns true when the [User] has been marked for deletion
// from the database.
func (u *User) Deleted() bool {
	return u._deleted
}

// Insert inserts the [User] to the database.
func (u *User) Insert(ctx context.Context, db DB) error {
	switch {
	case u._exists: // already exists
		return logerror(&ErrInsertFailed{ErrAlreadyExists})
	case u._deleted: // deleted
		return logerror(&ErrInsertFailed{ErrMarkedForDeletion})
	}
	// insert (primary key generated and returned by database)
	const sqlstr = `INSERT INTO public.users (` +
		`username, pssword, employees, rights, created_by, created_on` +
		`) VALUES (` +
		`$1, $2, $3, $4, $5, $6` +
		`) RETURNING id`
	// run
	logf(sqlstr, u.Username, Encrypt(u.Pssword), u.Employees, u.Rights, u.CreatedBy, u.CreatedOn)
	if err := db.QueryRowContext(ctx, sqlstr, u.Username, u.Pssword, u.Employees, u.Rights, u.CreatedBy, u.CreatedOn).Scan(&u.ID); err != nil {
		return logerror(err)
	}
	// set exists
	u._exists = true
	return nil
}

// Update updates a [User] in the database.
func (u *User) Update(ctx context.Context, db DB) error {
	switch {
	case !u._exists: // doesn't exist
		return logerror(&ErrUpdateFailed{ErrDoesNotExist})
	case u._deleted: // deleted
		return logerror(&ErrUpdateFailed{ErrMarkedForDeletion})
	}
	// update with composite primary key
	const sqlstr = `UPDATE public.users SET ` +
		`username = $1, pssword = $2, employees = $3, rights = $4, created_by = $5, created_on = $6 ` +
		`WHERE id = $7`
	// run
	logf(sqlstr, u.Username, u.Pssword, u.Employees, u.Rights, u.CreatedBy, u.CreatedOn, u.ID)
	if _, err := db.ExecContext(ctx, sqlstr, u.Username, u.Pssword, u.Employees, u.Rights, u.CreatedBy, u.CreatedOn, u.ID); err != nil {
		return logerror(err)
	}
	return nil
}

// Save saves the [User] to the database.
func (u *User) Save(ctx context.Context, db DB) error {
	if u.Exists() {
		return u.Update(ctx, db)
	}
	return u.Insert(ctx, db)
}

// Upsert performs an upsert for [User].
func (u *User) Upsert(ctx context.Context, db DB) error {
	switch {
	case u._deleted: // deleted
		return logerror(&ErrUpsertFailed{ErrMarkedForDeletion})
	}
	// upsert
	const sqlstr = `INSERT INTO public.users (` +
		`id, username, pssword, employees, rights, created_by, created_on` +
		`) VALUES (` +
		`$1, $2, $3, $4, $5, $6, $7` +
		`)` +
		` ON CONFLICT (id) DO ` +
		`UPDATE SET ` +
		`username = EXCLUDED.username, pssword = EXCLUDED.pssword, employees = EXCLUDED.employees, rights = EXCLUDED.rights, created_by = EXCLUDED.created_by, created_on = EXCLUDED.created_on `
	// run
	logf(sqlstr, u.ID, u.Username, Encrypt(u.Pssword), u.Employees, u.Rights, u.CreatedBy, u.CreatedOn)
	if _, err := db.ExecContext(ctx, sqlstr, u.ID, u.Username, u.Pssword, u.Employees, u.Rights, u.CreatedBy, u.CreatedOn); err != nil {
		return logerror(err)
	}
	// set exists
	u._exists = true
	return nil
}

// Delete deletes the [User] from the database.
func (u *User) Delete(ctx context.Context, db DB) error {
	switch {
	case !u._exists: // doesn't exist
		return nil
	case u._deleted: // deleted
		return nil
	}
	// delete with single primary key
	const sqlstr = `DELETE FROM public.users ` +
		`WHERE id = $1`
	// run
	logf(sqlstr, u.ID)
	if _, err := db.ExecContext(ctx, sqlstr, u.ID); err != nil {
		return logerror(err)
	}
	// set deleted
	u._deleted = true
	return nil
}

// UserByID retrieves a row from 'public.users' as a [User].
//
// Generated from index 'users_pkey'.
func UserByID(ctx context.Context, db DB, id int) (*User, error) {
	// query
	const sqlstr = `SELECT ` +
		`id, username, pssword, employees, rights, created_by, created_on ` +
		`FROM public.users ` +
		`WHERE id = $1`
	// run
	logf(sqlstr, id)
	u := User{
		_exists: true,
	}
	if err := db.QueryRowContext(ctx, sqlstr, id).Scan(&u.ID, &u.Username, &u.Pssword, &u.Employees, &u.Rights, &u.CreatedBy, &u.CreatedOn); err != nil {
		return nil, logerror(err)
	}
	return &u, nil
}

func UserByEmail(ctx context.Context, db DB, email string) (*User, error) {
	// query
	const sqlstr = `SELECT ` +
		`id, username, pssword, employees, rights, created_by, created_on ` +
		`FROM public.users ` +
		`WHERE username = $1`
	// run
	logf(sqlstr, email)
	u := User{
		_exists: true,
	}
	if err := db.QueryRowContext(ctx, sqlstr, email).Scan(&u.ID, &u.Username, &u.Pssword, &u.Employees, &u.Rights, &u.CreatedBy, &u.CreatedOn); err != nil {
		return nil, logerror(err)
	}
	return &u, nil
}

func UserByEmailPass(ctx context.Context, db DB, email string, pass string) (*User, error) {
	// query
	const sqlstr = `SELECT ` +
		`id, username, pssword, employees, rights, created_by, created_on ` +
		`FROM public.users ` +
		`WHERE username = $1 AND pssword = $2`
	// run
	
	zepass := Encrypt(pass)
	
	logf(sqlstr, email, zepass)
	u := User{
		_exists: true,
	}
	if err := db.QueryRowContext(ctx, sqlstr, email, zepass).Scan(&u.ID, &u.Username, &u.Pssword, &u.Employees, &u.Rights, &u.CreatedBy, &u.CreatedOn); err != nil {
		return nil, logerror(err)
	}
	return &u, nil
}

func Users(ctx context.Context, db DB, flt string, start int, cnt int) ([]*User, error) {
	var sqlstr, whereString string

	whereString = ""
	if flt!= "" {
        whereString = "WHERE " + flt
    }

	lmt := ""
	if cnt > 0 {
		lmt = " LIMIT " + strconv.Itoa(start)  + " " + strconv.Itoa(cnt) 
	} 

	sqlstr = `SELECT ` +
	         `id, username, pssword, employees, rights, created_by, created_on ` +
			 `FROM public.users ` + whereString + lmt
	
	rows, err := db.QueryContext(ctx,sqlstr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []*User{}

	for rows.Next() {
		t := &User{}
		err = rows.Scan(
			&t.ID, 
			&t.Username, 
			&t.Pssword, 
			&t.Employees, 
			&t.Rights, 
			&t.CreatedBy, 
			&t.CreatedOn, 
		)

		if err != nil {
		return nil, err
		}

		users = append(users, t)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
	
}

func UserCount(ctx context.Context, db DB, flt string) (int, error) {
	var whereString string

	whereString = ""
	if flt!= "" {
        whereString = "WHERE " + flt
    }
	// SQL statement to count the rows in the users table
	stml := "SELECT COUNT(id) AS C FROM users " + whereString

	// Query the database
	rows, err := db.QueryContext(ctx, stml)
	if err != nil {
		return 0, fmt.Errorf("error querying user count: %w", err)
	}
	defer rows.Close()

	var count int

	// Iterate over the result (though it should only have one row)
	if rows.Next() {
		// Scan the result into the count variable
		err = rows.Scan(&count)
		if err != nil {
			return 0, fmt.Errorf("error scanning count: %w", err)
		}
	}

	// Check for errors after looping through the rows
	if err = rows.Err(); err != nil {
		return 0, fmt.Errorf("error reading result set: %w", err)
	}

	return count, nil
}
