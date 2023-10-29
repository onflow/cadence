package interpreter

import "github.com/onflow/atree"

type NonStorable struct {
	Value Value
}

var _ atree.Storable = NonStorable{}

func (s NonStorable) Encode(_ *atree.Encoder) error {
	//nolint:gosimple
	return NonStorableValueError{
		Value: s.Value,
	}
}

func (s NonStorable) ByteSize() uint32 {
	// Return 1 so that atree split and merge operations don't have to handle special cases.
	// Any value larger than 0 and smaller than half of the max slab size works,
	// but 1 results in fewer number of slabs which is ideal for non-storable values.
	return 1
}

func (s NonStorable) StoredValue(_ atree.SlabStorage) (atree.Value, error) {
	return s.Value, nil
}

func (NonStorable) ChildStorables() []atree.Storable {
	return nil
}
