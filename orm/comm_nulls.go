package orm

import (
	"database/sql"
	"database/sql/driver"
	"github.com/go-sql-driver/mysql"
)

//string,可能为空,重用sql.NullString
type NullString sql.NullString

func (ns *NullString) Scan(value interface{}) error {
	return (*sql.NullString)(ns).Scan(value)
}
func (ns NullString) Value() (driver.Value, error) {
	return sql.NullString(ns).Value()
}

//bool,可能为空,重用sql.NullBool
type NullBool sql.NullBool

func (ns *NullBool) Scan(value interface{}) error {
	return (*sql.NullBool)(ns).Scan(value)
}
func (ns NullBool) Value() (driver.Value, error) {
	return sql.NullBool(ns).Value()
}

//int64,可能为空,重用sql.NullInt64
type NullInt64 sql.NullInt64

func (ns *NullInt64) Scan(value interface{}) error {
	return (*sql.NullInt64)(ns).Scan(value)
}
func (ns NullInt64) Value() (driver.Value, error) {
	return sql.NullInt64(ns).Value()
}

//float64,可能为空
type NullFloat64 sql.NullFloat64

func (ns *NullFloat64) Scan(value interface{}) error {
	return (*sql.NullFloat64)(ns).Scan(value)
}
func (ns NullFloat64) Value() (driver.Value, error) {
	return sql.NullFloat64(ns).Value()
}

//time,可能为空的
type NullTime mysql.NullTime

func (ns *NullTime) Scan(value interface{}) error {
	return (*mysql.NullTime)(ns).Scan(value)
}
func (ns NullTime) Value() (driver.Value, error) {
	return mysql.NullTime(ns).Value()
}

//int,可能为空
type NullInt struct {
	Int   int
	Valid bool
}

func (n *NullInt) Scan(value interface{}) error {
	if value == nil {
		n.Int, n.Valid = 0, false
		return nil
	}
	n.Valid = true
	return convertAssign(&n.Int, value)
}

func (n NullInt) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Int, nil
}

//int8,可能为空
type NullInt8 struct {
	Int8  int8
	Valid bool
}

func (n *NullInt8) Scan(value interface{}) error {
	if value == nil {
		n.Int8, n.Valid = 0, false
		return nil
	}
	n.Valid = true
	return convertAssign(&n.Int8, value)
}

func (n NullInt8) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Int8, nil
}

//int16,可能为空
type NullInt16 struct {
	Int16 int16
	Valid bool
}

func (n *NullInt16) Scan(value interface{}) error {
	if value == nil {
		n.Int16, n.Valid = 0, false
		return nil
	}
	n.Valid = true
	return convertAssign(&n.Int16, value)
}

func (n NullInt16) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Int16, nil
}

//int32,可能为空
type NullInt32 struct {
	Int32 int32
	Valid bool
}

func (n *NullInt32) Scan(value interface{}) error {
	if value == nil {
		n.Int32, n.Valid = 0, false
		return nil
	}
	n.Valid = true
	return convertAssign(&n.Int32, value)
}

func (n NullInt32) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Int32, nil
}

//float32,可能为空
type NullFloat32 struct {
	Float32 float32
	Valid   bool
}

func (n *NullFloat32) Scan(value interface{}) error {
	if value == nil {
		n.Float32, n.Valid = 0, false
		return nil
	}
	n.Valid = true
	return convertAssign(&n.Float32, value)
}

func (n NullFloat32) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.Float32, nil
}
