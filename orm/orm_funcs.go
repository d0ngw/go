package orm

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/BurntSushi/ty/fun"
	c "github.com/d0ngw/go/common"
)

func toSlice(s string, count int) []string {
	slice := make([]string, 0, count)
	for i := 0; i < count; i++ {
		slice = append(slice, s)
	}
	return slice
}

//除了自增主键的过滤函数
var exceptIDPred = func(field *modelField) bool {
	if field == nil || (field.pk && field.pkAuto) {
		return false
	}
	return true
}

//除了主键的过滤函数
var noIDPred = func(field *modelField) bool {
	if field == nil || field.pk {
		return false
	}
	return true
}

//检查实体参数
func checkEntity(modelInfo *modelMeta, entity EntityInterface, tx interface{}) (ind reflect.Value) {
	val, ind, typ := extract(entity)
	if val.Kind() != reflect.Ptr {
		panic(NewDBErrorf(nil, "Expect ptr ,but it's %s,type:%s", val.Kind(), typ))
	}
	if typ != modelInfo.modelType {
		panic(NewDBErrorf(nil, "Not same model type %v and %v", typ, modelInfo.modelType))
	}
	if tx == nil {
		panic(NewDBError(nil, "Not in Trans"))
	}
	return
}

func exec(executor interface{}, execSQL string, args []interface{}) (rs sql.Result, err error) {
	c.Debugf("Exec sql %s with %T", execSQL, executor)
	if tx, ok := executor.(*sql.Tx); ok {
		rs, err = tx.Exec(execSQL, args...)
	} else if db, ok := executor.(*sql.DB); ok {
		rs, err = db.Exec(execSQL, args...)
	} else {
		panic(NewDBErrorf(nil, "Not a valid executor:%T", executor))
	}
	return
}

func query(executor interface{}, execSQL string, args []interface{}) (rows *sql.Rows, err error) {
	c.Debugf("Exec sql %s with %T", execSQL, executor)
	if tx, ok := executor.(*sql.Tx); ok {
		rows, err = tx.Query(execSQL, args...)
	} else if db, ok := executor.(*sql.DB); ok {
		rows, err = db.Query(execSQL, args...)
	} else {
		panic(NewDBErrorf(nil, "Not a valid executor:%T", executor))
	}
	return
}

func buildParamValues(ind reflect.Value, fields []*modelField) []interface{} {
	paramValues := make([]interface{}, 0, len(fields))
	for _, field := range fields {
		fv := ind.FieldByIndex(field.index).Interface()
		paramValues = append(paramValues, fv)
	}
	return paramValues
}

//构建实体模型的插入函数
func createInsertFunc(modelInfo *modelMeta) entityInsertFunc {
	insertFields := fun.Filter(exceptIDPred, modelInfo.fields).([]*modelField)
	columns := strings.Join(fun.Map(func(field *modelField) string {
		return field.column
	}, insertFields).([]string), ",")
	params := strings.Join(toSlice("?", len(insertFields)), ",")

	return func(executor interface{}, entity EntityInterface) error {
		ind := checkEntity(modelInfo, entity, executor)
		paramValues := buildParamValues(ind, insertFields)
		insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES(%s)", entity.TableName(), columns, params)
		c.Debugf("insertSql:%s", insertSQL)

		rs, err := exec(executor, insertSQL, paramValues)
		if err != nil {
			return err
		}

		if modelInfo.pkField.pkAuto {
			if id, err := rs.LastInsertId(); err == nil {
				ind.FieldByIndex(modelInfo.pkField.index).SetInt(id)
			} else {
				return err
			}
		}
		return nil
	}
}

//构建实体模型的更新函数
func createUpdateFunc(modelInfo *modelMeta) entityUpdateFunc {
	updateFields := fun.Filter(exceptIDPred, modelInfo.fields).([]*modelField)
	columns := strings.Join(fun.Map(func(field *modelField) string {
		return field.column + "=?"
	}, updateFields).([]string), ",")

	return func(executor interface{}, entity EntityInterface) (bool, error) {
		ind := checkEntity(modelInfo, entity, executor)
		id := ind.FieldByIndex(modelInfo.pkField.index).Interface()
		paramValues := buildParamValues(ind, updateFields)
		paramValues = append(paramValues, id)

		updateSQL := fmt.Sprintf("UPDATE %s SET %s where %s = %s", entity.TableName(), columns, modelInfo.pkField.column, "?")
		c.Debugf("updateSql:%s", updateSQL)
		rs, err := exec(executor, updateSQL, paramValues)
		if err != nil {
			return false, err
		}
		//检查更新的记录数
		rows, err := rs.RowsAffected()
		if err == nil {
			if rows != 1 {
				return false, nil
			}
			return true, nil
		}
		return false, err
	}
}

