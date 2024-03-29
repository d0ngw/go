// Package orm 简单的DAL 封装
package orm

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"

	c "github.com/d0ngw/go/common"
)

// Meta meta
type Meta interface {
	Name() string
	Type() reflect.Type
	FieldValue(entity Entity, name string) (val interface{}, err error)
}

//AddMeta register entity meta
func AddMeta(entity Entity) Meta {
	m, err := defaultMetaReg.regModel(entity)
	if err != nil {
		panic(err)
	}
	return m
}

//MetaOf parse meta
func MetaOf(entity Entity) Meta {
	m, err := parseMeta(entity)
	if err != nil {
		panic(err)
	}
	return m
}

func findMeta(typ reflect.Type) *meta {
	pkgPath := typ.PkgPath()
	name := typ.Name()

	return findMetaWithPkgAndName(pkgPath, name)
}

func findMetaWithPkgAndName(pkgPath, name string) *meta {
	pkgPathCache := defaultMetaReg.pkgCache[pkgPath]
	if pkgPathCache == nil {
		return nil
	}
	return pkgPathCache[name]
}

type metaReg struct {
	cache    map[string]*meta
	pkgCache map[string]map[string]*meta
	done     bool
}

var (
	defaultMetaReg = &metaReg{
		cache:    make(map[string]*meta),
		pkgCache: make(map[string]map[string]*meta),
		done:     false,
	}
)

type meta struct {
	name                     string
	pkField                  *metaField
	fields                   []*metaField
	columnFields             map[string]*metaField
	modelType                reflect.Type
	insertFunc               entityInsertFunc
	updateFunc               entityUpdateFunc
	updateReplaceFunc        entityUpdateReplaceColumnsFunc
	updateExcludeColumnsFunc entityUpdateExcludeColumnsFunc
	updateColumnsFunc        entityUpdateColumnFunc
	entityQueryFunc          entityQueryFunc
	entityQueryColumnFunc    entityQueryColumnFunc
	clumnsQueryFunc          queryColumnsFunc
	getFunc                  entityGetFunc
	delFunc                  entityDeleteFunc
	delEFunc                 entityDeleteByIDFunc
	insertOrUpdateFunc       entityInsertOrUpdateFunc
}

// Name implements Meta.Name
func (p *meta) Name() string {
	return p.name
}

// Type implements Meta.Type
func (p *meta) Type() reflect.Type {
	return p.modelType
}

func (p *meta) FieldValue(entity Entity, name string) (val interface{}, err error) {
	if entity == nil || name == "" {
		return nil, fmt.Errorf("invalid params")
	}

	f := p.columnFields[name]
	if f == nil {
		return nil, fmt.Errorf("can't find field %s", name)
	}

	_, ind, _ := extract(entity)
	return ind.FieldByIndex(f.index).Interface(), nil
}

func (p *meta) String() string {
	return fmt.Sprintf("name:%s,fields:%s,pkfield:%s,type:%s", p.name, p.fields, p.pkField, p.modelType)
}

type metaField struct {
	name        string              //struct中的字段名称
	column      string              //表列名
	pk          bool                //是否主键
	pkAuto      bool                //如果是主键,是否是自增的id
	index       []int               //索引
	structField reflect.StructField //StructField
}

func (f *metaField) String() string {
	return fmt.Sprintf("{name:%v,conlumn:%v,pk:%v,pkAuto:%v,index:%v}", f.name, f.column, f.pk, f.pkAuto, f.index)
}

func (reg *metaReg) clean() {
	reg.cache = make(map[string]*meta)
	reg.pkgCache = make(map[string]map[string]*meta)
}

