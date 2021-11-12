package orm

import (
	"database/sql"
	"errors"
	"fmt"
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

// Op 数据库操作接口,与sql.DB对应,封装了事务等
type Op struct {
	pool           *Pool          //数据连接
	tx             *sql.Tx        //事务
	txDone         bool           //事务是否结束
	rollbackOnly   bool           //是否只回滚
	transDepth     int            //调用的深度
	sharDBSerevcie ShardDBService //分片服务
}

// DB sql.DB
func (p *Op) DB() *sql.DB {
	return p.pool.db
}

// Pool pool
func (p *Op) Pool() *Pool {
	return p.pool
}

// PoolName name of pool
func (p *Op) PoolName() string {
	return p.pool.name
}

// SetupTableShard use op pool setup entity table shard
func (p *Op) SetupTableShard(entity Entity, ruleName string) error {
	if p.sharDBSerevcie == nil {
		return errors.New("no shard db service")
	}
	poolName, err := p.sharDBSerevcie.setupTableShard(entity, ruleName)
	if err != nil {
		return err
	}
	if poolName != p.PoolName() {
		return fmt.Errorf("op.PoolName %s != entity.PoolName %s", p.PoolName(), poolName)
	}
	return nil
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
}

func (p *Op) decrTransDepth() error {
	p.transDepth = p.transDepth - 1
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
		return p.tx.Rollback()
	}
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
		p.txDone = false
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
func findEntityMeta(entity Entity) *meta {
	_, _, typ := extract(entity)
	modelMeta := findMeta(typ)
	if modelMeta == nil {
		panic(NewDBErrorf(nil, "Can't find modelMeta for:%v ", typ))
	}
	return modelMeta
}

// Add 添加实体
func Add(op *Op, entity Entity) error {
	modelMeta := findEntityMeta(entity)
	if op.tx != nil {
		return modelMeta.insertFunc(op.tx, entity)
	}
	return modelMeta.insertFunc(op.DB(), entity)
}

// Update 更新实体
func Update(op *Op, entity Entity) (bool, error) {
	modelMeta := findEntityMeta(entity)
	if op.tx != nil {
		bvalue, err := modelMeta.updateFunc(op.tx, entity)
		if err != nil {
			return false, err
		}
		return reflect.ValueOf(bvalue).Bool(), nil
	}
	return modelMeta.updateFunc(op.DB(), entity)
}

// UpdateReplace 更新实体
func UpdateReplace(op *Op, entity Entity, replColumns map[string]ReplColumn, excludeColumns map[string]struct{}) (bool, error) {
	modelMeta := findEntityMeta(entity)
	if op.tx != nil {
		bvalue, err := modelMeta.updateReplaceFunc(op.tx, entity, replColumns, excludeColumns)
		if err != nil {
			return false, err
		}
		return reflect.ValueOf(bvalue).Bool(), nil
	}
	return modelMeta.updateReplaceFunc(op.DB(), entity, replColumns, excludeColumns)
}

// UpdateExcludeColumns 更新除columns之外的字段
func UpdateExcludeColumns(op *Op, entity Entity, columns ...string) (bool, error) {
	modelMeta := findEntityMeta(entity)
	if op.tx != nil {
		bvalue, err := modelMeta.updateExcludeColumnsFunc(op.tx, entity, columns...)
		if err != nil {
			return false, err
		}
		return reflect.ValueOf(bvalue).Bool(), nil
	}
	return modelMeta.updateExcludeColumnsFunc(op.DB(), entity, columns...)
}

// UpdateColumns 更新列
func UpdateColumns(op *Op, entity Entity, columns string, condition string, params ...interface{}) (int64, error) {
	modelMeta := findEntityMeta(entity)
	if op.tx != nil {
		return modelMeta.updateColumnsFunc(op.tx, entity, columns, condition, params)
	}
	return modelMeta.updateColumnsFunc(op.DB(), entity, columns, condition, params)
}

// Get 根据ID查询实体
func Get(op *Op, entity Entity, id interface{}) (Entity, error) {
	modelMeta := findEntityMeta(entity)
	if op.tx != nil {
		e, err := modelMeta.getFunc(op.tx, entity, id)
		if e == nil || err != nil {
			return nil, err
		}
		return e.(Entity), nil
	}
	return modelMeta.getFunc(op.DB(), entity, id)
}

// Query 根据条件查询实体
func Query(op *Op, entity Entity, condition string, params ...interface{}) ([]Entity, error) {
	modelMeta := findEntityMeta(entity)
	if op.tx != nil {
		return modelMeta.entityQueryFunc(op.tx, entity, condition, params)
	}
	return modelMeta.entityQueryFunc(op.DB(), entity, condition, params)
}

// QueryColumns 根据条件查询columns指定的字段
func QueryColumns(op *Op, entity Entity, columns []string, condition string, params ...interface{}) ([]Entity, error) {
	modelMeta := findEntityMeta(entity)
	if op.tx != nil {
		return modelMeta.entityQueryColumnFunc(op.tx, entity, columns, condition, params)
	}
	return modelMeta.entityQueryColumnFunc(op.DB(), entity, columns, condition, params)
}

type count struct {
	Count int64
}

// QueryCount 根据条件查询条数
func QueryCount(op *Op, entity Entity, column string, condition string, params ...interface{}) (num int64, err error) {
	modelMeta := findEntityMeta(entity)
	columns := []string{"count(" + column + ")"}
	var counts []*count
	if op.tx != nil {
		err = modelMeta.clumnsQueryFunc(op.tx, entity, &counts, columns, condition, params)
	} else {
		err = modelMeta.clumnsQueryFunc(op.DB(), entity, &counts, columns, condition, params)
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
func QueryColumnsForDestSlice(op *Op, entity Entity, destSlicePtr interface{}, columns []string, condition string, params ...interface{}) (err error) {
	modelMeta := findEntityMeta(entity)
	if op.tx != nil {
		err = modelMeta.clumnsQueryFunc(op.tx, entity, destSlicePtr, columns, condition, params)
	} else {
		err = modelMeta.clumnsQueryFunc(op.DB(), entity, destSlicePtr, columns, condition, params)
	}
	if err != nil {
		return
	}
	return
}

// Del 根据ID删除实体
func Del(op *Op, entity Entity, id interface{}) (bool, error) {
	modelMeta := findEntityMeta(entity)
	if op.tx != nil {
		return modelMeta.delEFunc(op.tx, entity, id)
	}
	return modelMeta.delEFunc(op.DB(), entity, id)
}

// DelByCondition 根据条件删除
func DelByCondition(op *Op, entity Entity, condition string, params ...interface{}) (int64, error) {
	modelMeta := findEntityMeta(entity)
	if op.tx != nil {
		return modelMeta.delFunc(op.tx, entity, condition, params)
	}
	return modelMeta.delFunc(op.DB(), entity, condition, params)
}

// AddOrUpdate 添加或者更新实体(如果id已经存在),只支持MySql
func AddOrUpdate(op *Op, entity Entity) (int64, error) {
	modelMeta := findEntityMeta(entity)
	if op.tx != nil {
		return modelMeta.insertOrUpdateFunc(op.tx, entity)
	}
	return modelMeta.insertOrUpdateFunc(op.DB(), entity)
}
