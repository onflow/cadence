package interpreter

import (
	"fmt"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/onflow/atree"

	"github.com/onflow/cadence/runtime/ast"
	"github.com/onflow/cadence/runtime/common"
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
	"github.com/onflow/cadence/runtime/tests/utils"
)

const containerMaxDepth = 3
const containerMaxSize = 300

// TODO make this a param
const valueCount = 1_000

func TestRandomMapOperations(t *testing.T) {
	// TODO Skip by default

	seed := time.Now().UnixNano()
	fmt.Printf("Seed used for test: %d \n", seed)
	rand.Seed(seed)

	storage := interpreter.NewInMemoryStorage()
	inter, err := interpreter.NewInterpreter(
		&interpreter.Program{
			Program:     ast.NewProgram([]ast.Declaration{}),
			Elaboration: sema.NewElaboration(),
		},
		utils.TestLocation,
		interpreter.WithStorage(storage),
		interpreter.WithImportLocationHandler(
			func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				return interpreter.VirtualImport{
					Elaboration: inter.Program.Elaboration,
				}
			},
		),
	)
	require.NoError(t, err)

	numberOfValues := rand.Intn(valueCount)

	var testMap, copyOfTestMap *interpreter.DictionaryValue
	var storageSize, slabCounts int

	entries := make(map[interface{}]interpreter.Value, numberOfValues)
	orgOwner := common.Address([8]byte{'A'})

	t.Run("dictionary construction", func(t *testing.T) {
		keyValues := make([]interpreter.Value, numberOfValues*2)
		for i := 0; i < numberOfValues; i++ {
			key := randomHashableValue(inter)
			value := randomStorableValue(inter, orgOwner, 0)

			mapKey := goMapKey(key)
			entries[mapKey] = value

			keyValues[i*2] = key
			keyValues[i*2+1] = deepCopyValue(inter, value)
		}

		testMap = interpreter.NewDictionaryValueWithAddress(inter,
			interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeAnyStruct,
				ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			orgOwner,
			keyValues...,
		)

		storageSize, slabCounts = getSlabStorageSize(t, storage)

		for orgKey, orgValue := range entries {
			key := dictionaryKey(orgKey)

			exists := testMap.ContainsKey(inter, interpreter.ReturnEmptyLocationRange, key)
			require.True(t, bool(exists))

			value, found := testMap.Get(inter, interpreter.ReturnEmptyLocationRange, key)
			require.True(t, found)
			utils.AssertValuesEqual(t, inter, orgValue, value)
		}

		require.Equal(t, testMap.Count(), len(entries))

		owner := testMap.GetOwner()
		require.Equal(t, owner[:], orgOwner[:])
	})

	t.Run("iterate", func(t *testing.T) {
		require.Equal(t, testMap.Count(), len(entries))

		testMap.Iterate(func(key, value interpreter.Value) (resume bool) {
			mapKey := goMapKey(key)
			orgValue, _ := entries[mapKey]
			utils.AssertValuesEqual(t, inter, orgValue, value)
			return true
		})
	})

	t.Run("deep copy", func(t *testing.T) {
		newOwner := atree.Address([8]byte{'B'})
		copyOfTestMap = testMap.DeepCopy(
			inter,
			interpreter.ReturnEmptyLocationRange,
			newOwner,
		).(*interpreter.DictionaryValue)

		for orgKey, orgValue := range entries {
			key := dictionaryKey(orgKey)

			exists := copyOfTestMap.ContainsKey(inter, interpreter.ReturnEmptyLocationRange, key)
			require.True(t, bool(exists))

			value, found := copyOfTestMap.Get(inter, interpreter.ReturnEmptyLocationRange, key)
			require.True(t, found)
			utils.AssertValuesEqual(t, inter, orgValue, value)
		}

		require.Equal(t, copyOfTestMap.Count(), len(entries))

		owner := copyOfTestMap.GetOwner()
		require.Equal(t, owner[:], newOwner[:])
	})

	t.Run("deep remove", func(t *testing.T) {
		copyOfTestMap.DeepRemove(inter)
		err = storage.Remove(copyOfTestMap.StorageID())
		require.NoError(t, err)

		// deep removal should clean up everything
		newStorageSize, newSlabCounts := getSlabStorageSize(t, storage)
		require.Equal(t, slabCounts, newSlabCounts)
		require.Equal(t, storageSize, newStorageSize)

		// go over original values again and check no missing data (no side effect should be found)
		for orgKey, orgValue := range entries {
			key := dictionaryKey(orgKey)

			exists := testMap.ContainsKey(inter, interpreter.ReturnEmptyLocationRange, key)
			require.True(t, bool(exists))

			value, found := testMap.Get(inter, interpreter.ReturnEmptyLocationRange, key)
			require.True(t, found)
			utils.AssertValuesEqual(t, inter, orgValue, value)
		}

		require.Equal(t, testMap.Count(), len(entries))

		owner := testMap.GetOwner()
		require.Equal(t, owner[:], orgOwner[:])
	})
}