//构建实体模型的指定类名的更新函数
func createUpdateColumnsFunc(modelInfo *modelMeta) entityUpdateColumnFunc {
	return func(executor interface{}, entity EntityInterface, columns string, condition string, params []interface{}) (int64, error) {
		checkEntity(modelInfo, entity, executor)
		if len(columns) == 0 {
			panic(NewDBError(nil, "Can't update empty columns"))
		}
		updateSQL := fmt.Sprintf("UPDATE %s SET %s ", entity.TableName(), columns)
		if len(condition) > 0 {
			updateSQL += condition

		}
		c.Debugf("updateColumnSql:%s", updateSQL)

		rs, err := exec(executor, updateSQL, params)
		if err != nil {
			return 0, err
		}

		//检查更新的记录数
		rows, err := rs.RowsAffected()
		if err == nil {
			c.Debugf("Updated rows:%v", rows)
			return rows, nil
		}
		return 0, err
	}
}

//构建查询函数
func createQueryFunc(modelInfo *modelMeta) entityQueryFunc {
	columns := strings.Join(fun.Map(func(field *modelField) string {
		return "`" + field.column + "`"
	}, modelInfo.fields).([]string), ",")

	return func(executor interface{}, entity EntityInterface, condition string, params []interface{}) ([]EntityInterface, error) {
		ind := checkEntity(modelInfo, entity, executor)
		querySQL := fmt.Sprintf("SELECT %s FROM %s ", columns, entity.TableName())
		if len(condition) > 0 {
			querySQL += condition
		}
		c.Debugf("querySql:%v", querySQL)

		rows, err := query(executor, querySQL, params)
		if err != nil {
			return nil, err
		}

		defer rows.Close()

		var rt = make([]EntityInterface, 0, 10)
		for rows.Next() {
			ptrValue := reflect.New(ind.Type())
			ptrValueInd := reflect.Indirect(ptrValue)
			ptrValueSlice := make([]interface{}, 0, len(modelInfo.fields))
			for _, field := range modelInfo.fields {
				fv := ptrValueInd.FieldByIndex(field.index).Addr().Interface()
				ptrValueSlice = append(ptrValueSlice, fv)
			}
			if err := rows.Scan(ptrValueSlice...); err == nil {
				rt = append(rt, ptrValue.Interface().(EntityInterface))
			} else {
				return nil, err
			}
		}
		return rt, nil
	}
}

//构建查询函数
func createQueryColumnFunc(modelInfo *modelMeta) entityQueryColumnFunc {
	return func(executor interface{}, entity EntityInterface, columns []string, condition string, params []interface{}) ([]EntityInterface, error) {
		ind := checkEntity(modelInfo, entity, executor)
		fields := make([]*modelField, 0, len(columns))
		for _, column := range columns {
			if field, ok := modelInfo.columnFields[column]; ok {
				fields = append(fields, field)
			}
		}

		querySQL := fmt.Sprintf("SELECT %s FROM %s ", strings.Join(columns, ","), entity.TableName())
		if len(condition) > 0 {
			querySQL += condition
		}
		c.Debugf("querySql:%v", querySQL)

		rows, err := query(executor, querySQL, params)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var rt = make([]EntityInterface, 0, 10)
		for rows.Next() {
			ptrValue := reflect.New(ind.Type())
			ptrValueInd := reflect.Indirect(ptrValue)
			ptrValueSlice := make([]interface{}, 0, len(modelInfo.fields))
			for _, field := range fields {
				fv := ptrValueInd.FieldByIndex(field.index).Addr().Interface()
				ptrValueSlice = append(ptrValueSlice, fv)
			}

			if err := rows.Scan(ptrValueSlice...); err == nil {
				rt = append(rt, ptrValue.Interface().(EntityInterface))
			} else {
				return nil, err
			}
		}
		return rt, nil
	}
}

