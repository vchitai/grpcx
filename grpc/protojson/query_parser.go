package protojson

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/grpc-ecosystem/grpc-gateway/v2/utilities"
	"google.golang.org/protobuf/proto"
)

// DefaultQueryParser returns the standard grpc-gateway query parameter parser.
func DefaultQueryParser() runtime.QueryParameterParser {
	return &runtime.DefaultQueryParser{}
}

// NewGeaQueryParser return a customized query parser.
func NewGeaQueryParser() runtime.QueryParameterParser {
	return &GeaQueryParser{QueryParameterParser: DefaultQueryParser()}
}

type GeaQueryParser struct {
	runtime.QueryParameterParser
}

func (qp *GeaQueryParser) Parse(msg proto.Message, values url.Values, filter *utilities.DoubleArray) error {
	return wrapQueryParserError(qp.QueryParameterParser.Parse(msg, values, filter))
}

var ErrQueryParserPrefix = errors.New("query parser")

func wrapQueryParserError(err error) error {
	if err == nil {
		return nil
	}

	// move to errors.Join when it's stable
	return fmt.Errorf("%w: %w", ErrQueryParserPrefix, err)
}
