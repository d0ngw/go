package common

//PageParam 分页参数
type PageParam struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

//Limit 根据maxPage和maxPageSize限制Page和PageSize
func (p *PageParam) Limit(maxPage, maxPageSize int) {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 10
	}
	if maxPage > 0 && p.Page > maxPage {
		p.Page = maxPage
	}
	if maxPageSize > 0 && p.PageSize > maxPageSize {
		p.PageSize = maxPageSize
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

// ResultSet is the result set with total and items
type ResultSet interface {
	SetTotal(total int64)
	SetItems(items []interface{})
}

//PageResult 分页结果
type PageResult struct {
	PageParam
	Total int64 `json:"total"`
	Items []interface{}
}

// SetTotal implements ResultSet.SetTotal
func (p *PageResult) SetTotal(total int64) {
	p.Total = total
}

// SetItems  implements ResultSet.SetItems
func (p *PageResult) SetItems(items []interface{}) {
	p.Items = items
}

//Query 基本的查询参数
type Query struct {
	PageParam
	ID int64
}