func TestRandomArrayOperations(t *testing.T) {
	// TODO Skip by default

	seed := time.Now().UnixNano()
	fmt.Printf("Seed used for test: %d \n", seed)
	rand.Seed(seed)

	storage := interpreter.NewInMemoryStorage()
	inter, err := interpreter.NewInterpreter(
		&interpreter.Program{
			Program:     ast.NewProgram([]ast.Declaration{}),
			Elaboration: sema.NewElaboration(),
		},
		utils.TestLocation,
		interpreter.WithStorage(storage),
		interpreter.WithImportLocationHandler(
			func(inter *interpreter.Interpreter, location common.Location) interpreter.Import {
				return interpreter.VirtualImport{
					Elaboration: inter.Program.Elaboration,
				}
			},
		),
	)
	require.NoError(t, err)

	numberOfValues := rand.Intn(valueCount / 100)

	var testArray, copyOfTestArray *interpreter.ArrayValue
	var storageSize, slabCounts int

	elements := make([]interpreter.Value, numberOfValues)
	orgOwner := common.Address([8]byte{'A'})

	values := make([]interpreter.Value, numberOfValues)

	t.Run("array construction", func(t *testing.T) {
		for i := 0; i < numberOfValues; i++ {
			value := randomStorableValue(inter, orgOwner, 0)
			elements[i] = value
			values[i] = deepCopyValue(inter, value)
		}

		testArray = interpreter.NewArrayValue(
			inter,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			orgOwner,
			values...,
		)

		storageSize, slabCounts = getSlabStorageSize(t, storage)

		for index, orgElement := range elements {
			element := testArray.Get(inter, interpreter.ReturnEmptyLocationRange, index)
			utils.AssertValuesEqual(t, inter, orgElement, element)
		}

		require.Equal(t, testArray.Count(), len(elements))

		owner := testArray.GetOwner()
		require.Equal(t, owner[:], orgOwner[:])
	})

	t.Run("iterate", func(t *testing.T) {
		require.Equal(t, testArray.Count(), len(elements))

		index := 0
		testArray.Iterate(func(element interpreter.Value) (resume bool) {
			orgElement := elements[index]
			utils.AssertValuesEqual(t, inter, orgElement, element)

			elementByIndex := testArray.Get(inter, interpreter.ReturnEmptyLocationRange, index)
			utils.AssertValuesEqual(t, inter, element, elementByIndex)

			index++
			return true
		})
	})

	t.Run("deep copy", func(t *testing.T) {
		newOwner := atree.Address([8]byte{'B'})
		copyOfTestArray = testArray.DeepCopy(
			inter,
			interpreter.ReturnEmptyLocationRange,
			newOwner,
		).(*interpreter.ArrayValue)

		for index, orgElement := range elements {
			element := copyOfTestArray.Get(inter, interpreter.ReturnEmptyLocationRange, index)
			utils.AssertValuesEqual(t, inter, orgElement, element)
		}

		require.Equal(t, copyOfTestArray.Count(), len(elements))

		owner := copyOfTestArray.GetOwner()
		require.Equal(t, owner[:], newOwner[:])
	})

	t.Run("deep removal", func(t *testing.T) {
		copyOfTestArray.DeepRemove(inter)
		err = storage.Remove(copyOfTestArray.StorageID())
		require.NoError(t, err)

		// deep removal should clean up everything
		newStorageSize, newSlabCounts := getSlabStorageSize(t, storage)
		require.Equal(t, slabCounts, newSlabCounts)
		require.Equal(t, storageSize, newStorageSize)

		// go over original elements again and check no missing data (no side effect should be found)
		for index, orgElement := range elements {
			element := testArray.Get(inter, interpreter.ReturnEmptyLocationRange, index)
			utils.AssertValuesEqual(t, inter, orgElement, element)
		}

		require.Equal(t, testArray.Count(), len(elements))

		owner := testArray.GetOwner()
		require.Equal(t, owner[:], orgOwner[:])
	})
}

