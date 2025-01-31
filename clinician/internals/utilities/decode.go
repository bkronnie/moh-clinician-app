package utilities

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/schema"
)

func GetNullString(oldS string) (newS sql.NullString) {
	newS = sql.NullString{String: oldS, Valid: oldS != ""}
	return
}

func GetNullInt(oldI int64) (newI sql.NullInt64) {
	newI = sql.NullInt64{Int64: oldI, Valid: true}
	return
}

type NullableString struct {
	sql.NullString
}

// Implementing TextUnmarshaler interface
func (ns *NullableString) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		ns.Valid = false
		ns.String = ""
		return nil
	}
	ns.Valid = true
	ns.String = string(text)
	return nil
}

// NullableFloat64 represents a float64 that can be null
type NullableFloat64 struct {
	sql.NullFloat64
}

// Implementing TextUnmarshaler interface
func (nf *NullableFloat64) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		nf.Valid = false
		return nil
	}
	val, err := strconv.ParseFloat(string(text), 64)
	if err != nil {
		return err
	}
	nf.Float64 = val
	nf.Valid = true
	return nil
}

// NullableInt64 represents an int64 that can be null
type NullableInt64 struct {
	sql.NullInt64
}

// Implementing TextUnmarshaler interface
func (ni *NullableInt64) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		ni.Valid = false
		return nil
	}
	val, err := strconv.ParseInt(string(text), 10, 64)
	if err != nil {
		return err
	}
	ni.Int64 = val
	ni.Valid = true
	return nil
}

// DecodeFormData decodes form data into a given struct
// DecodeFormData decodes form data into a given struct

func DecodeFormData(c *gin.Context, v interface{}) error {
	// Parse form data into map[string][]string
	if err := c.Request.ParseForm(); err != nil {
		return err
	}

	decoder := schema.NewDecoder()

	// Set the strict option to false to ignore unknown fields
	decoder.IgnoreUnknownKeys(true)

	// Register converters for embedded sql.Null* types
	decoder.RegisterConverter(sql.NullString{}, func(s string) reflect.Value {
		var ns NullableString //sql.NullString

		if err := ns.UnmarshalText([]byte(s)); err != nil {
			return reflect.ValueOf(sql.NullString{})
		}
		return reflect.ValueOf(ns.NullString)
	})

	decoder.RegisterConverter(sql.NullFloat64{}, func(s string) reflect.Value {
		var nf NullableFloat64 //sql.NullFloat64
		if err := nf.UnmarshalText([]byte(s)); err != nil {
			return reflect.ValueOf(sql.NullFloat64{})
		}
		return reflect.ValueOf(nf.NullFloat64)
	})

	decoder.RegisterConverter(sql.NullInt64{}, func(s string) reflect.Value {
		var ni NullableInt64 //sql.NullInt64
		if err := ni.UnmarshalText([]byte(s)); err != nil {
			return reflect.ValueOf(sql.NullInt64{})
		}
		return reflect.ValueOf(ni.NullInt64)
	})

	// Decode form data into struct
	if err := decoder.Decode(v, c.Request.PostForm); err != nil {
		fmt.Println(err.Error())
		return err
	}

	return nil
}