func (reg *metaReg) regModel(model Entity) (*meta, error) {
	if model == nil {
		return nil, NewDBError(nil, "invalid model")
	}
	m, err := parseMeta(model)
	if err != nil {
		return nil, err
	}

	err = reg.regMeta(m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (reg *metaReg) regMeta(m *meta) error {
	if m == nil {
		return NewDBError(nil, "Invalid meta")
	}

	if _, exist := reg.cache[m.name]; exist {
		return &DBError{"Duplicate mode name:" + m.name, nil}
	}

	metaType := m.Type()
	pkgPath := metaType.PkgPath()
	name := metaType.Name()

	pkgPathCache := reg.pkgCache[pkgPath]
	if pkgPathCache == nil {
		pkgPathCache = make(map[string]*meta)
		reg.pkgCache[pkgPath] = pkgPathCache
	}

	if _, exist := pkgPathCache[name]; exist {
		return &DBError{"Duplicate mode name:" + m.name, nil}
	}

	pkgPathCache[name] = m
	reg.cache[m.name] = m
	return nil
}

var (
	scannerType = reflect.TypeOf((*sql.Scanner)(nil)).Elem()
	valuerType  = reflect.TypeOf((*driver.Valuer)(nil)).Elem()
)

func parseMeta(model Entity) (*meta, error) {
	if model == nil {
		return nil, NewDBError(nil, "Invalid model")
	}

	val, ind, typ := extract(model)
	fullName := fullTypeName(typ)

	if val.Kind() != reflect.Ptr {
		panic(NewDBErrorf(nil, "Expect ptr ,but it's %s,type:%s", val.Kind(), typ))
	}
	if ind.Kind() != reflect.Struct {
		panic(NewDBErrorf(nil, "Expect struct ,but it's %s,type:%s", typ.Kind(), typ))
	}

	fieldCount := ind.NumField()
	fields := make([]*metaField, 0, fieldCount)
	mInfo := &meta{name: fullName, modelType: typ}
	var pkField *metaField

	fields = parseFields(nil, ind, typ, &pkField, fields)
	if pkField == nil {
		panic(NewDBErrorf(nil, "Can't find pk column for %s,found fields:%v", typ, fields))
	} else {
		mInfo.pkField = pkField
	}
	dupName := map[string]struct{}{}
	dupColumn := map[string]struct{}{}
	for _, field := range fields {
		if _, ok := dupName[field.name]; ok {
			panic(fmt.Errorf("Duplicate field name %s", field.name))
		} else {
			dupName[field.name] = struct{}{}
		}
		if _, ok := dupColumn[field.column]; ok {
			panic(fmt.Errorf("Duplicate column name %s", field.column))
		} else {
			dupColumn[field.column] = struct{}{}
		}
	}

	mInfo.fields = fields
	mInfo.insertFunc = createInsertFunc(mInfo)
	mInfo.updateFunc = createUpdateFunc(mInfo)
	mInfo.updateReplaceFunc = createUpdateReplaceFunc(mInfo)
	mInfo.updateExcludeColumnsFunc = createUpdateExcludeColmnsFunc(mInfo)
	mInfo.updateColumnsFunc = createUpdateColumnsFunc(mInfo)
	mInfo.entityQueryFunc = createQueryFunc(mInfo)
	mInfo.entityQueryColumnFunc = createQueryColumnFunc(mInfo)
	mInfo.clumnsQueryFunc = createQueryColumnsFunc(mInfo)
	mInfo.insertOrUpdateFunc = createInsertOrUpdateFunc(mInfo)
	mInfo.delFunc = createDelFunc(mInfo)
	mInfo.getFunc = func(executor interface{}, entity Entity, id interface{}) (e Entity, err error) {
		e = nil
		var l []Entity
		if l, err = mInfo.entityQueryFunc(executor, entity, " WHERE "+mInfo.pkField.column+" = ?", []interface{}{id}); err == nil {
			if len(l) == 1 {
				e = l[0]
			}
		}
		return
	}
	mInfo.delEFunc = func(executor interface{}, entity Entity, id interface{}) (r bool, err error) {
		var l int64
		if l, err = mInfo.delFunc(executor, entity, " WHERE "+mInfo.pkField.column+" = ?", []interface{}{id}); err == nil {
			if l == 1 {
				r = true
			}
		}
		return
	}

	columnFields := map[string]*metaField{}
	for _, field := range mInfo.fields {
		columnFields[field.column] = field
	}
	mInfo.columnFields = columnFields

	return mInfo, nil
}

func parseFields(index []int, ind reflect.Value, typ reflect.Type, pkField **metaField, fields []*metaField) []*metaField {
	for i := 0; i < ind.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}

		stFieldType := field.Type
		ptrStFieldType := reflect.PtrTo(stFieldType)
		isScannerAndValuer := (ptrStFieldType.Implements(scannerType) || stFieldType.Implements(scannerType)) && (ptrStFieldType.Implements(valuerType) || stFieldType.Implements(valuerType))

		if field.Type.Kind() == reflect.Ptr && !isScannerAndValuer {
			panic(NewDBErrorf(nil, "unsupported field type,%s is poniter,only scanner and valuer can be pointer", field.Name))
		}
		if stFieldType.Kind() == reflect.Struct && !isScannerAndValuer {
			if !field.Anonymous {
				panic(NewDBErrorf(nil, "field %s is struct it must be anonymous", field.Name))
			}

			newIndex := make([]int, len(index))
			copy(newIndex, index)
			fields = parseFields(append(newIndex, i), ind.Field(i), stFieldType, pkField, fields)
			continue
		}

		tag := field.Tag
		column, exist := tag.Lookup("column")
		if !exist {
			continue
		}

		if len(column) == 0 {
			panic(NewDBErrorf(nil, "Can't find the column tag for %s.%s,skip", typ, field.Name))
		}

		pk := strings.ToLower(tag.Get("pk"))
		pkAuto := strings.ToLower(tag.Get("pkAuto"))

		newIndex := make([]int, len(index))
		copy(newIndex, index)
		fieldIndex := append(newIndex, i)
		mField := &metaField{
			name:        field.Name,
			column:      column,
			pk:          pk == "y",
			pkAuto:      pk == "y" && !(pkAuto == "n"),
			index:       fieldIndex,
			structField: field}

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

func extract(model Entity) (val reflect.Value, ind reflect.Value, typ reflect.Type) {
	return c.ExtractRefTuple(model)
}

func fullTypeName(typ reflect.Type) string {
	return typ.PkgPath() + "." + typ.Name()
}
