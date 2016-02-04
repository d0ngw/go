package orm

import (
	"database/sql"
	c "github.com/d0ngw/go/common"
	"reflect"
)

//数据库操作接口
type DBOper struct {
	db           *sql.DB //数据连接
	tx           *sql.Tx //事务
	txDone       bool    //事务是否结束
	rollbackOnly bool    //是否只回滚
	transDepth   int     //调用的深度
}

//
func NewDBOper(db *sql.DB) *DBOper {
	return &DBOper{db: db}
}

//在事务中处理的函数
type DBOperTxFunc func(tx *sql.Tx) (interface{}, error)

func (op *DBOper) close() {
	op.tx = nil
	op.rollbackOnly = false
	op.transDepth = 0
}

//检查事务的状态
func (op *DBOper) checkTransStatus() error {
	if op.txDone {
		return sql.ErrTxDone
	}
	if op.tx == nil {
		panic(&DBError{"Not begin transaction", nil})
	}
	return nil
}

func (op *DBOper) incrTransDepth() {
	op.transDepth = op.transDepth + 1
	c.Debugf("op.tranDepth:%v", op.transDepth)
}

func (op *DBOper) decrTransDepth() {
	op.transDepth = op.transDepth - 1
	c.Debugf("op.tranDepth:%v", op.transDepth)
	if op.transDepth < 0 {
		panic(NewDBError(nil, "Too many invoke commit or rollback"))
	}
}

//结束事务
func (op *DBOper) finishTrans() error {
	if err := op.checkTransStatus(); err != nil {
		return err
	}
	op.decrTransDepth()
	if op.transDepth > 0 {
		return nil
	}
	defer op.close()
	op.txDone = true
	if op.rollbackOnly {
		c.Debugf("Rollback")
		return op.tx.Rollback()
	} else {
		c.Debugf("Commit")
		return op.tx.Commit()
	}
}

func (op *DBOper) ReSet() {
	op.close()
	op.txDone = false
}

//开始事务,支持简单的嵌套调用,如果已经开始了事务,则直接返回成功
func (op *DBOper) BeginTx() error {
	op.incrTransDepth()
	if op.tx != nil {
		return nil //事务已经开启
	}
	if tx, err := op.db.Begin(); err == nil {
		op.tx = tx
		return nil
	} else {
		return err
	}
}

//提交事务
func (op *DBOper) Commit() error {
	return op.finishTrans()
}

//回滚事务
func (op *DBOper) Rollback() error {
	op.SetRollbackOnly(true)
	return op.finishTrans()
}

//设置只回滚
func (op *DBOper) SetRollbackOnly(rollback bool) {
	op.rollbackOnly = rollback
}

//是否只回滚
func (op *DBOper) IsRollbackOnly() bool {
	return op.rollbackOnly
}

//在事务中执行
func (op *DBOper) DoInTrans(operation DBOperTxFunc) (rt interface{}, err error) {
	if err := op.BeginTx(); err != nil {
		return nil, err
	}
	var succ = false
	//结束事务
	defer func() {
		if !succ {
			op.SetRollbackOnly(true)
		}
		transErr := op.finishTrans()
		if transErr != nil {
			c.Errorf("Finish transaction erro:%v", transErr)
			rt = nil
			err = transErr
		}
	}()
	rt, err = operation(op.tx)
	if err != nil {
		c.Errorf("Operation fail:%v", err)
		succ = false
	} else {
		succ = true
	}
	return
}

//查找实体对应的模型元
func getEntityModelInfo(entity EntityInterface) *modelMeta {
	_, _, typ := extract(entity)
	modelInfo := findModelInfo(typ)
	if modelInfo == nil {
		panic(NewDBErrorf(nil, "Can't find modelInfo for:%v ", typ))
	}
	return modelInfo
}

//添加实体
func Add(dbOper *DBOper, entity EntityInterface) error {
	modelInfo := getEntityModelInfo(entity)
	if dbOper.tx != nil {
		return modelInfo.insertFunc(dbOper.tx, entity)
	} else {
		return modelInfo.insertFunc(dbOper.db, entity)
	}
}

//更新实体
func Update(dbOper *DBOper, entity EntityInterface) (bool, error) {
	modelInfo := getEntityModelInfo(entity)
	if dbOper.tx != nil {
		bvalue, err := modelInfo.updateFunc(dbOper.tx, entity)
		return reflect.ValueOf(bvalue).Bool(), err
	} else {
		return modelInfo.updateFunc(dbOper.db, entity)
	}
}

//更新列
func UpdateColumns(dbOper *DBOper, entity EntityInterface, columns string, condition string, params ...interface{}) (int64, error) {
	modelInfo := getEntityModelInfo(entity)
	if dbOper.tx != nil {
		return modelInfo.updateColumnsFunc(dbOper.tx, entity, columns, condition, params)
	} else {
		return modelInfo.updateColumnsFunc(dbOper.db, entity, columns, condition, params)
	}
}

//根据ID查询实体
func Get(dbOper *DBOper, entity EntityInterface, id int64) (EntityInterface, error) {
	modelInfo := getEntityModelInfo(entity)
	if dbOper.tx != nil {
		e, err := modelInfo.getFunc(dbOper.tx, entity, id)
		return e.(EntityInterface), err
	} else {
		return modelInfo.getFunc(dbOper.db, entity, id)
	}
}

//根据条件查询实体
func Query(dbOper *DBOper, entity EntityInterface, condition string, params ...interface{}) ([]EntityInterface, error) {
	modelInfo := getEntityModelInfo(entity)
	if dbOper.tx != nil {
		return modelInfo.queryFunc(dbOper.tx, entity, condition, params)
	} else {
		return modelInfo.queryFunc(dbOper.db, entity, condition, params)
	}
}

//根据ID删除实体
func Del(dbOper *DBOper, entity EntityInterface, id int64) (bool, error) {
	modelInfo := getEntityModelInfo(entity)
	if dbOper.tx != nil {
		return modelInfo.delEFunc(dbOper.tx, entity, id)
	} else {
		return modelInfo.delEFunc(dbOper.db, entity, id)
	}
}

//根据条件删除
func DelByCondition(dbOper *DBOper, entity EntityInterface, condition string, params ...interface{}) (int64, error) {
	modelInfo := getEntityModelInfo(entity)
	if dbOper.tx != nil {
		return modelInfo.delFunc(dbOper.tx, entity, condition, params)
	} else {
		return modelInfo.delFunc(dbOper.db, entity, condition, params)
	}
}
