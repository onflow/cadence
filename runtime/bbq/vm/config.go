package vm

import (
	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/bbq/compiler"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/errors"
)

type Config struct {
	Storage
	common.MemoryGauge
	compiler.ImportHandler
	ContractValueHandler
}

type ContractValueHandler func(conf *Config, location common.Location) *CompositeValue

type Storage interface {
	atree.SlabStorage
}

func RemoveReferencedSlab(storage Storage, storable atree.Storable) {
	storageIDStorable, ok := storable.(atree.StorageIDStorable)
	if !ok {
		return
	}

	storageID := atree.StorageID(storageIDStorable)
	err := storage.Remove(storageID)
	if err != nil {
		panic(errors.NewExternalError(err))
	}
}
