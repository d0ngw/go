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

// NewDBOper 创建数据库操作接口
func NewDBOper(db *sql.DB) *DBOper {
	return &DBOper{db: db}
}

//在事务中处理的函数
type DBOperTxFunc func(tx *sql.Tx) (interface{}, error)

func (p *DBOper) close() {
	p.tx = nil
	p.rollbackOnly = false
	p.transDepth = 0
}

//检查事务的状态
func (p *DBOper) checkTransStatus() error {
	if p.txDone {
		return sql.ErrTxDone
	}
	if p.tx == nil {
		return NewDBError(nil, "Not begin transaction")
	}
	return nil
}

func (p *DBOper) incrTransDepth() {
	p.transDepth = p.transDepth + 1
	c.Debugf("p.tranDepth:%v", p.transDepth)
}

func (p *DBOper) decrTransDepth() error {
	p.transDepth = p.transDepth - 1
	c.Debugf("p.tranDepth:%v", p.transDepth)
	if p.transDepth < 0 {
		return NewDBError(nil, "Too many invoke commit or rollback")
	}
	return nil
}

//结束事务
func (p *DBOper) finishTrans() error {
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
	} else {
		c.Debugf("Commit")
		return p.tx.Commit()
	}
}

//开始事务,支持简单的嵌套调用,如果已经开始了事务,则直接返回成功
func (p *DBOper) BeginTx() error {
	p.incrTransDepth()
	if p.tx != nil {
		return nil //事务已经开启
	}
	if tx, err := p.db.Begin(); err == nil {
		p.tx = tx
		return nil
	} else {
		return err
	}
}

//提交事务
func (p *DBOper) Commit() error {
	return p.finishTrans()
}

//回滚事务
func (p *DBOper) Rollback() error {
	p.SetRollbackOnly(true)
	return p.finishTrans()
}

//设置只回滚
func (p *DBOper) SetRollbackOnly(rollback bool) {
	p.rollbackOnly = rollback
}

//是否只回滚
func (p *DBOper) IsRollbackOnly() bool {
	return p.rollbackOnly
}

//在事务中执行
func (p *DBOper) DoInTrans(peration DBOperTxFunc) (rt interface{}, err error) {
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
		if err != nil {
			return false, err
		} else {
			return reflect.ValueOf(bvalue).Bool(), nil
		}
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
		if e == nil || err != nil {
			return nil, err
		} else {
			return e.(EntityInterface), nil
		}
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
