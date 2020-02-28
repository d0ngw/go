package orm

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/BurntSushi/ty/fun"
)

type entityInsertFunc func(executor interface{}, entity Entity) error
type entityUpdateFunc func(executor interface{}, entity Entity) (bool, error)
type entityUpdateExcludeColumnsFunc func(executor interface{}, entity Entity, columns ...string) (bool, error)
type entityUpdateColumnFunc func(executor interface{}, entity Entity, columns string, contition string, params []interface{}) (int64, error)
type entityQueryFunc func(executor interface{}, entity Entity, condition string, params []interface{}) ([]Entity, error)
type entityQueryColumnFunc func(executor interface{}, entity Entity, columns []string, condition string, params []interface{}) ([]Entity, error)
type queryColumnsFunc func(executor interface{}, entity Entity, destStruct interface{}, columns []string, condition string, params []interface{}) error
type entityGetFunc func(executor interface{}, entity Entity, id interface{}) (Entity, error)
type entityDeleteFunc func(executor interface{}, entity Entity, condition string, params []interface{}) (int64, error)
type entityDeleteByIDFunc func(executor interface{}, entity Entity, id interface{}) (bool, error)
type entityInsertOrUpdateFunc func(executor interface{}, entity Entity) (int64, error)

func toSlice(s string, count int) []string {
	slice := make([]string, 0, count)
	for i := 0; i < count; i++ {
		slice = append(slice, s)
	}
	return slice
}

//除了自增主键的过滤函数
var exceptIDPred = func(field *metaField) bool {
	if field == nil || (field.pk && field.pkAuto) {
		return false
	}
	return true
}

//除了主键的过滤函数
var noIDPred = func(field *metaField) bool {
	if field == nil || field.pk {
		return false
	}
	return true
}

//检查实体参数
func checkEntity(modelInfo *meta, entity Entity, tx interface{}) (ind reflect.Value) {
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
	if tx, ok := executor.(*sql.Tx); ok {
		rows, err = tx.Query(execSQL, args...)
	} else if db, ok := executor.(*sql.DB); ok {
		rows, err = db.Query(execSQL, args...)
	} else {
		panic(NewDBErrorf(nil, "Not a valid executor:%T", executor))
	}
	return
}

func buildParamValues(ind reflect.Value, fields []*metaField) []interface{} {
	paramValues := make([]interface{}, 0, len(fields))
	for _, field := range fields {
		fv := ind.FieldByIndex(field.index).Interface()
		paramValues = append(paramValues, fv)
	}
	return paramValues
}

