package dto

// Gioi han phan trang de tranh client keo ca bang.
const (
	DefaultPage     = 1
	DefaultPageSize = 10
	MaxPageSize     = 100
)

// PageQuery la tham so phan trang + tim kiem tren query string.
type PageQuery struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	Search   string `form:"search" binding:"omitempty,max=255"`
}

// Normalize dien gia tri mac dinh cho tham so bi thieu hoac vuot gioi han.
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

// Offset tra ve vi tri bat dau cho cau lenh SQL.
func (q PageQuery) Offset() int { return (q.Page - 1) * q.PageSize }
