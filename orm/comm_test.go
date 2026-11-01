package orm

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Conf is a test configuration.
type Conf struct {
	IDs []int64 `json:"ids"`
}

// Value impls  driver.Valuer for Range
func (p *Conf) Value() (driver.Value, error) {
	if p == nil {
		return nil, nil
	}
	v, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return string(v), nil
}

// Scan implements sql.Scanner for string and byte slice values.
func (p *Conf) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	var source []byte
	switch val := src.(type) {
	case string:
		source = []byte(val)
	case []byte:
		source = val
	default:
		return fmt.Errorf("Incompatible range %T,value:%v", val, val)
	}
	if len(source) > 0 {
		pv := &Conf{}
		err := json.Unmarshal(source, pv)
		if err != nil {
			return err
		}
		*p = *pv
	}
	return nil
}

// Conf2 is a test configuration.
type Conf2 struct {
	IDs []int64 `json:"ids"`
}

// Value impls  driver.Valuer for Range
func (p Conf2) Value() (driver.Value, error) {
	v, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return string(v), nil
}

// Scan implements sql.Scanner for string and byte slice values.
func (p *Conf2) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	var source []byte
	switch val := src.(type) {
	case string:
		source = []byte(val)
	case []byte:
		source = val
	default:
		return fmt.Errorf("Incompatible range %T,value:%v", val, val)
	}
	if len(source) > 0 {
		pv := &Conf2{}
		err := json.Unmarshal(source, pv)
		if err != nil {
			return err
		}
		*p = *pv
	}
	return nil
}

type AutoID struct {
	ID    int64          `column:"id" pk:"Y"`
	Name2 sql.NullString `column:"name2"`
}

type tmodel struct {
	AutoID
	BaseShardEntity
	Name  sql.NullString  `column:"name"`
	Time  sql.NullInt64   `column:"create_time"`
	F64   sql.NullFloat64 `column:"f64"`
	Conf  *Conf           `column:"conf"`
	Conf2 Conf2           `column:"conf2"`
	Ver   int64           `column:"ver"`
	Age   int64           `column:"age"`
}

func (tm *tmodel) TableName() string {
	return "tt"
}

type User struct {
	BaseShardEntity
	ID       int64          `column:"id" pk:"Y"`
	Name     sql.NullString `column:"name"`
	Age      int64          `column:"age"`
	Birthday NullTime       `column:"birthday"`
}

func (p *User) TableName() string {
	return "user"
}