//构建实体模型的插入函数
func createInsertFunc(modelInfo *meta) entityInsertFunc {
	insertFields := fun.Filter(exceptIDPred, modelInfo.fields).([]*metaField)
	columns := strings.Join(fun.Map(func(field *metaField) string {
		return field.column
	}, insertFields).([]string), ",")
	params := strings.Join(toSlice("?", len(insertFields)), ",")

	return func(executor interface{}, entity Entity) error {
		ind := checkEntity(modelInfo, entity, executor)
		paramValues := buildParamValues(ind, insertFields)
		tname, err := tblName(entity)
		if err != nil {
			return err
		}
		insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES(%s)", tname, columns, params)

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
func createUpdateFunc(modelInfo *meta) entityUpdateFunc {
	updateFields := fun.Filter(noIDPred, modelInfo.fields).([]*metaField)
	columns := strings.Join(fun.Map(func(field *metaField) string {
		return field.column + "=?"
	}, updateFields).([]string), ",")

	return func(executor interface{}, entity Entity) (bool, error) {
		ind := checkEntity(modelInfo, entity, executor)
		id := ind.FieldByIndex(modelInfo.pkField.index).Interface()
		paramValues := buildParamValues(ind, updateFields)
		paramValues = append(paramValues, id)

		tname, err := tblName(entity)
		if err != nil {
			return false, err
		}

		updateSQL := fmt.Sprintf("UPDATE %s SET %s where %s = %s", tname, columns, modelInfo.pkField.column, "?")
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

func createUpdateExcludeColmnsFunc(modelInfo *meta) entityUpdateExcludeColumnsFunc {
	fields := fun.Filter(noIDPred, modelInfo.fields).([]*metaField)

	return func(executor interface{}, entity Entity, excludeColumns ...string) (bool, error) {
		updateFields := fields
		if len(excludeColumns) > 0 {
			var excludeColumnsMap = map[string]struct{}{}
			for _, column := range excludeColumns {
				excludeColumnsMap[column] = struct{}{}
			}

			leftFieldCount := len(updateFields) - len(excludeColumns)
			if leftFieldCount <= 0 {
				return false, errors.New("no column to update")
			}

			updateFields = make([]*metaField, 0, leftFieldCount)
			for _, f := range fields {
				if _, ok := excludeColumnsMap[f.column]; ok {
					continue
				}
				updateFields = append(updateFields, f)
			}
		}

		columns := strings.Join(fun.Map(func(field *metaField) string {
			return field.column + "=?"
		}, updateFields).([]string), ",")

		ind := checkEntity(modelInfo, entity, executor)
		id := ind.FieldByIndex(modelInfo.pkField.index).Interface()
		paramValues := buildParamValues(ind, updateFields)
		paramValues = append(paramValues, id)

		tname, err := tblName(entity)
		if err != nil {
			return false, err
		}

		updateSQL := fmt.Sprintf("UPDATE %s SET %s where %s = %s", tname, columns, modelInfo.pkField.column, "?")
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
func createUpdateColumnsFunc(modelInfo *meta) entityUpdateColumnFunc {
	return func(executor interface{}, entity Entity, columns string, condition string, params []interface{}) (int64, error) {
		checkEntity(modelInfo, entity, executor)
		if len(columns) == 0 {
			panic(NewDBError(nil, "Can't update empty columns"))
		}

		tname, err := tblName(entity)
		if err != nil {
			return 0, err
		}
		updateSQL := fmt.Sprintf("UPDATE %s SET %s ", tname, columns)
		if len(condition) > 0 {
			updateSQL += condition

		}

		rs, err := exec(executor, updateSQL, params)
		if err != nil {
			return 0, err
		}

		//检查更新的记录数
		rows, err := rs.RowsAffected()
		if err == nil {
			return rows, nil
		}
		return 0, err
	}
}

//构建查询函数
func createQueryFunc(modelInfo *meta) entityQueryFunc {
	columns := strings.Join(fun.Map(func(field *metaField) string {
		return "`" + field.column + "`"
	}, modelInfo.fields).([]string), ",")

	return func(executor interface{}, entity Entity, condition string, params []interface{}) ([]Entity, error) {
		ind := checkEntity(modelInfo, entity, executor)
		tname, err := tblName(entity)
		if err != nil {
			return nil, err
		}
		querySQL := fmt.Sprintf("SELECT %s FROM %s ", columns, tname)
		if len(condition) > 0 {
			querySQL += condition
		}

		rows, err := query(executor, querySQL, params)
		if err != nil {
			return nil, err
		}

		defer rows.Close()

		var rt = make([]Entity, 0, 10)
		for rows.Next() {
			ptrValue := reflect.New(ind.Type())
			ptrValueInd := reflect.Indirect(ptrValue)
			ptrValueSlice := make([]interface{}, 0, len(modelInfo.fields))
			for _, field := range modelInfo.fields {
				fv := ptrValueInd.FieldByIndex(field.index).Addr().Interface()
				ptrValueSlice = append(ptrValueSlice, fv)
			}
			if err := rows.Scan(ptrValueSlice...); err == nil {
				rt = append(rt, ptrValue.Interface().(Entity))
			} else {
				return nil, err
			}
		}
		return rt, nil
	}
}

//构建查询函数
func createQueryColumnFunc(modelInfo *meta) entityQueryColumnFunc {
	return func(executor interface{}, entity Entity, columns []string, condition string, params []interface{}) ([]Entity, error) {
		ind := checkEntity(modelInfo, entity, executor)
		fields := make([]*metaField, 0, len(columns))
		for _, column := range columns {
			if field, ok := modelInfo.columnFields[column]; ok {
				fields = append(fields, field)
			}
		}

		tname, err := tblName(entity)
		if err != nil {
			return nil, err
		}

		querySQL := fmt.Sprintf("SELECT %s FROM %s ", strings.Join(columns, ","), tname)
		if len(condition) > 0 {
			querySQL += condition
		}

		rows, err := query(executor, querySQL, params)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var rt = make([]Entity, 0, 10)
		for rows.Next() {
			ptrValue := reflect.New(ind.Type())
			ptrValueInd := reflect.Indirect(ptrValue)
			ptrValueSlice := make([]interface{}, 0, len(modelInfo.fields))
			for _, field := range fields {
				fv := ptrValueInd.FieldByIndex(field.index).Addr().Interface()
				ptrValueSlice = append(ptrValueSlice, fv)
			}

			if err := rows.Scan(ptrValueSlice...); err == nil {
				rt = append(rt, ptrValue.Interface().(Entity))
			} else {
				return nil, err
			}
		}
		return rt, nil
	}
}

//构建查询函数
func createQueryColumnsFunc(modelInfo *meta) queryColumnsFunc {
	return func(executor interface{}, entity Entity, destStructs interface{}, columns []string, condition string, params []interface{}) error {
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

		tname, err := tblName(entity)
		if err != nil {
			return err
		}

		querySQL := fmt.Sprintf("SELECT %s FROM %s ", strings.Join(columns, ","), tname)
		if len(condition) > 0 {
			querySQL += condition
		}

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
func createDelFunc(modelInfo *meta) entityDeleteFunc {
	return func(executor interface{}, entity Entity, condition string, params []interface{}) (int64, error) {
		checkEntity(modelInfo, entity, executor)
		tname, err := tblName(entity)
		if err != nil {
			return 0, err
		}
		delSQL := fmt.Sprintf("DELETE FROM %s ", tname)
		if len(condition) > 0 {
			delSQL += condition
		}

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

func createInsertOrUpdateFunc(modelInfo *meta) entityInsertOrUpdateFunc {
	insertFields := fun.Filter(exceptIDPred, modelInfo.fields).([]*metaField)
	columns := strings.Join(fun.Map(func(field *metaField) string {
		return field.column
	}, insertFields).([]string), ",")

	insertParams := strings.Join(toSlice("?", len(insertFields)), ",")

	updateFields := fun.Filter(noIDPred, modelInfo.fields).([]*metaField)
	updateColumns := strings.Join(fun.Map(func(field *metaField) string {
		return field.column + "=?"
	}, updateFields).([]string), ",")

	return func(executor interface{}, entity Entity) (int64, error) {
		ind := checkEntity(modelInfo, entity, executor)
		paramValues := buildParamValues(ind, insertFields)
		updateParamValues := buildParamValues(ind, updateFields)
		allParamValues := append(paramValues, updateParamValues...)
		tname, err := tblName(entity)
		if err != nil {
			return 0, err
		}
		insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES(%s) ON DUPLICATE KEY UPDATE %s", tname, columns, insertParams, updateColumns)

		rs, err := exec(executor, insertSQL, allParamValues)
		if err != nil {
			return 0, err
		}

		//检查更新的记录数
		rows, err := rs.RowsAffected()
		if err == nil {
			return rows, nil
		}
		return 0, err
	}
}

// tblName 确定表名
func tblName(entity Entity) (string, error) {
	if shardEntity, ok := entity.(ShardEntity); ok {
		if shardEntity.TableShardFunc() != nil {
			tblName, err := shardEntity.TableShardFunc()()
			return tblName, err
		}
	}
	return entity.TableName(), nil
}
