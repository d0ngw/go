package common

import "reflect"

//PageParam 分页参数
type PageParam struct {
	//页数,从1开始
	Page int `json:"page"`
	//每页的条数,>0
	PageSize int `json:"page_size"`
	//游标
	Cursor int64 `json:"cursor"`
}

//Limit 根据maxPage和maxPageSize限制Page和PageSize
func (p *PageParam) Limit(maxPage, maxPageSize int) {
	if p.Page <= 0 {
		p.Page = 1
	}
	if maxPage > 0 && p.Page > maxPage {
		p.Page = maxPage
	}
	if maxPageSize > 0 && (p.PageSize > maxPageSize || p.PageSize <= 0) {
		p.PageSize = maxPageSize
	}
	if p.PageSize <= 0 {
		p.PageSize = 10
	}
}

//StartIndex 返回从0开始的起始索引
func (p *PageParam) StartIndex() int {
	return (p.Page - 1) * p.PageSize
}

//EndIndex 返回从0开始的截止索引
func (p *PageParam) EndIndex() int {
	return p.StartIndex() + p.PageSize - 1
}

// PageResultItemsSetter 结果设置
type PageResultItemsSetter func(sum int, index int, elem interface{})

// ResultSet is the result set with total and items
type ResultSet interface {
	SetTotal(total int64)
	SetData(data interface{}, itemsSetter PageResultItemsSetter)
	GetItemsSetter() PageResultItemsSetter
	CalTotalPage()
}

//PageResult 分页结果
type PageResult struct {
	PageParam
	Total     int64       `json:"total"`
	TotalPage int64       `json:"totalPage"`
	Items     interface{} `json:"items"`
}

// SetTotal implements ResultSet.SetTotal
func (p *PageResult) SetTotal(total int64) {
	p.Total = total
}

// SetData implements ResultSet.SetData
func (p *PageResult) SetData(data interface{}, itemsSetter PageResultItemsSetter) {
	p.Items = data

	if itemsSetter == nil {
		return
	}

	sliceVal, _, _ := ExtractRefTuple(data)
	if sliceVal.Kind() != reflect.Slice {
		return
	}

	itemsLen := sliceVal.Len()

	for i := 0; i < itemsLen; i++ {
		itemsSetter(itemsLen, i, sliceVal.Index(i).Interface())
	}
}

// CalTotalPage 计算总页数
func (p *PageResult) CalTotalPage() {
	if p.PageSize > 0 {
		if p.Total%int64(p.PageSize) == 0 {
			p.TotalPage = p.Total / int64(p.PageSize)
		} else {
			p.TotalPage = p.Total/int64(p.PageSize) + 1
		}
	}
}

//Query 基本的查询参数
type Query struct {
	PageParam
	//ID
	ID int64 `json:"id"`
}
