package query

import "strings"

type Direction string

const (
	Asc  Direction = "ASC"
	Desc Direction = "DESC"
)

// Order holds a column + direction for ORDER BY clauses.
//
// Column is used directly in SQL strings and must come from a server-side
// allowlist — never from raw user input, as it is not escaped.
type Order struct {
	Column    string
	Direction Direction
}

// NewOrder creates an Order, falling back to safe defaults for empty/invalid inputs.
func NewOrder(column string, dir Direction) Order {
	if column == "" {
		column = "created_at"
	}
	if dir != Asc && dir != Desc {
		dir = Asc
	}
	return Order{Column: column, Direction: dir}
}

// String returns the ORDER BY fragment, e.g. "created_at ASC NULLS FIRST".
func (o Order) String() string {
	if o.Direction == Desc {
		return o.Column + " DESC NULLS LAST"
	}
	return o.Column + " ASC NULLS FIRST"
}

// ParseDirection converts a string to Direction, defaulting to Asc.
func ParseDirection(s string) Direction {
	if strings.EqualFold(s, "desc") {
		return Desc
	}
	return Asc
}
