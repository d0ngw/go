package counter

import (
	"encoding/json"
	"errors"
	"fmt"

	c "github.com/d0ngw/go/common"
	"github.com/d0ngw/go/orm"
)

// EntityCounter entity counter
type EntityCounter interface {
	orm.EntityInterface
	// Fields convert entity to Fields
	Fields() (Fields, error)
	// ZeroFields return zero fields,not nil
	ZeroFields() Fields
	// Entity convert fields to EntityInterface
	Entity(counterID string, fields Fields) (orm.EntityInterface, error)
}

// BaseEntity is the base counter entity
type BaseEntity struct {
	ID    string `column:"id" pk:"Y" pkAuto:"n"`
	Value string `column:"val"`
}

// TableName implements EntityInterface.TableName,it must be overrided or it will panic
func (p *BaseEntity) TableName() string {
	panic("please overried TableName method")
}

// Fields implements EntityCounter.Fields
func (p *BaseEntity) Fields() (Fields, error) {
	if p.Value == "" {
		return p.ZeroFields(), nil
	}
	fileds := Fields{}
	err := json.Unmarshal([]byte(p.Value), &fileds)
	if err != nil {
		return nil, err
	}
	return fileds, nil
}

// ZeroFields implements EntityCounter.ZeroFields,must be overrided
func (p *BaseEntity) ZeroFields() Fields {
	panic("please overried ZeroFields method")
}

// BaseEntity convert counterID and fields to BaseEntity
func (p *BaseEntity) BaseEntity(counterID string, fields Fields) (*BaseEntity, error) {
	v := &BaseEntity{ID: counterID}
	if fields == nil {
		return v, nil
	}
	b, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}
	v.Value = string(b)
	return v, nil
}

// DBPersist implements Persist which persist counter to db
type DBPersist struct {
	dbpool     *orm.DBPool
	entityType EntityCounter
}

// NewDBPersist create DBPersist
func NewDBPersist(dbpool *orm.DBPool, entityType EntityCounter) (*DBPersist, error) {
	if c.HasNil(dbpool, entityType) {
		return nil, errors.New("dbpool and entityType must not be nil")
	}
	return &DBPersist{
		dbpool:     dbpool,
		entityType: entityType,
	}, nil
}

// Load  implements Persist.Load
func (p *DBPersist) Load(counterID string) (fields Fields, err error) {
	oper := p.dbpool.NewDBOper()
	entity, err := orm.Get(oper, p.entityType, counterID)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return p.entityType.ZeroFields(), nil
	}
	if toCounter, ok := entity.(EntityCounter); ok {
		fields, err = toCounter.Fields()
		if err != nil {
			return nil, err
		}
		return
	}
	return nil, fmt.Errorf("%T is not a valid ToCounter", entity)
}

// Del implements Persist.Del
func (p *DBPersist) Del(counterID string) (deleted bool, err error) {
	oper := p.dbpool.NewDBOper()
	return orm.Del(oper, p.entityType, counterID)
}

// Store implements Persist.Store
func (p *DBPersist) Store(counterID string, fields Fields) (err error) {
	oper := p.dbpool.NewDBOper()
	entity, err := p.entityType.Entity(counterID, fields)
	if err != nil {
		return err
	}
	_, err = orm.AddOrUpdate(oper, entity)
	return
}
