package orm

import (
	"database/sql"
	"reflect"

	c "github.com/d0ngw/go/common"
)

// OpTxFunc 在事务中处理的函数
type OpTxFunc func(tx *sql.Tx) (interface{}, error)

// OpCreator Op
type OpCreator interface {
	//NewOp create a new Op
	NewOp() (*Op, error)
}

// Op 数据库操作接口
type Op struct {
	pool         *Pool   //数据连接
	tx           *sql.Tx //事务
	txDone       bool    //事务是否结束
	rollbackOnly bool    //是否只回滚
	transDepth   int     //调用的深度
}

// DB sql.DB
func (p *Op) DB() *sql.DB {
	return p.pool.db
}

// Pool pool
func (p *Op) Pool() *Pool {
	return p.pool
}

func (p *Op) close() {
	p.tx = nil
	p.rollbackOnly = false
	p.transDepth = 0
}

//检查事务的状态
func (p *Op) checkTransStatus() error {
	if p.txDone {
		return sql.ErrTxDone
	}
	if p.tx == nil {
		return NewDBError(nil, "Not begin transaction")
	}
	return nil
}

func (p *Op) incrTransDepth() {
	p.transDepth = p.transDepth + 1
	c.Debugf("p.tranDepth:%v", p.transDepth)
}

func (p *Op) decrTransDepth() error {
	p.transDepth = p.transDepth - 1
	c.Debugf("p.tranDepth:%v", p.transDepth)
	if p.transDepth < 0 {
		return NewDBError(nil, "Too many invoke commit or rollback")
	}
	return nil
}

//结束事务
func (p *Op) finishTrans() error {
	if err := p.checkTransStatus(); err != nil {
		return err
	}
	if err := p.decrTransDepth(); err != nil {
		return err
	}
	if p.transDepth > 0 {
		return nil
	}
	defer p.close()
	p.txDone = true
	if p.rollbackOnly {
		c.Debugf("Rollback")
		return p.tx.Rollback()
	}
	c.Debugf("Commit")
	return p.tx.Commit()
}

// BeginTx 开始事务,支持简单的嵌套调用,如果已经开始了事务,则直接返回成功
func (p *Op) BeginTx() (err error) {
	p.incrTransDepth()
	if p.tx != nil {
		return nil //事务已经开启
	}
	if tx, err := p.DB().Begin(); err == nil {
		p.tx = tx
		return nil
	}
	return err
}

// Commit 提交事务
func (p *Op) Commit() error {
	return p.finishTrans()
}

// Rollback 回滚事务
func (p *Op) Rollback() error {
	p.SetRollbackOnly(true)
	return p.finishTrans()
}

// SetRollbackOnly 设置只回滚
func (p *Op) SetRollbackOnly(rollback bool) {
	p.rollbackOnly = rollback
}

// IsRollbackOnly 是否只回滚
func (p *Op) IsRollbackOnly() bool {
	return p.rollbackOnly
}

// DoInTrans 在事务中执行
func (p *Op) DoInTrans(peration OpTxFunc) (rt interface{}, err error) {
	if err := p.BeginTx(); err != nil {
		return nil, err
	}
	var succ = false
	//结束事务
	defer func() {
		if !succ {
			p.SetRollbackOnly(true)
		}
		transErr := p.finishTrans()
		if transErr != nil {
			c.Errorf("Finish transaction err:%v", transErr)
			rt = nil
			err = transErr
		}
	}()
	rt, err = peration(p.tx)
	if err != nil {
		c.Errorf("Operation fail:%v", err)
		succ = false
	} else {
		succ = true
	}
	return
}

//查找实体对应的模型元
func getEntityModelInfo(entity Entity) *meta {
	_, _, typ := extract(entity)
	modelInfo := findModelInfo(typ)
	if modelInfo == nil {
		panic(NewDBErrorf(nil, "Can't find modelInfo for:%v ", typ))
	}
	return modelInfo
}

// Add 添加实体
func Add(dbOper *Op, entity Entity) error {
	modelInfo := getEntityModelInfo(entity)
	if dbOper.tx != nil {
		return modelInfo.insertFunc(dbOper.tx, entity)
	}
	return modelInfo.insertFunc(dbOper.DB(), entity)
}

// Update 更新实体
func Update(dbOper *Op, entity Entity) (bool, error) {
	modelInfo := getEntityModelInfo(entity)
	if dbOper.tx != nil {
		bvalue, err := modelInfo.updateFunc(dbOper.tx, entity)
		if err != nil {
			return false, err
		}
		return reflect.ValueOf(bvalue).Bool(), nil
	}
	return modelInfo.updateFunc(dbOper.DB(), entity)
}

