// Package orm 简单的DAL 封装
package orm

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
	"sync"

	c "github.com/d0ngw/go/common"
)

//模型元信息
type modelMeta struct {
	name               string
	pkField            *modelField
	fields             []*modelField
	columnFields       map[string]*modelField
	modelType          reflect.Type
	insertFunc         entityInsertFunc
	updateFunc         entityUpdateFunc
	updateColumnsFunc  entityUpdateColumnFunc
	queryFunc          entityQueryFunc
	queryColumnFunc    entityQueryColumnFunc
	getFunc            entityGetFunc
	delFunc            entityDeleteFunc
	delEFunc           entityDeleteByIDFunc
	insertOrUpdateFunc entityInsertOrUpdateFunc
}

//模型的字段定义
type modelField struct {
	name        string              //struct中的字段名称
	column      string              //表列名
	pk          bool                //是否主键
	pkAuto      bool                //如果是主键,是否是自增的id
	index       []int               //索引
	structField reflect.StructField //StructField
}

func (f *modelField) String() string {
	return fmt.Sprintf("{name:%v,conlumn:%v,pk:%v,pkAuto:%v}", f.name, f.column, f.pk, f.pkAuto)
}

//模型注册
type modelReg struct {
	lock  *sync.RWMutex
	cache map[string]*modelMeta
	done  bool
}

var (
	//模型注册实例
	_modelReg = &modelReg{
		lock:  new(sync.RWMutex),
		cache: make(map[string]*modelMeta),
		done:  false,
	}
)

type entityInsertFunc func(executor interface{}, entity EntityInterface) error
type entityUpdateFunc func(executor interface{}, entity EntityInterface) (bool, error)
type entityUpdateColumnFunc func(executor interface{}, entity EntityInterface, columns string, contition string, params []interface{}) (int64, error)
type entityQueryFunc func(executor interface{}, entity EntityInterface, condition string, params []interface{}) ([]EntityInterface, error)
type entityQueryColumnFunc func(executor interface{}, entity EntityInterface, columns []string, condition string, params []interface{}) ([]EntityInterface, error)
type entityGetFunc func(executor interface{}, entity EntityInterface, id interface{}) (EntityInterface, error)
type entityDeleteFunc func(executor interface{}, entity EntityInterface, condition string, params []interface{}) (int64, error)
type entityDeleteByIDFunc func(executor interface{}, entity EntityInterface, id interface{}) (bool, error)
type entityInsertOrUpdateFunc func(executor interface{}, entity EntityInterface) (int64, error)

//抽取
func extract(model EntityInterface) (reflect.Value, reflect.Value, reflect.Type) {
	return c.ExtractRefTuple(model)
}

func getFullModelName(typ reflect.Type) string {
	return typ.PkgPath() + "." + typ.Name()
}

//AddModel 注册数据模型
func AddModel(model EntityInterface) error {
	return _modelReg.RegModel(model)
}

func findModelInfo(typ reflect.Type) *modelMeta {
	if v, ok := _modelReg.cache[getFullModelName(typ)]; ok {
		return v
	}
	return nil
}

func (reg *modelReg) clean() {
	reg.lock.Lock()
	defer _modelReg.lock.Unlock()
	reg.cache = make(map[string]*modelMeta)
}

