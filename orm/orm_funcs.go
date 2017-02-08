package orm

import (
	"database/sql"
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

//除了主键的过滤函数
var exceptIDPred = func(field *modelField) bool {
	if field == nil || (field.pk && field.pkAuto) {
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

//构建实体模型的插入函数
func createInsertFunc(modelInfo *modelMeta) EntityCFunc {
	insertFields := fun.Filter(exceptIDPred, modelInfo.fields).([]*modelField)
	columns := strings.Join(fun.Map(func(field *modelField) string {
		return field.column
	}, insertFields).([]string), ",")
	params := strings.Join(toSlice("?", len(insertFields)), ",")

	return func(executor interface{}, entity EntityInterface) error {
		ind := checkEntity(modelInfo, entity, executor)

		paramValues := make([]interface{}, 0, len(insertFields))
		for _, field := range insertFields {
			fv := ind.Field(field.index).Interface()
			paramValues = append(paramValues, fv)
		}

		insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES(%s)", entity.TableName(), columns, params)
		c.Debugf("insertSql:%v", insertSQL)

		rs, err := exec(executor, insertSQL, paramValues)
		if err != nil {
			return err
		}

		if modelInfo.pkField.pkAuto {
			if id, err := rs.LastInsertId(); err == nil {
				ind.Field(modelInfo.pkField.index).SetInt(id)
			} else {
				return err
			}
		}
		return nil
	}
}

//构建实体模型的更新函数
func createUpdateFunc(modelInfo *modelMeta) EntityUFunc {
	updateFields := fun.Filter(exceptIDPred, modelInfo.fields).([]*modelField)
	columns := strings.Join(fun.Map(func(field *modelField) string {
		return field.column + "=?"
	}, updateFields).([]string), ",")

	return func(executor interface{}, entity EntityInterface) (bool, error) {
		ind := checkEntity(modelInfo, entity, executor)

		paramValues := make([]interface{}, 0, len(updateFields)+1)
		for _, field := range updateFields {
			fv := ind.Field(field.index).Interface()
			paramValues = append(paramValues, fv)
		}

		id := ind.Field(modelInfo.pkField.index).Interface()
		paramValues = append(paramValues, id)

		updateSQL := fmt.Sprintf("UPDATE %s SET %s where %s = %s", entity.TableName(), columns, modelInfo.pkField.column, "?")
		rs, err := exec(executor, updateSQL, paramValues)
		if err != nil {
			return false, err
		}
		//检查更新的记录数
		if rows, err := rs.RowsAffected(); err == nil {
			if rows != 1 {
				return false, nil
			}
			return true, nil
		}
		return false, err
	}
}

//构建实体模型的指定类名的更新函数
func createUpdateColumnsFunc(modelInfo *modelMeta) EntityUCFunc {
	return func(executor interface{}, entity EntityInterface, columns string, condition string, params []interface{}) (int64, error) {
		checkEntity(modelInfo, entity, executor)
		if len(columns) == 0 {
			panic(NewDBError(nil, "Can't update empty columns"))
		}
		updateSQL := fmt.Sprintf("UPDATE %s SET %s ", entity.TableName(), columns)
		if len(condition) > 0 {
			updateSQL += condition

		}
		c.Debugf("updateSql:%v", updateSQL)

		rs, err := exec(executor, updateSQL, params)
		if err != nil {
			return 0, err
		}

		//检查更新的记录数
		if rows, err := rs.RowsAffected(); err == nil {
			c.Debugf("Updated rows:%v", rows)
			return rows, err
		}
		return 0, err
	}
}

//构建查询函数
func createQueryFunc(modelInfo *modelMeta) EntityQFunc {
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
				fv := ptrValueInd.Field(field.index).Addr().Interface()
				//c.Debugf("fv:%v,type:%T", fv, fv)
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

//构建删除函数
func createDelFunc(modelInfo *modelMeta) EntityDFunc {
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
		if rows, err := rs.RowsAffected(); err == nil {
			return rows, err
		}
		return 0, err
	}
}
