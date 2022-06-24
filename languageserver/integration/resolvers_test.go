package integration

import (
	"testing"

	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/flow-go-sdk"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func Test_FileImport(t *testing.T) {
	t.Parallel()
	mockFS := afero.NewMemMapFs()
	af := afero.Afero{Fs: mockFS}
	_ = afero.WriteFile(mockFS, "./test.cdc", []byte(`hello test`), 0644)

	resolver := resolvers{
		loader: af,
	}

	t.Run("existing file", func(t *testing.T) {
		resolved, err := resolver.fileImport("./test.cdc")
		assert.NoError(t, err)
		assert.Equal(t, "hello test", resolved)
	})

	t.Run("non existing file", func(t *testing.T) {
		resolved, err := resolver.fileImport("./foo.cdc")
		assert.EqualError(t, err, "open foo.cdc: file does not exist")
		assert.Equal(t, "", resolved)
	})
}

func Test_AddressImport(t *testing.T) {
	mock := &mockFlowClient{}
	resolver := resolvers{
		client: mock,
	}

	t.Run("existing address", func(t *testing.T) {
		a, _ := common.HexToAddress("1")
		address := common.NewAddressLocation(nil, a, "test")
		flowAddress := flow.HexToAddress(a.String())

		mock.
			On("GetAccount", flowAddress).
			Return(&flow.Account{
				Address: flowAddress,
				Contracts: map[string][]byte{
					"test": []byte("hello tests"),
					"foo":  []byte("foo bar"),
				},
			}, nil)

		resolved, err := resolver.addressImport(address)
		assert.NoError(t, err)
		assert.Equal(t, "hello tests", resolved)
	})
}
