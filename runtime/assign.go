package runtime

import (
	"github.com/onflow/cadence/runtime/interpreter"
	"github.com/onflow/cadence/runtime/sema"
)

// assignType attempts to assign the given type to the given value.
func assignType(v interpreter.Value, t sema.Type, inter *interpreter.Interpreter) error {
	// the following cases require extra processing to ensure that all composite
	// types are linked properly
	switch v := v.(type) {
	case *interpreter.ArrayValue:
		err := assignArrayType(v, t, inter)
		if err != nil {
			return err
		}
	case *interpreter.DictionaryValue:
		err := assignDictionaryType(v, t, inter)
		if err != nil {
			return err
		}
	case *interpreter.CompositeValue:
		err := assignCompositeType(v, t, inter)
		if err != nil {
			return err
		}
	case *interpreter.SomeValue:
		err := assignSomeType(v, t, inter)
		if err != nil {
			return err
		}
	}

	// after type linking, check that argument is a subtype of parameter type
	if !interpreter.IsSubType(v.DynamicType(inter), t) {
		return &InvalidTypeAssignmentError{
			Value: v,
			Type:  t,
		}
	}

	return nil
}

// assignArrayType attempts to assign the given type to an array value.
//
// Array type assignment is valid if:
// 1. Type implements sema.ArrayType
// 2. Each array element is assignable to the array element type
func assignArrayType(v *interpreter.ArrayValue, t sema.Type, inter *interpreter.Interpreter) error {
	arrayType, ok := t.(sema.ArrayType)
	if !ok {
		return &InvalidTypeAssignmentError{Value: v, Type: t}
	}

	elemType := arrayType.ElementType(false)

	for _, elem := range v.Values {
		err := assignType(elem, elemType, inter)
		if err != nil {
			return err
		}
	}

	return nil
}

// assignDictionaryType attempts to assign the given type to a dictionary value.
//
// Dictionary type assignment is valid if:
// 1. Type is sema.DictionaryType
// 2. Each dictionary key is assignable to the dictionary key type
// 3. Each dictionary element is assignable to the dictionary element type
func assignDictionaryType(v *interpreter.DictionaryValue, t sema.Type, inter *interpreter.Interpreter) error {
	dictType, ok := t.(*sema.DictionaryType)
	if !ok {
		return &InvalidTypeAssignmentError{Value: v, Type: t}
	}

	keyType := dictType.KeyType
	elemType := dictType.ElementType(false)

	for _, key := range v.Keys.Values {
		err := assignType(key, keyType, inter)
		if err != nil {
			return &InvalidTypeAssignmentError{
				Value: v,
				Type:  t,
				Err: &InvalidTypeAssignmentDictionaryIncompatibleKeyError{
					Key: key,
					Err: err,
				},
			}
		}
	}

	for key, elem := range v.Entries {
		err := assignType(elem, elemType, inter)
		if err != nil {
			return &InvalidTypeAssignmentError{
				Value: v,
				Type:  t,
				Err: &InvalidTypeAssignmentDictionaryIncompatibleElementError{
					Key: key,
					Err: err,
				},
			}
		}
	}

	return nil
}

// assignCompositeType attempts to assign the given type to a composite value.
//
// Composite type assignment is valid if:
// 1. Type is sema.CompositeType
// 2. The composite value contains a field value for each field declared in the composite type.
// 3. Each composite field is assignable to its corresponding field type.
func assignCompositeType(v *interpreter.CompositeValue, t sema.Type, inter *interpreter.Interpreter) error {
	compType, ok := t.(*sema.CompositeType)
	if !ok {
		return &InvalidTypeAssignmentError{Value: v, Type: t}
	}

	for name, member := range compType.Members {
		field, ok := v.Fields[name]
		if !ok {
			return &InvalidTypeAssignmentError{
				Value: v,
				Type:  t,
				Err: &InvalidTypeAssignmentCompositeMissingFieldError{
					Field: name,
				},
			}
		}

		err := assignType(field, member.TypeAnnotation.Type, inter)
		if err != nil {
			return &InvalidTypeAssignmentError{
				Value: v,
				Type:  t,
				Err: &InvalidTypeAssignmentCompositeIncompatibleFieldError{
					Field: name,
					Err:   err,
				},
			}
		}
	}

	// Link the type, location and kind for this composite value. This allows
	// us to derive the dynamic type of this value in the future.
	v.TypeID = compType.ID()
	v.Location = compType.Location
	v.Kind = compType.Kind

	return nil
}

// assignSomeType attempts to assign the given type to some value.
//
// Type assignment is valid if:
// 1. Type is sema.OptionalType
// 2. The wrapped value is assignable to the inner type of the optional type.
func assignSomeType(v *interpreter.SomeValue, t sema.Type, inter *interpreter.Interpreter) error {
	optType, ok := t.(*sema.OptionalType)
	if !ok {
		return &InvalidTypeAssignmentError{Value: v, Type: t}
	}

	err := assignType(v.Value, optType.Type, inter)
	if err != nil {
		return nil
	}

	return nil
}
