package gatewayopt

import (
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	pbtest "github.com/vchitai/grpcx/grpc/gatewayopt/internal/pbtest"
)

func TestFormMarshaler_NewDecoder(t *testing.T) {
	// Arrange
	m := &formMarshaler{
		JSONPb: &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				UseProtoNames: true,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{},
		},
	}

	data := "errorCode=0&requestId=978fb5c4-88ff-42bc-8b69-522eda5c56a3&amount=300000&orderId=QD122IWPGJ51&orderInfo=QD122IWPGJ51&orderType=momo_wallet&transId=2319925829&message=Success"
	decoder := m.NewDecoder(strings.NewReader(data))
	actual := &pbtest.TestRequest{}
	expected := &pbtest.TestRequest{
		RequestId: "978fb5c4-88ff-42bc-8b69-522eda5c56a3",
		Amount:    "300000",
		OrderId:   "QD122IWPGJ51",
		OrderInfo: "QD122IWPGJ51",
		OrderType: "momo_wallet",
		TransId:   "2319925829",
		Message:   "Success",
	}
	// Act
	err := decoder.Decode(actual)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, expected.GetRequestId(), actual.GetRequestId())
	assert.Equal(t, expected.GetAmount(), actual.GetAmount())
	assert.Equal(t, expected.GetOrderId(), actual.GetOrderId())
	assert.Equal(t, expected.GetOrderInfo(), actual.GetOrderInfo())
	assert.Equal(t, expected.GetOrderType(), actual.GetOrderType())
	assert.Equal(t, expected.GetTransId(), actual.GetTransId())
	assert.Equal(t, expected.GetMessage(), actual.GetMessage())
}