//注册一个数据模型
func (reg *modelReg) RegModel(model EntityInterface) error {
	if model == nil {
		panic(NewDBError(nil, "Invalid model"))
	}

	val, ind, typ := extract(model)
	fullName := getFullModelName(typ)

	if val.Kind() != reflect.Ptr {
		panic(NewDBErrorf(nil, "Expect ptr ,but it's %s,type:%s", val.Kind(), typ))
	}
	if ind.Kind() != reflect.Struct {
		panic(NewDBErrorf(nil, "Expect struct ,but it's %s,type:%s", typ.Kind(), typ))
	}

	fieldCount := ind.NumField()
	fields := make([]*modelField, 0, fieldCount)
	mInfo := &modelMeta{name: fullName, modelType: typ}
	var pkField *modelField

	fields = reg.parseFields(nil, ind, typ, &pkField, fields)
	if pkField == nil {
		panic(NewDBErrorf(nil, "Can't find pk column for %s", typ))
	} else {
		mInfo.pkField = pkField
	}
	dupColumn := map[string]struct{}{}
	for _, field := range fields {
		if _, ok := dupColumn[field.name]; ok {
			panic(fmt.Errorf("Duplicate field name %s", field.name))
		} else {
			dupColumn[field.name] = struct{}{}
		}
	}
	c.Debugf("Register Model:%s,fields:%s,pkFiled:%+v", fullName, fields, pkField)

	mInfo.fields = fields
	mInfo.insertFunc = createInsertFunc(mInfo)
	mInfo.updateFunc = createUpdateFunc(mInfo)
	mInfo.updateColumnsFunc = createUpdateColumnsFunc(mInfo)
	mInfo.queryFunc = createQueryFunc(mInfo)
	mInfo.queryColumnFunc = createQueryColumnFunc(mInfo)
	mInfo.insertOrUpdateFunc = createInsertOrUpdateFunc(mInfo)
	mInfo.delFunc = createDelFunc(mInfo)
	mInfo.getFunc = func(executor interface{}, entity EntityInterface, id interface{}) (e EntityInterface, err error) {
		e = nil
		var l []EntityInterface
		if l, err = mInfo.queryFunc(executor, entity, " WHERE "+mInfo.pkField.column+" = ?", []interface{}{id}); err == nil {
			if len(l) == 1 {
				e = l[0]
			}
		}
		return
	}
	mInfo.delEFunc = func(executor interface{}, entity EntityInterface, id interface{}) (r bool, err error) {
		var l int64
		if l, err = mInfo.delFunc(executor, entity, " WHERE "+mInfo.pkField.column+" = ?", []interface{}{id}); err == nil {
			if l == 1 {
				r = true
			}
		}
		return
	}

	columnFields := map[string]*modelField{}
	for _, field := range mInfo.fields {
		columnFields[field.column] = field
	}
	mInfo.columnFields = columnFields

	_modelReg.lock.Lock()
	defer _modelReg.lock.Unlock()
	if _, exist := _modelReg.cache[fullName]; exist {
		return &DBError{"Duplicate mode name:" + fullName, nil}
	}
	_modelReg.cache[fullName] = mInfo
	return nil
}

var (
	scannerType = reflect.TypeOf((*sql.Scanner)(nil)).Elem()
	valuerType  = reflect.TypeOf((*driver.Valuer)(nil)).Elem()
)

func (reg *modelReg) parseFields(index []int, ind reflect.Value, typ reflect.Type, pkField **modelField, fields []*modelField) []*modelField {
	for i := 0; i < ind.NumField(); i++ {
		structField := typ.Field(i)
		if structField.Type.Kind() == reflect.Ptr {
			panic(NewDBErrorf(nil, "unsupported field type,%s is poniter", structField.Name))
		}
		stFieldType := structField.Type
		if stFieldType.Kind() == reflect.Struct && !(reflect.PtrTo(stFieldType).Implements(scannerType) && stFieldType.Implements(valuerType)) {
			if !structField.Anonymous {
				panic(NewDBErrorf(nil, "field %s is struct it must be anonymous", structField.Name))
			}
			fields = reg.parseFields(append(index, i), ind.Field(i), stFieldType, pkField, fields)
			continue
		}
		sfTag := structField.Tag
		column := sfTag.Get("column")
		pk := strings.ToLower(sfTag.Get("pk"))
		pkAuto := strings.ToLower(sfTag.Get("pkAuto"))
		if len(column) == 0 {
			panic(NewDBErrorf(nil, "Can't find the column tag for %s.%s", typ, structField.Name))
		}

		fieldIndex := append(index, i)
		mField := &modelField{
			name:        structField.Name,
			column:      column,
			pk:          pk == "y",
			pkAuto:      pk == "y" && !(pkAuto == "n"),
			index:       fieldIndex,
			structField: structField}

		if mField.pk {
			if *pkField == nil {
				*pkField = mField
			} else {
				panic(NewDBErrorf(nil, "Duplicate pk column for %s.%s and %s ", typ, (*pkField).name, mField.name))
			}
		}
		fields = append(fields, mField)
	}
	return fields
}