func goMapKey(key interpreter.Value) interface{} {
	// Dereference string-keys before putting into go-map,
	// as go-map hashing treats pointers as unique values.
	// i.e: Maintain the value as the key, rather than the pointer.
	switch key := key.(type) {
	case *interpreter.StringValue:
		return *key
	case interpreter.Value:
		return key
	default:
		panic("unreachable")
	}
}

func dictionaryKey(i interface{}) interpreter.Value {
	switch key := i.(type) {
	case interpreter.StringValue:
		return &key
	case interpreter.Value:
		return key
	default:
		panic("unreachable")
	}
}

func getSlabStorageSize(t *testing.T, storage interpreter.InMemoryStorage) (totalSize int, slabCounts int) {
	slabs, err := storage.Encode()
	require.NoError(t, err)

	for _, slab := range slabs {
		totalSize += len(slab)
		slabCounts++
	}

	return
}

func deepCopyValue(inter *interpreter.Interpreter, value interpreter.Value) interpreter.Value {
	switch v := value.(type) {

	// Int
	case interpreter.IntValue:
		var n big.Int
		n.Set(v.BigInt)
		return interpreter.NewIntValueFromBigInt(&n)
	case interpreter.Int8Value:
		return interpreter.Int8Value(int8(v))
	case interpreter.Int16Value:
		return interpreter.Int16Value(int16(v))
	case interpreter.Int32Value:
		return interpreter.Int32Value(int32(v))
	case interpreter.Int64Value:
		return interpreter.Int64Value(int64(v))
	case interpreter.Int128Value:
		var n big.Int
		n.Set(v.BigInt)
		return interpreter.NewInt128ValueFromBigInt(&n)
	case interpreter.Int256Value:
		var n big.Int
		n.Set(v.BigInt)
		return interpreter.NewInt256ValueFromBigInt(&n)

	// Uint
	case interpreter.UIntValue:
		var n big.Int
		n.Set(v.BigInt)
		return interpreter.NewUIntValueFromBigInt(&n)
	case interpreter.UInt8Value:
		return interpreter.UInt8Value(uint8(v))
	case interpreter.UInt16Value:
		return interpreter.UInt16Value(uint16(v))
	case interpreter.UInt32Value:
		return interpreter.UInt32Value(uint32(v))
	case interpreter.UInt64Value:
		return interpreter.UInt64Value(uint64(v))
	case interpreter.UInt128Value:
		var n big.Int
		n.Set(v.BigInt)
		return interpreter.NewUInt128ValueFromBigInt(&n)
	case interpreter.UInt256Value:
		var n big.Int
		n.Set(v.BigInt)
		return interpreter.NewUInt256ValueFromBigInt(&n)

	case interpreter.Word8Value,
		interpreter.Word16Value,
		interpreter.Word32Value,
		interpreter.Word64Value:
		return v

	case *interpreter.StringValue:
		b := []byte(v.Str)
		data := make([]byte, len(b))
		copy(data, b)
		return interpreter.NewStringValue(string(data))

	case interpreter.AddressValue:
		b := v[:]
		data := make([]byte, len(b))
		copy(data, b)
		return interpreter.NewAddressValueFromBytes(data)
	case interpreter.Fix64Value:
		return interpreter.NewFix64ValueWithInteger(int64(v.ToInt()))
	case interpreter.UFix64Value:
		return interpreter.NewUFix64ValueWithInteger(uint64(v.ToInt()))

	case interpreter.PathValue:
		return interpreter.PathValue{
			Domain:     v.Domain,
			Identifier: v.Identifier,
		}

	case interpreter.BoolValue:
		return v

	case interpreter.VoidValue:
		return interpreter.VoidValue{}

	case *interpreter.DictionaryValue:
		keyValues := make([]interpreter.Value, 0, v.Count()*2)
		v.Iterate(func(key, value interpreter.Value) (resume bool) {
			keyValues = append(keyValues, deepCopyValue(inter, key))
			keyValues = append(keyValues, deepCopyValue(inter, value))
			return true
		})

		return interpreter.NewDictionaryValueWithAddress(
			inter,
			interpreter.DictionaryStaticType{
				KeyType:   v.Type.KeyType,
				ValueType: v.Type.ValueType,
			},
			v.GetOwner(),
			keyValues...,
		)
	case *interpreter.ArrayValue:
		elements := make([]interpreter.Value, 0, v.Count())
		v.Iterate(func(value interpreter.Value) (resume bool) {
			elements = append(elements, deepCopyValue(inter, value))
			return true
		})

		return interpreter.NewArrayValue(
			inter,
			v.Type,
			v.GetOwner(),
			elements...,
		)
	case *interpreter.CompositeValue:
		fields := make([]interpreter.CompositeField, 0)
		v.ForEachField(func(name string, value interpreter.Value) {
			fields = append(fields, interpreter.CompositeField{
				Name:  name,
				Value: deepCopyValue(inter, value),
			})
		})

		return interpreter.NewCompositeValue(
			inter,
			v.Location,
			v.QualifiedIdentifier,
			v.Kind,
			fields,
			v.GetOwner(),
		)
	default:
		return interpreter.NilValue{}
	}
}