// UpdateColumns 更新列
func UpdateColumns(dbOper *Op, entity Entity, columns string, condition string, params ...interface{}) (int64, error) {
	modelInfo := getEntityModelInfo(entity)
	if dbOper.tx != nil {
		return modelInfo.updateColumnsFunc(dbOper.tx, entity, columns, condition, params)
	}
	return modelInfo.updateColumnsFunc(dbOper.DB(), entity, columns, condition, params)
}

// Get 根据ID查询实体
func Get(dbOper *Op, entity Entity, id interface{}) (Entity, error) {
	modelInfo := getEntityModelInfo(entity)
	if dbOper.tx != nil {
		e, err := modelInfo.getFunc(dbOper.tx, entity, id)
		if e == nil || err != nil {
			return nil, err
		}
		return e.(Entity), nil
	}
	return modelInfo.getFunc(dbOper.DB(), entity, id)
}

// Query 根据条件查询实体
func Query(dbOper *Op, entity Entity, condition string, params ...interface{}) ([]Entity, error) {
	modelInfo := getEntityModelInfo(entity)
	if dbOper.tx != nil {
		return modelInfo.entityQueryFunc(dbOper.tx, entity, condition, params)
	}
	return modelInfo.entityQueryFunc(dbOper.DB(), entity, condition, params)
}

// QueryColumns 根据条件查询columns指定的字段
func QueryColumns(dbOper *Op, entity Entity, columns []string, condition string, params ...interface{}) ([]Entity, error) {
	modelInfo := getEntityModelInfo(entity)
	if dbOper.tx != nil {
		return modelInfo.entityQueryColumnFunc(dbOper.tx, entity, columns, condition, params)
	}
	return modelInfo.entityQueryColumnFunc(dbOper.DB(), entity, columns, condition, params)
}

type count struct {
	Count int64
}

// QueryCount 根据条件查询条数
func QueryCount(dbOper *Op, entity Entity, column string, condition string, params ...interface{}) (num int64, err error) {
	modelInfo := getEntityModelInfo(entity)
	columns := []string{"count(" + column + ")"}
	var counts []*count
	if dbOper.tx != nil {
		err = modelInfo.clumnsQueryFunc(dbOper.tx, entity, &counts, columns, condition, params)
	} else {
		err = modelInfo.clumnsQueryFunc(dbOper.DB(), entity, &counts, columns, condition, params)
	}
	if err != nil {
		return
	}
	if len(counts) > 0 {
		num = counts[0].Count
	}
	return
}

// QueryColumnsForDestSlice 根据条件查询数据,结果保存到destSlicePtr
func QueryColumnsForDestSlice(dbOper *Op, entity Entity, destSlicePtr interface{}, columns []string, condition string, params ...interface{}) (err error) {
	modelInfo := getEntityModelInfo(entity)
	if dbOper.tx != nil {
		err = modelInfo.clumnsQueryFunc(dbOper.tx, entity, destSlicePtr, columns, condition, params)
	} else {
		err = modelInfo.clumnsQueryFunc(dbOper.DB(), entity, destSlicePtr, columns, condition, params)
	}
	if err != nil {
		return
	}
	return
}

// Del 根据ID删除实体
func Del(dbOper *Op, entity Entity, id interface{}) (bool, error) {
	modelInfo := getEntityModelInfo(entity)
	if dbOper.tx != nil {
		return modelInfo.delEFunc(dbOper.tx, entity, id)
	}
	return modelInfo.delEFunc(dbOper.DB(), entity, id)
}

// DelByCondition 根据条件删除
func DelByCondition(dbOper *Op, entity Entity, condition string, params ...interface{}) (int64, error) {
	modelInfo := getEntityModelInfo(entity)
	if dbOper.tx != nil {
		return modelInfo.delFunc(dbOper.tx, entity, condition, params)
	}
	return modelInfo.delFunc(dbOper.DB(), entity, condition, params)
}

// AddOrUpdate 添加或者更新实体(如果id已经存在),只支持MySql
func AddOrUpdate(dbOper *Op, entity Entity) (int64, error) {
	modelInfo := getEntityModelInfo(entity)
	if dbOper.tx != nil {
		return modelInfo.insertOrUpdateFunc(dbOper.tx, entity)
	}
	return modelInfo.insertOrUpdateFunc(dbOper.DB(), entity)
}
