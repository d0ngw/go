package common

// PageParam contains pagination parameters.
type PageParam struct {
	// Page number, starting at 1.
	Page int `json:"page"`
	// Number of items per page; must be positive.
	PageSize int `json:"page_size"`
	// Cursor.
	Cursor int64 `json:"cursor"`
}

// Limit bounds Page and PageSize by the supplied maximum values.
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

// StartIndex returns the zero-based starting index.
func (p *PageParam) StartIndex() int {
	return (p.Page - 1) * p.PageSize
}

// EndIndex returns the zero-based end index.
func (p *PageParam) EndIndex() int {
	return p.StartIndex() + p.PageSize - 1
}

// PageResult contains paginated query results.
type PageResult[T any] struct {
	PageParam
	Total     int64 `json:"total"`
	TotalPage int64 `json:"totalPage"`
	Items     []T   `json:"items"`
}

// SetTotal set total
func (p *PageResult[T]) SetTotal(total int64) {
	p.Total = total
}

// SetData implements ResultSet.SetData
func (p *PageResult[T]) SetData(items []T) {
	p.Items = make([]T, 0, len(items))
	p.Items = append(p.Items, items...)
}

// CalTotalPage computes the total number of pages.
func (p *PageResult[T]) CalTotalPage() {
	if p.PageSize > 0 {
		if p.Total%int64(p.PageSize) == 0 {
			p.TotalPage = p.Total / int64(p.PageSize)
		} else {
			p.TotalPage = p.Total/int64(p.PageSize) + 1
		}
	}
}

// CopyPageResult copy PageResult from src to dest (no items)
func CopyPageResult[S any, T any](src PageResult[S], dest *PageResult[T]) {
	dest.PageParam = src.PageParam
	dest.Total = src.Total
	dest.TotalPage = src.TotalPage
}

// Query contains common query parameters.
type Query struct {
	PageParam
	//ID
	ID int64 `json:"id"`
}