func randomStorableValue(inter *interpreter.Interpreter, owner common.Address, currentDepth int) interpreter.Value {
	n := 0
	if currentDepth < containerMaxDepth {
		n = rand.Intn(Resource)
	} else {
		n = rand.Intn(Nil)
	}

	switch n {

	// Non-hashable
	case Void:
		return interpreter.VoidValue{}
	case Nil:
		return interpreter.NilValue{}
	case Dictionary:
		entryCount := rand.Intn(containerMaxSize)
		keyValues := make([]interpreter.Value, entryCount*2)
		entries := make(map[interface{}]interpreter.Value, entryCount)

		for i := 0; i < entryCount; i++ {
			key := randomHashableValue(inter)
			value := randomStorableValue(inter, owner, currentDepth+1)

			var mapKey interface{}

			// Dereference string-keys before putting into go-map,
			// as go-map hashing treats pointers as unique values.
			// i.e: Maintain the value as the key, rather than the pointer.
			switch key := deepCopyValue(inter, key).(type) {
			case *interpreter.StringValue:
				mapKey = *key
			case interpreter.Value:
				mapKey = key
			default:
				panic("unreachable")
			}

			entries[mapKey] = deepCopyValue(inter, value)

			keyValues[i*2] = key
			keyValues[i*2+1] = value
		}

		return interpreter.NewDictionaryValueWithAddress(
			inter,
			interpreter.DictionaryStaticType{
				KeyType:   interpreter.PrimitiveStaticTypeAnyStruct,
				ValueType: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			owner,
			keyValues...,
		)
	case Array:
		elementsCount := rand.Intn(containerMaxSize)
		elements := make([]interpreter.Value, elementsCount)

		for i := 0; i < elementsCount; i++ {
			value := randomStorableValue(inter, owner, currentDepth+1)
			elements[i] = deepCopyValue(inter, value)
		}

		return interpreter.NewArrayValue(
			inter,
			interpreter.VariableSizedStaticType{
				Type: interpreter.PrimitiveStaticTypeAnyStruct,
			},
			owner,
			elements...,
		)

	case Struct:
		return generateCompositeValue(inter, common.CompositeKindStructure, owner, currentDepth)
	case Resource:
		return generateCompositeValue(inter, common.CompositeKindResource, owner, currentDepth)
	case Contract:
		return generateCompositeValue(inter, common.CompositeKindContract, owner, currentDepth)

	// Hashable
	default:
		return generateRandomHashableValue(inter, n)
	}
}

func randomHashableValue(interpreter *interpreter.Interpreter) interpreter.Value {
	return generateRandomHashableValue(interpreter, rand.Intn(Enum))
}

func generateRandomHashableValue(inter *interpreter.Interpreter, n int) interpreter.Value {
	switch n {
	// TODO deal with negative numbers

	// Int
	case Int:
		return interpreter.NewIntValueFromInt64(rand.Int63())
	case Int8:
		return interpreter.Int8Value(rand.Intn(255))
	case Int16:
		return interpreter.Int16Value(rand.Intn(65535))
	case Int32:
		return interpreter.Int32Value(rand.Int31())
	case Int64:
		return interpreter.Int64Value(rand.Int63())
	case Int128:
		return interpreter.NewInt128ValueFromInt64(rand.Int63())
	case Int256:
		return interpreter.NewInt256ValueFromInt64(rand.Int63())

	// UInt
	case UInt:
		return interpreter.NewUIntValueFromUint64(rand.Uint64())
	case UInt8:
		return interpreter.UInt8Value(rand.Intn(255))
	case UInt16:
		return interpreter.UInt16Value(rand.Intn(65535))
	case UInt32:
		return interpreter.UInt32Value(rand.Uint32())
	case UInt64_1, UInt64_2, UInt64_3, UInt64_4: // should be more common
		return interpreter.UInt64Value(rand.Uint64())
	case UInt128:
		return interpreter.NewUInt128ValueFromBigInt(big.NewInt(rand.Int63()))
	case UInt256:
		return interpreter.NewUInt256ValueFromBigInt(big.NewInt(rand.Int63()))

	// Word
	case Word8:
		return interpreter.Word8Value(rand.Intn(255))
	case Word16:
		return interpreter.Word16Value(rand.Intn(65535))
	case Word32:
		return interpreter.Word32Value(rand.Uint32())
	case Word64:
		return interpreter.Word64Value(rand.Uint64())

	// Fixed point
	case Fix64:
		return interpreter.NewFix64ValueWithInteger(rand.Int63n(sema.Fix64TypeMaxInt))
	case UFix64:
		return interpreter.NewUFix64ValueWithInteger(
			uint64(rand.Int63n(
				int64(sema.UFix64TypeMaxInt),
			)),
		)

	// String
	case String_1, String_2, String_3, String_4: // small string - should be more common
		size := rand.Intn(255)
		data := make([]byte, size)
		rand.Read(data)
		return interpreter.NewStringValue(string(data))
	case String_5: // large string
		size := rand.Intn(4048) + 255
		data := make([]byte, size)
		rand.Read(data)
		return interpreter.NewStringValue(string(data))

	case Bool_True:
		return interpreter.BoolValue(true)
	case Bool_False:
		return interpreter.BoolValue(false)

	case Address:
		data := make([]byte, 8)
		rand.Read(data)
		return interpreter.NewAddressValueFromBytes(data)

	case Path:
		randomDomain := rand.Intn(len(common.AllPathDomains))
		identifier := make([]byte, 8)
		rand.Read(identifier)

		return interpreter.PathValue{
			Domain:     common.AllPathDomains[randomDomain],
			Identifier: string(identifier),
		}

	case Enum:
		typ := rand.Intn(Word64)
		rawValue := generateRandomHashableValue(inter, typ).(interpreter.NumberValue)

		identifier := make([]byte, 8)
		rand.Read(identifier)

		address := make([]byte, 8)
		rand.Read(address)

		location := common.AddressLocation{
			Address: common.BytesToAddress(address),
			Name:    string(identifier),
		}

		enumType := &sema.CompositeType{
			Identifier:  string(identifier),
			EnumRawType: intSubtype(typ),
			Kind:        common.CompositeKindEnum,
			Location:    location,
		}

		inter.Program.Elaboration.CompositeTypes[enumType.ID()] = enumType

		return interpreter.NewEnumCaseValue(
			inter,
			enumType,
			rawValue,
			nil,
		)

	default:
		panic(fmt.Sprintf("unsupported:  %d", n))
	}
}

func generateCompositeValue(
	inter *interpreter.Interpreter,
	kind common.CompositeKind,
	owner common.Address,
	currentDepth int,
) interpreter.Value {

	identifier := make([]byte, 8)
	rand.Read(identifier)

	address := make([]byte, 8)
	rand.Read(address)

	location := common.AddressLocation{
		Address: common.BytesToAddress(address),
		Name:    string(identifier),
	}

	fieldsCount := rand.Intn(8)
	fields := make([]interpreter.CompositeField, fieldsCount)

	for i := 0; i < fieldsCount; i++ {
		fieldName := make([]byte, 8)
		rand.Read(fieldName)

		fields[i] = interpreter.CompositeField{
			Name:  string(fieldName),
			Value: randomStorableValue(inter, owner, currentDepth+1),
		}
	}

	compositeType := &sema.CompositeType{
		Location:   location,
		Identifier: string(identifier),
		Kind:       common.CompositeKindContract,
	}

	compositeType.Members = sema.NewStringMemberOrderedMap()
	for _, field := range fields {
		compositeType.Members.Set(
			field.Name,
			sema.NewPublicConstantFieldMember(
				compositeType,
				field.Name,
				sema.AnyStructType, // TODO: handle resources
				"",
			),
		)
	}

	// Add the type to the elaboration, to short-circuit the type lookup
	inter.Program.Elaboration.CompositeTypes[compositeType.ID()] = compositeType

	return interpreter.NewCompositeValue(
		inter,
		location,
		string(identifier),
		kind,
		fields,
		owner,
	)
}

func intSubtype(n int) sema.Type {
	switch n {
	// Int
	case Int:
		return sema.IntType
	case Int8:
		return sema.Int8Type
	case Int16:
		return sema.Int16Type
	case Int32:
		return sema.Int32Type
	case Int64:
		return sema.Int64Type
	case Int128:
		return sema.Int128Type
	case Int256:
		return sema.Int256Type

	// UInt
	case UInt:
		return sema.UIntType
	case UInt8:
		return sema.UInt8Type
	case UInt16:
		return sema.UInt16Type
	case UInt32:
		return sema.UInt32Type
	case UInt64_1, UInt64_2, UInt64_3, UInt64_4:
		return sema.UInt64Type
	case UInt128:
		return sema.UInt128Type
	case UInt256:
		return sema.UInt256Type

	// Word
	case Word8:
		return sema.Word8Type
	case Word16:
		return sema.Word16Type
	case Word32:
		return sema.Word32Type
	case Word64:
		return sema.Word64Type

	default:
		panic(fmt.Sprintf("unsupported:  %d", n))
	}
}

const (
	// Hashable values
	Int = iota
	Int8
	Int16
	Int32
	Int64
	Int128
	Int256

	UInt
	UInt8
	UInt16
	UInt32
	UInt64_1
	UInt64_2
	UInt64_3
	UInt64_4
	UInt128
	UInt256

	Word8
	Word16
	Word32
	Word64

	Fix64
	UFix64

	String_1
	String_2
	String_3
	String_4
	String_5

	Bool_True
	Bool_False
	Path
	Address
	Enum

	// Non-hashable values

	Void
	Nil // `Never?`

	// Containers
	Array
	Dictionary
	Struct
	Resource

	// TODO:
	Contract
)
