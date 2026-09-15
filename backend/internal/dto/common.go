package dto

const (
	DefaultPage     = 1
	DefaultPageSize = 10
	MaxPageSize     = 100
)

type PageQuery struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Search   string `form:"search" binding:"omitempty,max=255"`
}

func (q *PageQuery) Normalize() {
	if q.Page < 1 {
		q.Page = DefaultPage
	}
	if q.PageSize < 1 {
		q.PageSize = DefaultPageSize
	}
	if q.PageSize > MaxPageSize {
		q.PageSize = MaxPageSize
	}
}

func (q PageQuery) Offset() int { return (q.Page - 1) * q.PageSize }
