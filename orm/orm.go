//简单的DAL 封装
package orm

import (
	"fmt"
	c "github.com/d0ngw/go/common"
	"reflect"
	"strings"
	"sync"
)

//模型元信息
type modelMeta struct {
	name              string        //模型的名称
	table             string        //对应的表名
	pkField           *modelField   //主键
	fields            []*modelField //字段列表
	modelType         reflect.Type  //模型的类型
	insertFunc        EntityCFunc   //插入函数
	updateFunc        EntityUFunc   //更新函数
	updateColumnsFunc EntityUCFunc  //更新列函数
	queryFunc         EntityQFunc   //查询函数
	getFunc           EntityGFunc   //根据ID查询实体的函数
	delFunc           EntityDFunc   //删除函数
	delEFunc          EntityDEFunc  //根据ID删除实体的函数
}

//模型的字段定义
type modelField struct {
	name        string              //struct中的字段名称
	column      string              //表列名
	pk          bool                //是否主键
	pkAuto      bool                //如果是主键,是否是自增的id
	index       int                 //索引
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

//实体的Create函数原型
type EntityCFunc func(executor interface{}, entity EntityInterface) error

//实体的Update函数原型
type EntityUFunc func(executor interface{}, entity EntityInterface) (bool, error)

//实体列的Update函数原型
type EntityUCFunc func(executor interface{}, entity EntityInterface, columns string, contition string, params []interface{}) (int64, error)

//实体的查询函数原型
type EntityQFunc func(executor interface{}, entity EntityInterface, condition string, params []interface{}) ([]EntityInterface, error)

//根据id获取单个实体的函数原型
type EntityGFunc func(executor interface{}, entity EntityInterface, id int64) (EntityInterface, error)

//实体的删除函数原型
type EntityDFunc func(executor interface{}, entity EntityInterface, condition string, params []interface{}) (int64, error)

//根据id删除实体的函数原型
type EntityDEFunc func(executor interface{}, entity EntityInterface, id int64) (bool, error)

//抽取
func extract(model EntityInterface) (reflect.Value, reflect.Value, reflect.Type) {
	return c.ExtractRefTuple(model)
}

func getFullModelName(typ reflect.Type) string {
	return typ.PkgPath() + "." + typ.Name()
}

// 注册数据模型
func AddModel(model EntityInterface) error {
	return _modelReg.RegModel(model)
}

func findModelInfo(typ reflect.Type) *modelMeta {
	if v, ok := _modelReg.cache[getFullModelName(typ)]; ok {
		return v
	} else {
		return nil
	}
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
	mInfo := &modelMeta{name: fullName, table: model.TableName(), modelType: typ}
	var pkField *modelField = nil

	for i := 0; i < ind.NumField(); i++ {
		structField := typ.Field(i)
		sfTag := structField.Tag
		column := sfTag.Get("column")
		pk := strings.ToLower(sfTag.Get("pk"))
		pkAuto := strings.ToLower(sfTag.Get("pkAuto"))
		if len(column) == 0 {
			panic(NewDBErrorf(nil, "Can't find the column tag for %s.%s", typ, structField.Name))
		}

		mField := &modelField{
			name:        structField.Name,
			column:      column,
			pk:          pk == "y",
			pkAuto:      !(pkAuto == "n"),
			index:       i,
			structField: structField}

		if mField.pk {
			if pkField == nil {
				pkField = mField
			} else {
				panic(NewDBErrorf(nil, "Duplicate pk column for %s.%s and %s ", typ, pkField.name, mField.name))
			}
		}
		fields = append(fields, mField)
	}

	if pkField == nil {
		panic(NewDBErrorf(nil, "Can't find pk column for %s", typ))
	} else {
		mInfo.pkField = pkField
	}
	c.Debugf("Register Model:%s,fields:%s,pkFiled:%+v", fullName, fields, pkField)

	mInfo.fields = fields
	mInfo.insertFunc = createInsertFunc(mInfo)
	mInfo.updateFunc = createUpdateFunc(mInfo)
	mInfo.updateColumnsFunc = createUpdateColumnsFunc(mInfo)
	mInfo.queryFunc = createQueryFunc(mInfo)
	mInfo.getFunc = func(executor interface{}, entity EntityInterface, id int64) (e EntityInterface, err error) {
		e = nil
		var l []EntityInterface = nil
		if l, err = mInfo.queryFunc(executor, entity, " WHERE "+mInfo.pkField.column+" = ?", []interface{}{id}); err == nil {
			if len(l) == 1 {
				e = l[0]
			}
		}
		return
	}
	mInfo.delFunc = createDelFunc(mInfo)
	mInfo.delEFunc = func(executor interface{}, entity EntityInterface, id int64) (r bool, err error) {
		var l int64
		if l, err = mInfo.delFunc(executor, entity, " WHERE "+mInfo.pkField.column+" = ?", []interface{}{id}); err == nil {
			if l == 1 {
				r = true
			}
		}
		return
	}

	_modelReg.lock.Lock()
	defer _modelReg.lock.Unlock()
	if _, exist := _modelReg.cache[fullName]; exist {
		return &DBError{"Duplicate mode name:" + fullName, nil}
	}
	_modelReg.cache[fullName] = mInfo
	return nil
}
