package query_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vchitai/grpcx/query"
)

func TestNewOffsetPaging_defaults(t *testing.T) {
	p := query.NewOffsetPaging(0, 0)
	assert.Equal(t, query.DefaultPage, p.Page)
	assert.Equal(t, query.DefaultPageSize, p.PageSize)
}

func TestNewOffsetPaging_caps(t *testing.T) {
	p := query.NewOffsetPaging(1, 9999)
	assert.Equal(t, query.MaxPageSize, p.PageSize)
}

func TestNewOffsetPaging_normal(t *testing.T) {
	p := query.NewOffsetPaging(3, 20)
	assert.Equal(t, uint32(3), p.Page)
	assert.Equal(t, uint32(20), p.PageSize)
	assert.Equal(t, 40, p.Offset())
	assert.Equal(t, 20, p.Limit())
}

func TestNewOffsetPaging_firstPage(t *testing.T) {
	p := query.NewOffsetPaging(1, 10)
	assert.Equal(t, 0, p.Offset())
}

func TestNewOrder_defaults(t *testing.T) {
	o := query.NewOrder("", "")
	assert.Equal(t, "created_at", o.Column)
	assert.Equal(t, query.Asc, o.Direction)
}

func TestNewOrder_desc(t *testing.T) {
	o := query.NewOrder("name", query.Desc)
	assert.Equal(t, "name DESC NULLS LAST", o.String())
}

func TestNewOrder_asc(t *testing.T) {
	o := query.NewOrder("created_at", query.Asc)
	assert.Equal(t, "created_at ASC NULLS FIRST", o.String())
}

func TestParseDirection(t *testing.T) {
	assert.Equal(t, query.Desc, query.ParseDirection("desc"))
	assert.Equal(t, query.Desc, query.ParseDirection("DESC"))
	assert.Equal(t, query.Asc, query.ParseDirection("asc"))
	assert.Equal(t, query.Asc, query.ParseDirection("anything"))
}
