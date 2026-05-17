package gatewayopt

import (
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func TestFormMarshaler_NewDecoder(t *testing.T) {
	m := &formMarshaler{
		JSONPb: &runtime.JSONPb{
			MarshalOptions:   protojson.MarshalOptions{UseProtoNames: true},
			UnmarshalOptions: protojson.UnmarshalOptions{},
		},
	}

	data := "paths=name&paths=email"
	decoder := m.NewDecoder(strings.NewReader(data))

	actual := &fieldmaskpb.FieldMask{}
	err := decoder.Decode(actual)

	require.NoError(t, err)
	assert.Equal(t, []string{"name", "email"}, actual.GetPaths())
}
