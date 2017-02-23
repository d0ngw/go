// Package list has some list cache implemented by redis
package list

type ListEntity interface {
}

// BaseEntity define the base list entity
type BaseEntity struct {
	ID         int64  `column:"id" pk:"Y"` //auto increment id
	OwnerID    string `column:"o_id"`
	TargetID   int64  `column:"t_id"`
	CreateTime int64  `column:"ct"`
}

// TableName implement Entity.TableName()
func (p *BaseEntity) TableName() string {
	panic("please override this method")
}

// Cache define the list cache
type Cache struct {
}

func (p *Cache) Add(entity *BaseEntity)