//构建查询函数
func createQueryColumnsFunc(modelInfo *modelMeta) queryColumnsFunc {
	return func(executor interface{}, entity EntityInterface, destStructs interface{}, columns []string, condition string, params []interface{}) error {
		if destStructs == nil {
			return errors.New("dest must not be nil")
		}

		ptrVal := reflect.ValueOf(destStructs)
		if ptrVal.Kind() != reflect.Ptr {
			return errors.New("the length of dest must be 1")
		}

		var destVal = ptrVal.Elem()
		if destVal.Kind() != reflect.Slice {
			return errors.New("the destStructs must be slice")
		}

		destStructTyp := destVal.Type().Elem()
		if destStructTyp.Kind() != reflect.Ptr || destStructTyp.Elem().Kind() != reflect.Struct {
			return errors.New("the element of dest must be struct pointer")
		}

		destTyp := destStructTyp.Elem()
		if destTyp.NumField() < len(columns) {
			return fmt.Errorf("number of %T's fields must >= columns", destTyp)
		}

		querySQL := fmt.Sprintf("SELECT %s FROM %s ", strings.Join(columns, ","), entity.TableName())
		if len(condition) > 0 {
			querySQL += condition
		}
		c.Debugf("querySql:%v", querySQL)

		rows, err := query(executor, querySQL, params)
		if err != nil {
			return err
		}
		defer rows.Close()

		var rt = reflect.MakeSlice(destVal.Type(), 0, 10)
		for rows.Next() {
			ptrValue := reflect.New(destTyp)
			ptrValueInd := reflect.Indirect(ptrValue)
			ptrValueSlice := make([]interface{}, 0, destTyp.NumField())
			for i := 0; i < destTyp.NumField(); i++ {
				fv := ptrValueInd.Field(i).Addr().Interface()
				ptrValueSlice = append(ptrValueSlice, fv)
			}
			if err := rows.Scan(ptrValueSlice...); err == nil {
				rt = reflect.Append(rt, ptrValue)
			} else {
				return err
			}
		}
		ptrVal.Elem().Set(rt)
		return nil
	}
}

//构建删除函数
func createDelFunc(modelInfo *modelMeta) entityDeleteFunc {
	return func(executor interface{}, entity EntityInterface, condition string, params []interface{}) (int64, error) {
		checkEntity(modelInfo, entity, executor)
		delSQL := fmt.Sprintf("DELETE FROM %s ", entity.TableName())
		if len(condition) > 0 {
			delSQL += condition
		}
		c.Debugf("delSql:%v", delSQL)

		rs, err := exec(executor, delSQL, params)
		if err != nil {
			return 0, err
		}
		//检查更新的记录数
		rows, err := rs.RowsAffected()
		if err != nil {
			return 0, err
		}
		return rows, nil
	}
}

func createInsertOrUpdateFunc(modelInfo *modelMeta) entityInsertOrUpdateFunc {
	insertFields := fun.Filter(exceptIDPred, modelInfo.fields).([]*modelField)
	columns := strings.Join(fun.Map(func(field *modelField) string {
		return field.column
	}, insertFields).([]string), ",")
	insertParams := strings.Join(toSlice("?", len(insertFields)), ",")

	updateFields := fun.Filter(noIDPred, modelInfo.fields).([]*modelField)
	updateColumns := strings.Join(fun.Map(func(field *modelField) string {
		return field.column + "=?"
	}, updateFields).([]string), ",")

	return func(executor interface{}, entity EntityInterface) (int64, error) {
		ind := checkEntity(modelInfo, entity, executor)
		paramValues := buildParamValues(ind, insertFields)
		updateParamValues := buildParamValues(ind, updateFields)
		allParamValues := append(paramValues, updateParamValues...)
		insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES(%s) ON DUPLICATE KEY UPDATE %s", entity.TableName(), columns, insertParams, updateColumns)
		c.Debugf("insertSql:%s", insertSQL)

		rs, err := exec(executor, insertSQL, allParamValues)
		if err != nil {
			return 0, err
		}

		//检查更新的记录数
		rows, err := rs.RowsAffected()
		if err == nil {
			c.Debugf("Updated rows:%v", rows)
			return rows, nil
		}
		return 0, err
	}
}
