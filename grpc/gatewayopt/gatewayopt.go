// Package gatewayopt provides additional grpc-gateway ServeMuxOption helpers.
//
// [FormURLEncodedMarshaler] returns a [runtime.ServeMuxOption] that teaches the
// gateway to decode application/x-www-form-urlencoded request bodies as if they
// were JSON — useful for HTML form submissions hitting gRPC-backed endpoints.
//
// [Redirect] returns a [runtime.ServeMuxOption] that lets gRPC handlers trigger
// HTTP redirects by setting Redirect and (optionally) Redirect-Code metadata:
//
//	grpc.SendHeader(ctx, metadata.Pairs("Redirect", "https://example.com", "Redirect-Code", "308"))
package gatewayopt

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/grpc-ecosystem/grpc-gateway/v2/utilities"
	"google.golang.org/protobuf/proto"
)

// Redirect support redirect endpoint for GRPC gateway by using server metadata
// Example:
//
//	redirect := metadata.Pairs(
//		"Redirect", "https://5kbps.io",
//		"Redirect-Code", "308",
//	)
//	grpc.SendHeader(ctx, redirect)
//
// This will redirect endpoint to https://5kbps.io with code 308, by default redirect code is 301.
func Redirect() runtime.ServeMuxOption {
	fn := func(ctx context.Context, w http.ResponseWriter, _ proto.Message) error {
		md, ok := runtime.ServerMetadataFromContext(ctx)
		if !ok {
			return nil
		}

		redirect := md.HeaderMD.Get("redirect")
		redirectCode := md.HeaderMD.Get("redirect-code")
		if len(redirect) > 0 {
			w.Header().Set("Location", redirect[0])
			if len(redirectCode) > 0 {
				code, err := strconv.Atoi(redirectCode[0])
				if err != nil {
					return err
				}
				if code > 300 && code < 400 {
					w.WriteHeader(code)
				}
			}

			w.WriteHeader(http.StatusMovedPermanently)
		}
		return nil
	}
	return runtime.WithForwardResponseOption(fn)
}

// FormURLEncodedMarshaler is custom Marshaler that supports reading x-www-form-urlencoded mine type.
func FormURLEncodedMarshaler() runtime.ServeMuxOption {
	return runtime.WithMarshalerOption("application/x-www-form-urlencoded", &formMarshaler{
		&runtime.JSONPb{},
	})
}

type formMarshaler struct {
	*runtime.JSONPb
}

func (j *formMarshaler) NewDecoder(r io.Reader) runtime.Decoder {
	return runtime.DecoderFunc(func(v any) error { return formDecoderFunc(r, v) })
}

func formDecoderFunc(d io.Reader, v any) error {
	msg, ok := v.(proto.Message)
	if !ok {
		return errors.New("not proto message")
	}
	formData, err := io.ReadAll(d)
	if err != nil {
		return err
	}
	values, err := url.ParseQuery(string(formData))
	if err != nil {
		return err
	}
	filter := &utilities.DoubleArray{}
	err = runtime.PopulateQueryParameters(msg, values, filter)
	if err != nil {
		return err
	}
	return nil
}
