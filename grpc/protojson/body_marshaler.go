// Package protojson provides JSON marshaling options for grpc-gateway.
//
// [NewGeaMarshaler] returns a [runtime.Marshaler] configured for API-friendly JSON:
//   - proto field names (snake_case) rather than camelCase
//   - enum values as strings rather than numbers
//   - zero-value fields included in output (EmitUnpopulated)
//   - unknown JSON fields silently discarded on input
//
// [NewGeaQueryParser] returns a query-string parser that maps URL parameters to
// proto fields using the same snake_case conventions as the JSON marshaler.
package protojson

import (
	"errors"
	"fmt"
	"io"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/protobuf/encoding/protojson"
)

// DefaultMarshaler return the marshaler option with support serialization data with json_name specific.
func DefaultMarshaler() runtime.Marshaler {
	return &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			Multiline:       false,
			Indent:          "",
			AllowPartial:    false,
			UseProtoNames:   true,
			UseEnumNumbers:  false,
			EmitUnpopulated: true,
			Resolver:        nil,
		},
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	}
}

// NewGeaMarshaler return the marshaler option with support serialization data with json_name specific.
func NewGeaMarshaler() runtime.Marshaler {
	return &GeaMarshaler{Marshaler: DefaultMarshaler()}
}

type GeaMarshaler struct {
	runtime.Marshaler
}

// Marshal marshals "v" into byte sequence.
func (m *GeaMarshaler) Marshal(v any) ([]byte, error) {
	return wrapBodyMarshalerRetAndError(m.Marshaler.Marshal(v))
}

// Unmarshal unmarshals "data" into "v".
// "v" must be a pointer value.
func (m *GeaMarshaler) Unmarshal(data []byte, v any) error {
	return wrapBodyMarshalerError(m.Marshaler.Unmarshal(data, v))
}

// NewDecoder returns a Decoder which reads byte sequence from "r".
func (m *GeaMarshaler) NewDecoder(r io.Reader) runtime.Decoder {
	return newGeaDecoder(m.Marshaler.NewDecoder(r))
}

// NewEncoder returns an Encoder which writes bytes sequence into "w".
func (m *GeaMarshaler) NewEncoder(w io.Writer) runtime.Encoder {
	return newGeaEncoder(m.Marshaler.NewEncoder(w))
}

type GeaDecoder struct {
	runtime.Decoder
}

func (e *GeaDecoder) Decode(v any) error {
	return wrapBodyMarshalerError(e.Decoder.Decode(v))
}

func newGeaDecoder(decoder runtime.Decoder) *GeaDecoder {
	return &GeaDecoder{Decoder: decoder}
}

type GeaEncoder struct {
	runtime.Encoder
}

func (e *GeaEncoder) Encode(v any) error {
	return wrapBodyMarshalerError(e.Encoder.Encode(v))
}

func newGeaEncoder(encoder runtime.Encoder) *GeaEncoder {
	return &GeaEncoder{Encoder: encoder}
}

var ErrBodyMarshalerPrefix = errors.New("body marshaler")

func wrapBodyMarshalerError(err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%w: %w", ErrBodyMarshalerPrefix, err)
}

func wrapBodyMarshalerRetAndError[T any](ret T, err error) (T, error) {
	return ret, wrapBodyMarshalerError(err)
}
