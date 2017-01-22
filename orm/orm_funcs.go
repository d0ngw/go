package orm

import (
	"database/sql"
	"fmt"
	"github.com/BurntSushi/ty/fun"
	c "github.com/d0ngw/go/common"
	"reflect"
	"strings"
)

func toSlice(s string, count int) []string {
	slice := make([]string, 0, count)
	for i := 0; i < count; i++ {
		slice = append(slice, s)
	}
	return slice
}

//除了主键的过滤函数
var exceptIdPred = func(field *modelField) bool {
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

func exec(executor interface{}, execSql string, args []interface{}) (rs sql.Result, err error) {
	c.Debugf("Exec sql %s with %T", execSql, executor)
	if tx, ok := executor.(*sql.Tx); ok {
		rs, err = tx.Exec(execSql, args...)
	} else if db, ok := executor.(*sql.DB); ok {
		rs, err = db.Exec(execSql, args...)
	} else {
		panic(NewDBErrorf(nil, "Not a valid executor:%T", executor))
	}
	return
}

func query(executor interface{}, execSql string, args []interface{}) (rows *sql.Rows, err error) {
	c.Debugf("Exec sql %s with %T", execSql, executor)
	if tx, ok := executor.(*sql.Tx); ok {
		rows, err = tx.Query(execSql, args...)
	} else if db, ok := executor.(*sql.DB); ok {
		rows, err = db.Query(execSql, args...)
	} else {
		panic(NewDBErrorf(nil, "Not a valid executor:%T", executor))
	}
	return
}

//构建实体模型的插入函数
func createInsertFunc(modelInfo *modelMeta) EntityCFunc {
	insertFields := fun.Filter(exceptIdPred, modelInfo.fields).([]*modelField)
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

		insertSql := fmt.Sprintf("INSERT INTO %s (%s) VALUES(%s)", entity.TableName(), columns, params)
		c.Debugf("insertSql:%v", insertSql)

		rs, err := exec(executor, insertSql, paramValues)
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
	updateFields := fun.Filter(exceptIdPred, modelInfo.fields).([]*modelField)
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

		updateSql := fmt.Sprintf("UPDATE %s SET %s where %s = %s", entity.TableName(), columns, modelInfo.pkField.column, "?")
		rs, err := exec(executor, updateSql, paramValues)
		if err != nil {
			return false, err
		}

		//检查更新的记录数
		if rows, err := rs.RowsAffected(); err == nil {
			if rows != 1 {
				return false, nil
			} else {
				return true, nil
			}
		} else {
			return false, err
		}
	}
}

//构建实体模型的指定类名的更新函数
func createUpdateColumnsFunc(modelInfo *modelMeta) EntityUCFunc {
	return func(executor interface{}, entity EntityInterface, columns string, condition string, params []interface{}) (int64, error) {
		checkEntity(modelInfo, entity, executor)
		if len(columns) == 0 {
			panic(NewDBError(nil, "Can't update empty columns"))
		}
		updateSql := fmt.Sprintf("UPDATE %s SET %s ", entity.TableName(), columns)
		if len(condition) > 0 {
			updateSql += condition

		}
		c.Debugf("updateSql:%v", updateSql)

		rs, err := exec(executor, updateSql, params)
		if err != nil {
			return 0, err
		}

		//检查更新的记录数
		if rows, err := rs.RowsAffected(); err == nil {
			c.Debugf("Updated rows:%v", rows)
			return rows, err
		} else {
			return 0, err
		}
	}
}

//构建查询函数
func createQueryFunc(modelInfo *modelMeta) EntityQFunc {
	columns := strings.Join(fun.Map(func(field *modelField) string {
		return "`" + field.column + "`"
	}, modelInfo.fields).([]string), ",")

	return func(executor interface{}, entity EntityInterface, condition string, params []interface{}) ([]EntityInterface, error) {
		ind := checkEntity(modelInfo, entity, executor)
		querySql := fmt.Sprintf("SELECT %s FROM %s ", columns, entity.TableName())
		if len(condition) > 0 {
			querySql += condition
		}
		c.Debugf("querySql:%v", querySql)

		rows, err := query(executor, querySql, params)
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
		delSql := fmt.Sprintf("DELETE FROM %s ", entity.TableName())
		if len(condition) > 0 {
			delSql += condition
		}
		c.Debugf("delSql:%v", delSql)

		rs, err := exec(executor, delSql, params)
		if err != nil {
			return 0, err
		}
		//检查更新的记录数
		if rows, err := rs.RowsAffected(); err == nil {
			return rows, err
		} else {
			return 0, err
		}
	}
}
