package query

const (
	DefaultPage     uint32 = 1
	DefaultPageSize uint32 = 10
	MaxPageSize     uint32 = 100
)

// OffsetPaging holds validated pagination params for a list query.
type OffsetPaging struct {
	Page     uint32
	PageSize uint32
}

// NewOffsetPaging creates a normalized OffsetPaging from raw request values.
func NewOffsetPaging(page, pageSize uint32) *OffsetPaging {
	if page < DefaultPage {
		page = DefaultPage
	}
	if pageSize == 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return &OffsetPaging{Page: page, PageSize: pageSize}
}

// Offset returns the row offset for SQL queries.
func (p *OffsetPaging) Offset() int { return int((p.Page - 1) * p.PageSize) }

// Limit returns the page size as int for SQL queries.
func (p *OffsetPaging) Limit() int { return int(p.PageSize) }
