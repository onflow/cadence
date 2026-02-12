/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright Flow Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ast

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/turbolent/prettier"
)

func TestTypeAnnotation_Doc(t *testing.T) {

	t.Parallel()

	t.Run("non-resource, no type", func(t *testing.T) {
		t.Parallel()

		ty := &TypeAnnotation{}

		assert.Equal(t,
			prettier.Text(""),
			ty.Doc(),
		)
	})

	t.Run("non-resource, with type", func(t *testing.T) {
		t.Parallel()

		ty := &TypeAnnotation{
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "T",
				},
			},
		}

		assert.Equal(t,
			prettier.Text("T"),
			ty.Doc(),
		)
	})

	t.Run("resource, no type", func(t *testing.T) {
		t.Parallel()

		ty := &TypeAnnotation{
			IsResource: true,
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("@"),
				prettier.Text(""),
			},
			ty.Doc(),
		)
	})

	t.Run("resource, with type", func(t *testing.T) {
		t.Parallel()

		ty := &TypeAnnotation{
			IsResource: true,
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "R",
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("@"),
				prettier.Text("R"),
			},
			ty.Doc(),
		)
	})
}

func TestTypeAnnotation_String(t *testing.T) {

	t.Parallel()

	t.Run("non-resource, no type", func(t *testing.T) {
		t.Parallel()

		ty := &TypeAnnotation{}

		assert.Equal(t,
			"",
			ty.String(),
		)
	})

	t.Run("non-resource, with type", func(t *testing.T) {
		t.Parallel()

		ty := &TypeAnnotation{
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "T",
				},
			},
		}

		assert.Equal(t,
			"T",
			ty.String(),
		)
	})

	t.Run("resource, no type", func(t *testing.T) {
		t.Parallel()

		ty := &TypeAnnotation{
			IsResource: true,
		}

		assert.Equal(t,
			"@",
			ty.String(),
		)
	})

	t.Run("resource, with type", func(t *testing.T) {
		t.Parallel()

		ty := &TypeAnnotation{
			IsResource: true,
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "R",
				},
			},
		}

		assert.Equal(t,
			"@R",
			ty.String(),
		)
	})
}

func TestTypeAnnotation_MarshalJSON(t *testing.T) {

	t.Parallel()

	ty := &TypeAnnotation{
		IsResource: true,
		Type: &NominalType{
			Identifier: Identifier{
				Identifier: "foobar",
				Pos:        Position{Offset: 1, Line: 2, Column: 3},
			},
		},
		StartPos: Position{Offset: 4, Line: 5, Column: 6},
	}

	actual, err := json.Marshal(ty)
	require.NoError(t, err)

	assert.JSONEq(t,
		// language=json
		`
        {
            "IsResource": true,
            "AnnotatedType": {
                "Type": "NominalType",
                "Identifier": {
                    "Identifier": "foobar",
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
                },
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
            },
            "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
            "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
        }
        `,
		string(actual),
	)
}

func TestTypeStringParentheses(t *testing.T) {
	t.Parallel()

	t.Run("reference to optional", func(t *testing.T) {
		t.Parallel()

		ty := &ReferenceType{
			Type: &OptionalType{
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "T",
					},
				},
			},
		}

		assert.Equal(t,
			"&(T?)",
			ty.String(),
		)
	})

	t.Run("optional reference", func(t *testing.T) {
		t.Parallel()

		ty := &OptionalType{
			Type: &ReferenceType{
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "T",
					},
				},
			},
		}

		assert.Equal(t,
			"&T?",
			ty.String(),
		)
	})

	t.Run("optional function type", func(t *testing.T) {
		t.Parallel()

		ty := &OptionalType{
			Type: &FunctionType{
				ReturnTypeAnnotation: &TypeAnnotation{
					Type: &NominalType{
						Identifier: Identifier{
							Identifier: "T",
						},
					},
				},
			},
		}

		assert.Equal(t,
			"(fun (): T)?",
			ty.String(),
		)
	})

	t.Run("function type with optional return type", func(t *testing.T) {
		t.Parallel()

		ty := &FunctionType{
			ReturnTypeAnnotation: &TypeAnnotation{
				Type: &OptionalType{
					Type: &NominalType{
						Identifier: Identifier{
							Identifier: "T",
						},
					},
				},
			},
		}

		assert.Equal(t,
			"fun (): T?",
			ty.String(),
		)
	})

	t.Run("reference to function type", func(t *testing.T) {
		t.Parallel()

		ty := &ReferenceType{
			Type: &FunctionType{
				ReturnTypeAnnotation: &TypeAnnotation{
					Type: &NominalType{
						Identifier: Identifier{
							Identifier: "T",
						},
					},
				},
			},
		}

		assert.Equal(t,
			"&fun (): T",
			ty.String(),
		)
	})

	t.Run("optional instantiation", func(t *testing.T) {
		t.Parallel()

		ty := &OptionalType{
			Type: &InstantiationType{
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "Foo",
					},
				},
				TypeArguments: []*TypeAnnotation{
					{
						Type: &NominalType{
							Identifier: Identifier{
								Identifier: "Bar",
							},
						},
					},
				},
			},
		}

		assert.Equal(t,
			"Foo<Bar>?",
			ty.String(),
		)
	})

	t.Run("reference to instantiation", func(t *testing.T) {
		t.Parallel()

		ty := &ReferenceType{
			Type: &InstantiationType{
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "Foo",
					},
				},
				TypeArguments: []*TypeAnnotation{
					{
						Type: &NominalType{
							Identifier: Identifier{
								Identifier: "Bar",
							},
						},
					},
				},
			},
		}

		assert.Equal(t,
			"&Foo<Bar>",
			ty.String(),
		)
	})

	t.Run("instantiation of reference", func(t *testing.T) {
		t.Parallel()

		ty := &InstantiationType{
			Type: &ReferenceType{
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "Foo",
					},
				},
			},
			TypeArguments: []*TypeAnnotation{
				{
					Type: &NominalType{
						Identifier: Identifier{
							Identifier: "Bar",
						},
					},
				},
			},
		}

		assert.Equal(t,
			"(&Foo)<Bar>",
			ty.String(),
		)
	})

	t.Run("optional intersection type", func(t *testing.T) {
		t.Parallel()

		ty := &OptionalType{
			Type: &IntersectionType{
				Types: []*NominalType{
					{
						Identifier: Identifier{
							Identifier: "A",
						},
					},
					{
						Identifier: Identifier{
							Identifier: "B",
						},
					},
				},
			},
		}

		assert.Equal(t,
			"{A, B}?",
			ty.String(),
		)
	})

	t.Run("optional dictionary type", func(t *testing.T) {
		t.Parallel()

		ty := &OptionalType{
			Type: &DictionaryType{
				KeyType: &NominalType{
					Identifier: Identifier{
						Identifier: "K",
					},
				},
				ValueType: &NominalType{
					Identifier: Identifier{
						Identifier: "V",
					},
				},
			},
		}

		assert.Equal(t,
			"{K: V}?",
			ty.String(),
		)
	})

	t.Run("optional variable sized type", func(t *testing.T) {
		t.Parallel()

		variable := &OptionalType{
			Type: &VariableSizedType{
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "T",
					},
				},
			},
		}

		assert.Equal(t,
			"[T]?",
			variable.String(),
		)
	})

	t.Run("optional constant sized type", func(t *testing.T) {
		t.Parallel()

		constant := &OptionalType{
			Type: &ConstantSizedType{
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "T",
					},
				},
				Size: &IntegerExpression{
					Value:           big.NewInt(2),
					PositiveLiteral: []byte("2"),
				},
			},
		}

		assert.Equal(t,
			"[T; 2]?",
			constant.String(),
		)
	})

	t.Run("double optional", func(t *testing.T) {
		t.Parallel()

		ty := &OptionalType{
			Type: &OptionalType{
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "T",
					},
				},
			},
		}

		assert.Equal(t,
			"T??",
			ty.String(),
		)
	})

	t.Run("authorized reference to optional", func(t *testing.T) {
		t.Parallel()

		ty := &ReferenceType{
			Authorization: NewConjunctiveEntitlementSet(
				[]*NominalType{
					{
						Identifier: Identifier{
							Identifier: "E",
						},
					},
				},
			),
			Type: &OptionalType{
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "T",
					},
				},
			},
		}

		assert.Equal(t,
			"auth(E) &(T?)",
			ty.String(),
		)
	})

	t.Run("optional authorized reference", func(t *testing.T) {
		t.Parallel()

		ty := &OptionalType{
			Type: &ReferenceType{
				Authorization: NewConjunctiveEntitlementSet(
					[]*NominalType{
						{
							Identifier: Identifier{
								Identifier: "E",
							},
						},
					},
				),
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "T",
					},
				},
			},
		}

		assert.Equal(t,
			"(auth(E) &T)?",
			ty.String(),
		)
	})
}

func TestNominalType_Doc(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		ty := &NominalType{
			Identifier: Identifier{
				Identifier: "R",
			},
		}

		assert.Equal(t,
			prettier.Text("R"),
			ty.Doc(),
		)

	})

	t.Run("nested", func(t *testing.T) {

		t.Parallel()

		ty := &NominalType{
			Identifier: Identifier{
				Identifier: "R",
			},
			NestedIdentifiers: []Identifier{
				{
					Identifier: "S",
				},
				{
					Identifier: "T",
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("R"),
				prettier.Text("."),
				prettier.Text("S"),
				prettier.Text("."),
				prettier.Text("T"),
			},
			ty.Doc(),
		)

	})

}

func TestNominalType_String(t *testing.T) {

	t.Parallel()

	t.Run("simple", func(t *testing.T) {

		t.Parallel()

		ty := &NominalType{
			Identifier: Identifier{
				Identifier: "R",
			},
		}

		assert.Equal(t,
			"R",
			ty.String(),
		)

	})

	t.Run("nested", func(t *testing.T) {

		t.Parallel()

		ty := &NominalType{
			Identifier: Identifier{
				Identifier: "R",
			},
			NestedIdentifiers: []Identifier{
				{
					Identifier: "S",
				},
				{
					Identifier: "T",
				},
			},
		}

		assert.Equal(t,
			"R.S.T",
			ty.String(),
		)

	})

}

func TestNominalType_MarshalJSON(t *testing.T) {

	t.Parallel()

	ty := &NominalType{
		Identifier: Identifier{
			Identifier: "foobar",
			Pos:        Position{Offset: 1, Line: 2, Column: 3},
		},
		NestedIdentifiers: []Identifier{
			{
				Identifier: "baz",
				Pos:        Position{Offset: 4, Line: 5, Column: 6},
			},
		},
	}

	actual, err := json.Marshal(ty)
	require.NoError(t, err)

	assert.JSONEq(t,
		// language=json
		`
        {
            "Type": "NominalType",
            "Identifier": {
                "Identifier": "foobar",
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
            },
            "NestedIdentifiers": [
                {
                    "Identifier": "baz",
                    "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                    "EndPos": {"Offset": 6, "Line": 5, "Column": 8}
                }
            ],
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos": {"Offset": 6, "Line": 5, "Column": 8}
        }
        `,
		string(actual),
	)
}

func TestOptionalType_Doc(t *testing.T) {

	t.Parallel()

	t.Run("with type", func(t *testing.T) {
		t.Parallel()

		ty := &OptionalType{
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "R",
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("R"),
				prettier.Text("?"),
			},
			ty.Doc(),
		)
	})

	t.Run("nil type", func(t *testing.T) {
		t.Parallel()

		ty := &OptionalType{}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text(""),
				prettier.Text("?"),
			},
			ty.Doc(),
		)
	})
}

func TestOptionalType_String(t *testing.T) {

	t.Parallel()

	t.Run("with type", func(t *testing.T) {
		t.Parallel()

		ty := &OptionalType{
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "R",
				},
			},
		}

		assert.Equal(t,
			"R?",
			ty.String(),
		)
	})

	t.Run("nil type", func(t *testing.T) {
		t.Parallel()

		ty := &OptionalType{}

		assert.Equal(t,
			"?",
			ty.String(),
		)
	})
}

func TestOptionalType_MarshalJSON(t *testing.T) {

	t.Parallel()

	ty := &OptionalType{
		Type: &NominalType{
			Identifier: Identifier{
				Identifier: "foobar",
				Pos:        Position{Offset: 1, Line: 2, Column: 3},
			},
		},
		EndPos: Position{Offset: 4, Line: 5, Column: 6},
	}

	actual, err := json.Marshal(ty)
	require.NoError(t, err)

	assert.JSONEq(t,
		// language=json
		`
        {
            "Type": "OptionalType",
            "ElementType": {
                "Type": "NominalType",
                "Identifier": {
                    "Identifier": "foobar",
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
                },
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
            },
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos": {"Offset": 4, "Line": 5, "Column": 6}
        }
        `,
		string(actual),
	)
}

func TestVariableSizedType_Doc(t *testing.T) {

	t.Parallel()

	t.Run("with type", func(t *testing.T) {
		t.Parallel()

		ty := &VariableSizedType{
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "T",
				},
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("["),
					prettier.Indent{
						Doc: prettier.Concat{
							prettier.SoftLine{},
							prettier.Text("T"),
						},
					},
					prettier.SoftLine{},
					prettier.Text("]"),
				},
			},
			ty.Doc(),
		)
	})

	t.Run("nil type", func(t *testing.T) {
		t.Parallel()

		ty := &VariableSizedType{}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("["),
					prettier.Indent{
						Doc: prettier.Concat{
							prettier.SoftLine{},
							prettier.Text(""),
						},
					},
					prettier.SoftLine{},
					prettier.Text("]"),
				},
			},
			ty.Doc(),
		)
	})
}

func TestVariableSizedType_String(t *testing.T) {

	t.Parallel()

	t.Run("with type", func(t *testing.T) {
		t.Parallel()

		ty := &VariableSizedType{
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "T",
				},
			},
		}

		assert.Equal(t,
			"[T]",
			ty.String(),
		)
	})

	t.Run("nil type", func(t *testing.T) {
		t.Parallel()

		ty := &VariableSizedType{}

		assert.Equal(t,
			"[]",
			ty.String(),
		)
	})
}

func TestVariableSizedType_MarshalJSON(t *testing.T) {

	t.Parallel()

	ty := &VariableSizedType{
		Type: &NominalType{
			Identifier: Identifier{
				Identifier: "foobar",
				Pos:        Position{Offset: 1, Line: 2, Column: 3},
			},
		},
		Range: Range{
			StartPos: Position{Offset: 4, Line: 5, Column: 6},
			EndPos:   Position{Offset: 7, Line: 8, Column: 9},
		},
	}

	actual, err := json.Marshal(ty)
	require.NoError(t, err)

	assert.JSONEq(t,
		// language=json
		`
        {
            "Type": "VariableSizedType",
            "ElementType": {
                "Type": "NominalType",
                "Identifier": {
                    "Identifier": "foobar",
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
                },
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
            },
            "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
            "EndPos":  {"Offset": 7, "Line": 8, "Column": 9}
        }
        `,
		string(actual),
	)
}

func TestConstantSizedType_Doc(t *testing.T) {

	t.Parallel()

	t.Run("with type, with size", func(t *testing.T) {
		t.Parallel()

		ty := &ConstantSizedType{
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "T",
				},
			},
			Size: &IntegerExpression{
				PositiveLiteral: []byte("42"),
				Value:           big.NewInt(42),
				Base:            10,
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("["),
					prettier.Indent{
						Doc: prettier.Concat{
							prettier.SoftLine{},
							prettier.Text("T"),
							prettier.Text("; "),
							prettier.Text("42"),
						},
					},
					prettier.SoftLine{},
					prettier.Text("]"),
				},
			},
			ty.Doc(),
		)
	})

	t.Run("nil type, with size", func(t *testing.T) {
		t.Parallel()

		ty := &ConstantSizedType{
			Size: &IntegerExpression{
				PositiveLiteral: []byte("42"),
				Value:           big.NewInt(42),
				Base:            10,
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("["),
					prettier.Indent{
						Doc: prettier.Concat{
							prettier.SoftLine{},
							prettier.Text(""),
							prettier.Text("; "),
							prettier.Text("42"),
						},
					},
					prettier.SoftLine{},
					prettier.Text("]"),
				},
			},
			ty.Doc(),
		)
	})

	t.Run("with type, nil size", func(t *testing.T) {
		t.Parallel()

		ty := &ConstantSizedType{
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "T",
				},
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("["),
					prettier.Indent{
						Doc: prettier.Concat{
							prettier.SoftLine{},
							prettier.Text("T"),
							prettier.Text("; "),
							prettier.Text(""),
						},
					},
					prettier.SoftLine{},
					prettier.Text("]"),
				},
			},
			ty.Doc(),
		)
	})

	t.Run("nil type, nil size", func(t *testing.T) {
		t.Parallel()

		ty := &ConstantSizedType{}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("["),
					prettier.Indent{
						Doc: prettier.Concat{
							prettier.SoftLine{},
							prettier.Text(""),
							prettier.Text("; "),
							prettier.Text(""),
						},
					},
					prettier.SoftLine{},
					prettier.Text("]"),
				},
			},
			ty.Doc(),
		)
	})
}

func TestConstantSizedType_String(t *testing.T) {

	t.Parallel()

	t.Run("with type, with size", func(t *testing.T) {
		t.Parallel()

		ty := &ConstantSizedType{
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "T",
				},
			},
			Size: &IntegerExpression{
				PositiveLiteral: []byte("42"),
				Value:           big.NewInt(42),
				Base:            10,
			},
		}

		assert.Equal(t,
			"[T; 42]",
			ty.String(),
		)
	})

	t.Run("nil type, with size", func(t *testing.T) {
		t.Parallel()

		ty := &ConstantSizedType{
			Size: &IntegerExpression{
				PositiveLiteral: []byte("42"),
				Value:           big.NewInt(42),
				Base:            10,
			},
		}

		assert.Equal(t,
			"[; 42]",
			ty.String(),
		)
	})

	t.Run("with type, nil size", func(t *testing.T) {
		t.Parallel()

		ty := &ConstantSizedType{
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "T",
				},
			},
		}

		assert.Equal(t,
			"[T; ]",
			ty.String(),
		)
	})

	t.Run("nil type, nil size", func(t *testing.T) {
		t.Parallel()

		ty := &ConstantSizedType{}

		assert.Equal(t,
			"[; ]",
			ty.String(),
		)
	})
}

func TestConstantSizedType_MarshalJSON(t *testing.T) {

	t.Parallel()

	ty := &ConstantSizedType{
		Type: &NominalType{
			Identifier: Identifier{
				Identifier: "foobar",
				Pos:        Position{Offset: 1, Line: 2, Column: 3},
			},
		},
		Size: &IntegerExpression{
			PositiveLiteral: []byte("42"),
			Value:           big.NewInt(42),
			Base:            10,
			Range: Range{
				StartPos: Position{Offset: 4, Line: 5, Column: 6},
				EndPos:   Position{Offset: 7, Line: 8, Column: 9},
			},
		},
		Range: Range{
			StartPos: Position{Offset: 10, Line: 11, Column: 12},
			EndPos:   Position{Offset: 13, Line: 14, Column: 15},
		},
	}

	actual, err := json.Marshal(ty)
	require.NoError(t, err)

	assert.JSONEq(t,
		// language=json
		`
        {
            "Type": "ConstantSizedType",
            "ElementType": {
                "Type": "NominalType",
                "Identifier": {
                    "Identifier": "foobar",
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
                },
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 6, "Line": 2, "Column": 8}
            },
            "Size": {
                "Type": "IntegerExpression",
                "PositiveLiteral": "42",
                "Value": "42",
                "Base": 10,
                "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                "EndPos": {"Offset": 7, "Line": 8, "Column": 9}
            },
            "StartPos": {"Offset": 10, "Line": 11, "Column": 12},
            "EndPos":  {"Offset": 13, "Line": 14, "Column": 15}
        }
        `,
		string(actual),
	)
}

func TestDictionaryType_Doc(t *testing.T) {

	t.Parallel()

	t.Run("with key, with value", func(t *testing.T) {
		t.Parallel()

		ty := &DictionaryType{
			KeyType: &NominalType{
				Identifier: Identifier{
					Identifier: "AB",
				},
			},
			ValueType: &NominalType{
				Identifier: Identifier{
					Identifier: "CD",
				},
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("{"),
					prettier.Indent{
						Doc: prettier.Concat{
							prettier.SoftLine{},
							prettier.Text("AB"),
							prettier.Text(": "),
							prettier.Text("CD"),
						},
					},
					prettier.SoftLine{},
					prettier.Text("}"),
				},
			},
			ty.Doc(),
		)
	})

	t.Run("without key, with value", func(t *testing.T) {
		t.Parallel()

		ty := &DictionaryType{
			ValueType: &NominalType{
				Identifier: Identifier{
					Identifier: "CD",
				},
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("{"),
					prettier.Indent{
						Doc: prettier.Concat{
							prettier.SoftLine{},
							prettier.Text(""),
							prettier.Text(": "),
							prettier.Text("CD"),
						},
					},
					prettier.SoftLine{},
					prettier.Text("}"),
				},
			},
			ty.Doc(),
		)
	})

	t.Run("with key, without value", func(t *testing.T) {
		t.Parallel()

		ty := &DictionaryType{
			KeyType: &NominalType{
				Identifier: Identifier{
					Identifier: "AB",
				},
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("{"),
					prettier.Indent{
						Doc: prettier.Concat{
							prettier.SoftLine{},
							prettier.Text("AB"),
							prettier.Text(": "),
							prettier.Text(""),
						},
					},
					prettier.SoftLine{},
					prettier.Text("}"),
				},
			},
			ty.Doc(),
		)
	})

	t.Run("without key, without value", func(t *testing.T) {
		t.Parallel()

		ty := &DictionaryType{}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("{"),
					prettier.Indent{
						Doc: prettier.Concat{
							prettier.SoftLine{},
							prettier.Text(""),
							prettier.Text(": "),
							prettier.Text(""),
						},
					},
					prettier.SoftLine{},
					prettier.Text("}"),
				},
			},
			ty.Doc(),
		)
	})

}

func TestDictionaryType_String(t *testing.T) {

	t.Parallel()

	t.Run("with key, with value", func(t *testing.T) {
		t.Parallel()

		ty := &DictionaryType{
			KeyType: &NominalType{
				Identifier: Identifier{
					Identifier: "AB",
				},
			},
			ValueType: &NominalType{
				Identifier: Identifier{
					Identifier: "CD",
				},
			},
		}

		assert.Equal(t,
			"{AB: CD}",
			ty.String(),
		)
	})

	t.Run("without key, with value", func(t *testing.T) {
		t.Parallel()

		ty := &DictionaryType{
			ValueType: &NominalType{
				Identifier: Identifier{
					Identifier: "CD",
				},
			},
		}

		assert.Equal(t,
			"{: CD}",
			ty.String(),
		)
	})

	t.Run("with key, without value", func(t *testing.T) {
		t.Parallel()

		ty := &DictionaryType{
			KeyType: &NominalType{
				Identifier: Identifier{
					Identifier: "AB",
				},
			},
		}

		assert.Equal(t,
			"{AB: }",
			ty.String(),
		)
	})

	t.Run("without key, without value", func(t *testing.T) {
		t.Parallel()

		ty := &DictionaryType{}

		assert.Equal(t,
			"{: }",
			ty.String(),
		)
	})

}

func TestDictionaryType_MarshalJSON(t *testing.T) {

	t.Parallel()

	ty := &DictionaryType{
		KeyType: &NominalType{
			Identifier: Identifier{
				Identifier: "AB",
				Pos:        Position{Offset: 1, Line: 2, Column: 3},
			},
		},
		ValueType: &NominalType{
			Identifier: Identifier{
				Identifier: "CD",
				Pos:        Position{Offset: 4, Line: 5, Column: 6},
			},
		},
		Range: Range{
			StartPos: Position{Offset: 7, Line: 8, Column: 9},
			EndPos:   Position{Offset: 10, Line: 11, Column: 12},
		},
	}

	actual, err := json.Marshal(ty)
	require.NoError(t, err)

	assert.JSONEq(t,
		// language=json
		`
        {
            "Type": "DictionaryType",
            "KeyType": {
                "Type": "NominalType",
                "Identifier": {
                    "Identifier": "AB",
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 2, "Line": 2, "Column": 4}
                },
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 2, "Line": 2, "Column": 4}
            },
            "ValueType": {
                "Type": "NominalType",
                "Identifier": {
                    "Identifier": "CD",
                    "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                    "EndPos": {"Offset": 5, "Line": 5, "Column": 7}
                },
                "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                "EndPos": {"Offset": 5, "Line": 5, "Column": 7}
            },
            "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
            "EndPos": {"Offset": 10, "Line": 11, "Column": 12}
        }
        `,
		string(actual),
	)
}

func TestFunctionType_Doc(t *testing.T) {

	t.Parallel()

	t.Run("with parameter types, with return type", func(t *testing.T) {

		t.Parallel()

		ty := &FunctionType{
			PurityAnnotation: FunctionPurityView,
			ParameterTypeAnnotations: []*TypeAnnotation{
				{
					IsResource: true,
					Type: &NominalType{
						Identifier: Identifier{
							Identifier: "AB",
						},
					},
				},
				{
					IsResource: true,
					Type: &NominalType{
						Identifier: Identifier{
							Identifier: "CD",
						},
					},
				},
			},
			ReturnTypeAnnotation: &TypeAnnotation{
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "EF",
					},
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("view"),
				prettier.Space,
				prettier.Text("fun"),
				prettier.Space,
				prettier.Group{
					Doc: prettier.Concat{
						prettier.Text("("),
						prettier.Indent{
							Doc: prettier.Concat{
								prettier.SoftLine{},
								prettier.Concat{
									prettier.Text("@"),
									prettier.Text("AB"),
								},
								prettier.Text(","),
								prettier.Line{},
								prettier.Concat{
									prettier.Text("@"),
									prettier.Text("CD"),
								},
							},
						},
						prettier.SoftLine{},
						prettier.Text(")"),
					},
				},
				prettier.Text(": "),
				prettier.Text("EF"),
			},
			ty.Doc(),
		)
	})

	t.Run("nil parameter type, nil return type", func(t *testing.T) {

		t.Parallel()

		ty := &FunctionType{
			ParameterTypeAnnotations: []*TypeAnnotation{
				nil,
			},
			ReturnTypeAnnotation: nil,
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("fun"),
				prettier.Space,
				prettier.Group{
					Doc: prettier.Concat{
						prettier.Text("("),
						prettier.Indent{
							Doc: prettier.Concat{
								prettier.SoftLine{},
								prettier.Text(""),
							},
						},
						prettier.SoftLine{},
						prettier.Text(")"),
					},
				},
				prettier.Text(": "),
				prettier.Text(""),
			},
			ty.Doc(),
		)
	})

}

func TestFunctionType_String(t *testing.T) {

	t.Parallel()

	t.Run("with parameter types, with return type", func(t *testing.T) {

		t.Parallel()

		ty := &FunctionType{
			ParameterTypeAnnotations: []*TypeAnnotation{
				{
					IsResource: true,
					Type: &NominalType{
						Identifier: Identifier{
							Identifier: "AB",
						},
					},
				},
				{
					IsResource: true,
					Type: &NominalType{
						Identifier: Identifier{
							Identifier: "CD",
						},
					},
				},
			},
			ReturnTypeAnnotation: &TypeAnnotation{
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "EF",
					},
				},
			},
		}

		assert.Equal(t,
			"fun (@AB, @CD): EF",
			ty.String(),
		)
	})

	t.Run("nil parameter type, nil return type", func(t *testing.T) {

		t.Parallel()

		ty := &FunctionType{
			ParameterTypeAnnotations: []*TypeAnnotation{
				nil,
			},
			ReturnTypeAnnotation: nil,
		}

		assert.Equal(t,
			"fun (): ",
			ty.String(),
		)
	})
}

func TestFunctionType_MarshalJSON(t *testing.T) {

	t.Parallel()

	ty := &FunctionType{
		ParameterTypeAnnotations: []*TypeAnnotation{
			{
				IsResource: true,
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "AB",
						Pos:        Position{Offset: 1, Line: 2, Column: 3},
					},
				},
				StartPos: Position{Offset: 4, Line: 5, Column: 6},
			},
		},
		ReturnTypeAnnotation: &TypeAnnotation{
			IsResource: true,
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "CD",
					Pos:        Position{Offset: 7, Line: 8, Column: 9},
				},
			},
			StartPos: Position{Offset: 10, Line: 11, Column: 12},
		},
		Range: Range{
			StartPos: Position{Offset: 13, Line: 14, Column: 15},
			EndPos:   Position{Offset: 16, Line: 17, Column: 18},
		},
	}

	actual, err := json.Marshal(ty)
	require.NoError(t, err)

	assert.JSONEq(t,
		// language=json
		`
        {
            "Type": "FunctionType",
            "ParameterTypeAnnotations": [
                {
                    "IsResource": true,
                    "AnnotatedType": {
                        "Type": "NominalType",
                        "Identifier": {
                            "Identifier": "AB",
                            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                            "EndPos": {"Offset": 2, "Line": 2, "Column": 4}
                        },
                        "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                        "EndPos": {"Offset": 2, "Line": 2, "Column": 4}
                    },
                    "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                    "EndPos": {"Offset": 2, "Line": 2, "Column": 4}
                }
           ],
		   "PurityAnnotation": "Unspecified",
           "ReturnTypeAnnotation": {
               "IsResource": true,
               "AnnotatedType": {
                   "Type": "NominalType",
                   "Identifier": {
                       "Identifier": "CD",
                       "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                       "EndPos": {"Offset": 8, "Line": 8, "Column": 10}
                   },
                   "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                   "EndPos": {"Offset": 8, "Line": 8, "Column": 10}
               },
               "StartPos": {"Offset": 10, "Line": 11, "Column": 12},
               "EndPos": {"Offset": 8, "Line": 8, "Column": 10}
           },
           "StartPos": {"Offset": 13, "Line": 14, "Column": 15},
           "EndPos": {"Offset": 16, "Line": 17, "Column": 18}
        }
        `,
		string(actual),
	)
}

func TestReferenceType_Doc(t *testing.T) {

	t.Parallel()

	t.Run("auth with entitlement", func(t *testing.T) {

		t.Parallel()

		ty := &ReferenceType{
			Authorization: &ConjunctiveEntitlementSet{
				Elements: []*NominalType{
					{
						Identifier: Identifier{
							Identifier: "X",
						},
					},
				},
			},
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "T",
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("auth"),
				prettier.Text("("),
				prettier.Text("X"),
				prettier.Text(")"),
				prettier.Space,
				prettier.Text("&"),
				prettier.Text("T"),
			},
			ty.Doc(),
		)
	})

	t.Run("auth with nil entitlement", func(t *testing.T) {

		t.Parallel()

		ty := &ReferenceType{
			Authorization: &ConjunctiveEntitlementSet{
				Elements: []*NominalType{
					nil,
				},
			},
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "T",
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("auth"),
				prettier.Text("("),
				prettier.Text(""),
				prettier.Text(")"),
				prettier.Space,
				prettier.Text("&"),
				prettier.Text("T"),
			},
			ty.Doc(),
		)
	})

	t.Run("auth with mapping", func(t *testing.T) {

		t.Parallel()

		ty := &ReferenceType{
			Authorization: &MappedAccess{
				EntitlementMap: &NominalType{
					Identifier: Identifier{
						Identifier: "X",
					},
				},
			},
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "T",
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("auth"),
				prettier.Text("("),
				prettier.Text("mapping "),
				prettier.Text("X"),
				prettier.Text(")"),
				prettier.Space,
				prettier.Text("&"),
				prettier.Text("T"),
			},
			ty.Doc(),
		)
	})

	t.Run("auth with nil mapping", func(t *testing.T) {

		t.Parallel()

		ty := &ReferenceType{
			Authorization: &MappedAccess{
				EntitlementMap: nil,
			},
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "T",
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("auth"),
				prettier.Text("("),
				prettier.Text("mapping "),
				prettier.Text(""),
				prettier.Text(")"),
				prettier.Space,
				prettier.Text("&"),
				prettier.Text("T"),
			},
			ty.Doc(),
		)
	})

	t.Run("auth with 2 conjunctive entitlements", func(t *testing.T) {

		t.Parallel()

		ty := &ReferenceType{
			Authorization: &ConjunctiveEntitlementSet{
				Elements: []*NominalType{
					{
						Identifier: Identifier{
							Identifier: "X",
						},
					},
					{
						Identifier: Identifier{
							Identifier: "Y",
						},
					},
				},
			},
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "T",
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("auth"),
				prettier.Text("("),
				prettier.Text("X"),
				prettier.Text(", "),
				prettier.Text("Y"),
				prettier.Text(")"),
				prettier.Space,
				prettier.Text("&"),
				prettier.Text("T"),
			},
			ty.Doc(),
		)
	})

	t.Run("auth with 2 disjunctive entitlements", func(t *testing.T) {

		t.Parallel()

		ty := &ReferenceType{
			Authorization: &DisjunctiveEntitlementSet{
				Elements: []*NominalType{
					{
						Identifier: Identifier{
							Identifier: "X",
						},
					},
					{
						Identifier: Identifier{
							Identifier: "Y",
						},
					},
				},
			},
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "T",
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("auth"),
				prettier.Text("("),
				prettier.Text("X"),
				prettier.Text(" | "),
				prettier.Text("Y"),
				prettier.Text(")"),
				prettier.Space,
				prettier.Text("&"),
				prettier.Text("T"),
			},
			ty.Doc(),
		)
	})

	t.Run("un-auth", func(t *testing.T) {

		t.Parallel()

		ty := &ReferenceType{
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "T",
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("&"),
				prettier.Text("T"),
			},
			ty.Doc(),
		)
	})

	t.Run("un-auth, nil type", func(t *testing.T) {

		t.Parallel()

		ty := &ReferenceType{}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("&"),
				prettier.Text(""),
			},
			ty.Doc(),
		)
	})
}

func TestReferenceType_String(t *testing.T) {

	t.Parallel()

	t.Run("auth with entitlement", func(t *testing.T) {

		t.Parallel()

		ty := &ReferenceType{
			Authorization: &ConjunctiveEntitlementSet{
				Elements: []*NominalType{
					{
						Identifier: Identifier{
							Identifier: "X",
						},
					},
				},
			},
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "T",
				},
			},
		}

		assert.Equal(t,
			"auth(X) &T",
			ty.String(),
		)
	})

	t.Run("auth with nil entitlement", func(t *testing.T) {

		t.Parallel()

		ty := &ReferenceType{
			Authorization: &ConjunctiveEntitlementSet{
				Elements: []*NominalType{
					nil,
				},
			},
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "T",
				},
			},
		}

		assert.Equal(t,
			"auth() &T",
			ty.String(),
		)
	})

	t.Run("auth with mapping", func(t *testing.T) {

		t.Parallel()

		ty := &ReferenceType{
			Authorization: &MappedAccess{
				EntitlementMap: &NominalType{
					Identifier: Identifier{
						Identifier: "X",
					},
				},
			},
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "T",
				},
			},
		}

		assert.Equal(t,
			"auth(mapping X) &T",
			ty.String(),
		)
	})

	t.Run("auth with nil mapping", func(t *testing.T) {

		t.Parallel()

		ty := &ReferenceType{
			Authorization: &MappedAccess{
				EntitlementMap: nil,
			},
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "T",
				},
			},
		}

		assert.Equal(t,
			"auth(mapping ) &T",
			ty.String(),
		)
	})

	t.Run("auth with 2 conjunctive entitlements", func(t *testing.T) {

		t.Parallel()

		ty := &ReferenceType{
			Authorization: &ConjunctiveEntitlementSet{
				Elements: []*NominalType{
					{
						Identifier: Identifier{
							Identifier: "X",
						},
					},
					{
						Identifier: Identifier{
							Identifier: "Y",
						},
					},
				},
			},
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "T",
				},
			},
		}

		assert.Equal(t,
			"auth(X, Y) &T",
			ty.String(),
		)
	})

	t.Run("auth with 2 disjunctive entitlements", func(t *testing.T) {

		t.Parallel()

		ty := &ReferenceType{
			Authorization: &DisjunctiveEntitlementSet{
				Elements: []*NominalType{
					{
						Identifier: Identifier{
							Identifier: "X",
						},
					},
					{
						Identifier: Identifier{
							Identifier: "Y",
						},
					},
				},
			},
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "T",
				},
			},
		}

		assert.Equal(t,
			"auth(X | Y) &T",
			ty.String(),
		)
	})

	t.Run("un-auth", func(t *testing.T) {

		t.Parallel()

		ty := &ReferenceType{
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "T",
				},
			},
		}

		assert.Equal(t,
			"&T",
			ty.String(),
		)
	})

	t.Run("un-auth, nil type", func(t *testing.T) {

		t.Parallel()

		ty := &ReferenceType{}

		assert.Equal(t,
			"&",
			ty.String(),
		)
	})
}

func TestReferenceType_MarshalJSON(t *testing.T) {

	t.Parallel()

	ty := &ReferenceType{
		Authorization: &ConjunctiveEntitlementSet{
			Elements: []*NominalType{
				{
					Identifier: Identifier{
						Identifier: "X",
					},
				},
				{
					Identifier: Identifier{
						Identifier: "Y",
					},
				},
			},
		},
		Type: &NominalType{
			Identifier: Identifier{
				Identifier: "AB",
				Pos:        Position{Offset: 1, Line: 2, Column: 3},
			},
		},
		StartPos: Position{Offset: 4, Line: 5, Column: 6},
	}

	actual, err := json.Marshal(ty)
	require.NoError(t, err)

	assert.JSONEq(t,
		// language=json
		`
        {
            "Type": "ReferenceType",
			"LegacyAuthorized": false,
            "Authorization": {
				 "ConjunctiveElements": [
					{ 
						"Type": "NominalType",
						"Identifier": {
							"Identifier": "X",
							"StartPos": {"Offset": 0, "Line": 0, "Column": 0},
							"EndPos": {"Offset": 0, "Line": 0, "Column": 0}
						},
						"StartPos": {"Offset": 0, "Line": 0, "Column": 0},
						"EndPos": {"Offset": 0, "Line": 0, "Column": 0}
					}, 
					{ 
						"Type": "NominalType",
						"Identifier": {
							"Identifier": "Y",
							"StartPos": {"Offset": 0, "Line": 0, "Column": 0},
							"EndPos": {"Offset": 0, "Line": 0, "Column": 0}
						},
						"StartPos": {"Offset": 0, "Line": 0, "Column": 0},
						"EndPos": {"Offset": 0, "Line": 0, "Column": 0}
					}
				]
			},
            "ReferencedType": {
                "Type": "NominalType",
                "Identifier": {
                    "Identifier": "AB",
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 2, "Line": 2, "Column": 4}
                },
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 2, "Line": 2, "Column": 4}
            },
            "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
            "EndPos": {"Offset": 2, "Line": 2, "Column": 4}
        }
        `,
		string(actual),
	)
}

func TestIntersectionType_Doc(t *testing.T) {

	t.Parallel()

	t.Run("no types", func(t *testing.T) {
		t.Parallel()

		ty := &IntersectionType{}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("{"),
					prettier.Indent{
						Doc: prettier.Concat{
							prettier.SoftLine{},
						},
					},
					prettier.SoftLine{},
					prettier.Text("}"),
				},
			},
			ty.Doc(),
		)
	})

	t.Run("nil type", func(t *testing.T) {
		t.Parallel()

		ty := &IntersectionType{
			Types: []*NominalType{
				nil,
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("{"),
					prettier.Indent{
						Doc: prettier.Concat{
							prettier.SoftLine{},
							prettier.Text(""),
						},
					},
					prettier.SoftLine{},
					prettier.Text("}"),
				},
			},
			ty.Doc(),
		)
	})

	t.Run("with types", func(t *testing.T) {
		t.Parallel()

		ty := &IntersectionType{
			Types: []*NominalType{
				{
					Identifier: Identifier{
						Identifier: "CD",
					},
				},
				{
					Identifier: Identifier{
						Identifier: "EF",
					},
				},
			},
		}

		assert.Equal(t,
			prettier.Group{
				Doc: prettier.Concat{
					prettier.Text("{"),
					prettier.Indent{
						Doc: prettier.Concat{
							prettier.SoftLine{},
							prettier.Text("CD"),
							prettier.Text(","),
							prettier.Line{},
							prettier.Text("EF"),
						},
					},
					prettier.SoftLine{},
					prettier.Text("}"),
				},
			},
			ty.Doc(),
		)
	})
}

func TestIntersectionType_String(t *testing.T) {

	t.Parallel()

	t.Run("no types", func(t *testing.T) {
		t.Parallel()

		ty := &IntersectionType{}

		assert.Equal(t,
			"{}",
			ty.String(),
		)
	})

	t.Run("nil type", func(t *testing.T) {
		t.Parallel()

		ty := &IntersectionType{
			Types: []*NominalType{
				{
					Identifier: Identifier{
						Identifier: "T",
					},
				},
				nil,
			},
		}

		assert.Equal(t,
			"{T, }",
			ty.String(),
		)
	})

	t.Run("with types", func(t *testing.T) {
		t.Parallel()

		ty := &IntersectionType{
			Types: []*NominalType{
				{
					Identifier: Identifier{
						Identifier: "CD",
					},
				},
				{
					Identifier: Identifier{
						Identifier: "EF",
					},
				},
			},
		}

		assert.Equal(t,
			"{CD, EF}",
			ty.String(),
		)
	})
}

func TestIntersectionType_MarshalJSON(t *testing.T) {

	t.Parallel()

	ty := &IntersectionType{
		Types: []*NominalType{
			{
				Identifier: Identifier{
					Identifier: "CD",
					Pos:        Position{Offset: 4, Line: 5, Column: 6},
				},
			},
			{
				Identifier: Identifier{
					Identifier: "EF",
					Pos:        Position{Offset: 7, Line: 8, Column: 9},
				},
			},
		},
		Range: Range{
			StartPos: Position{Offset: 10, Line: 11, Column: 12},
			EndPos:   Position{Offset: 13, Line: 14, Column: 15},
		},
	}

	actual, err := json.Marshal(ty)
	require.NoError(t, err)

	assert.JSONEq(t,
		// language=json
		`
        {
            "Type": "IntersectionType",
			"LegacyRestrictedType": null,
            "Types": [
                {
                    "Type": "NominalType",
                    "Identifier": {
                        "Identifier": "CD",
                        "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                        "EndPos": {"Offset": 5, "Line": 5, "Column": 7}
                    },
                    "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                    "EndPos": {"Offset": 5, "Line": 5, "Column": 7}
                },
                {
                    "Type": "NominalType",
                    "Identifier": {
                        "Identifier": "EF",
                        "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                        "EndPos": {"Offset": 8, "Line": 8, "Column": 10}
                    },
                    "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                    "EndPos": {"Offset": 8, "Line": 8, "Column": 10}
                }
            ],
            "StartPos": {"Offset": 10, "Line": 11, "Column": 12},
            "EndPos": {"Offset": 13, "Line": 14, "Column": 15}
        }
        `,
		string(actual),
	)
}

func TestInstantiationType_Doc(t *testing.T) {

	t.Parallel()

	t.Run("with type, no type arguments", func(t *testing.T) {
		t.Parallel()

		ty := &InstantiationType{
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "AB",
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("AB"),
				prettier.Group{
					Doc: prettier.Concat{
						prettier.Text("<"),
						prettier.Indent{
							Doc: prettier.Concat{
								prettier.SoftLine{},
							},
						},
						prettier.SoftLine{},
						prettier.Text(">"),
					},
				},
			},
			ty.Doc(),
		)
	})

	t.Run("nil type, no type arguments", func(t *testing.T) {
		t.Parallel()

		ty := &InstantiationType{}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text(""),
				prettier.Group{
					Doc: prettier.Concat{
						prettier.Text("<"),
						prettier.Indent{
							Doc: prettier.Concat{
								prettier.SoftLine{},
							},
						},
						prettier.SoftLine{},
						prettier.Text(">"),
					},
				},
			},
			ty.Doc(),
		)
	})

	t.Run("with type, type arguments", func(t *testing.T) {
		t.Parallel()

		ty := &InstantiationType{
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "AB",
				},
			},
			TypeArguments: []*TypeAnnotation{
				{
					IsResource: true,
					Type: &NominalType{
						Identifier: Identifier{
							Identifier: "CD",
						},
					},
				},
				{
					IsResource: false,
					Type: &NominalType{
						Identifier: Identifier{
							Identifier: "EF",
						},
					},
				},
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("AB"),
				prettier.Group{
					Doc: prettier.Concat{
						prettier.Text("<"),
						prettier.Indent{
							Doc: prettier.Concat{
								prettier.SoftLine{},
								prettier.Concat{
									prettier.Text("@"),
									prettier.Text("CD"),
								},
								prettier.Text(","),
								prettier.Line{},
								prettier.Text("EF"),
							},
						},
						prettier.SoftLine{},
						prettier.Text(">"),
					},
				},
			},
			ty.Doc(),
		)
	})

	t.Run("with type, nil type argument", func(t *testing.T) {
		t.Parallel()

		ty := &InstantiationType{
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "AB",
				},
			},
			TypeArguments: []*TypeAnnotation{
				nil,
			},
		}

		assert.Equal(t,
			prettier.Concat{
				prettier.Text("AB"),
				prettier.Group{
					Doc: prettier.Concat{
						prettier.Text("<"),
						prettier.Indent{
							Doc: prettier.Concat{
								prettier.SoftLine{},
								prettier.Text(""),
							},
						},
						prettier.SoftLine{},
						prettier.Text(">"),
					},
				},
			},
			ty.Doc(),
		)
	})
}

func TestInstantiationType_String(t *testing.T) {

	t.Parallel()

	t.Run("with type, no type arguments", func(t *testing.T) {
		t.Parallel()

		ty := &InstantiationType{
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "AB",
				},
			},
		}

		assert.Equal(t,
			"AB<>",
			ty.String(),
		)
	})

	t.Run("nil type, no type arguments", func(t *testing.T) {
		t.Parallel()

		ty := &InstantiationType{}

		assert.Equal(t,
			"<>",
			ty.String(),
		)
	})

	t.Run("with type, type arguments", func(t *testing.T) {
		t.Parallel()

		ty := &InstantiationType{
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "AB",
				},
			},
			TypeArguments: []*TypeAnnotation{
				{
					IsResource: true,
					Type: &NominalType{
						Identifier: Identifier{
							Identifier: "CD",
						},
					},
				},
				{
					IsResource: false,
					Type: &NominalType{
						Identifier: Identifier{
							Identifier: "EF",
						},
					},
				},
			},
		}

		assert.Equal(t,
			"AB<@CD, EF>",
			ty.String(),
		)
	})

	t.Run("with type, nil type argument", func(t *testing.T) {
		t.Parallel()

		ty := &InstantiationType{
			Type: &NominalType{
				Identifier: Identifier{
					Identifier: "AB",
				},
			},
			TypeArguments: []*TypeAnnotation{
				nil,
			},
		}

		assert.Equal(t,
			"AB<>",
			ty.String(),
		)
	})
}

func TestInstantiationType_MarshalJSON(t *testing.T) {

	t.Parallel()

	ty := &InstantiationType{
		Type: &NominalType{
			Identifier: Identifier{
				Identifier: "AB",
				Pos:        Position{Offset: 1, Line: 2, Column: 3},
			},
		},
		TypeArguments: []*TypeAnnotation{
			{
				IsResource: false,
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "CD",
						Pos:        Position{Offset: 4, Line: 5, Column: 6},
					},
				},
				StartPos: Position{Offset: 7, Line: 8, Column: 9},
			},
			{
				IsResource: false,
				Type: &NominalType{
					Identifier: Identifier{
						Identifier: "EF",
						Pos:        Position{Offset: 10, Line: 11, Column: 12},
					},
				},
				StartPos: Position{Offset: 13, Line: 14, Column: 15},
			},
		},
		TypeArgumentsStartPos: Position{Offset: 16, Line: 17, Column: 18},
		EndPos:                Position{Offset: 19, Line: 20, Column: 21},
	}

	actual, err := json.Marshal(ty)
	require.NoError(t, err)

	assert.JSONEq(t,
		// language=json
		`
        {
            "Type": "InstantiationType",
            "InstantiatedType": {
                "Type": "NominalType",
                "Identifier": {
                    "Identifier": "AB",
                    "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                    "EndPos": {"Offset": 2, "Line": 2, "Column": 4}
                },
                "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
                "EndPos": {"Offset": 2, "Line": 2, "Column": 4}
            },
            "TypeArguments": [
                {
                    "IsResource": false,
                    "AnnotatedType": {
                        "Type": "NominalType",
                        "Identifier": {
                            "Identifier": "CD",
                            "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                            "EndPos": {"Offset": 5, "Line": 5, "Column": 7}
                        },
                        "StartPos": {"Offset": 4, "Line": 5, "Column": 6},
                        "EndPos": {"Offset": 5, "Line": 5, "Column": 7}
                    },
                    "StartPos": {"Offset": 7, "Line": 8, "Column": 9},
                    "EndPos": {"Offset": 5, "Line": 5, "Column": 7}
                },
                {
                    "IsResource": false,
                    "AnnotatedType": {
                        "Type": "NominalType",
                        "Identifier": {
                            "Identifier": "EF",
                            "StartPos": {"Offset": 10, "Line": 11, "Column": 12},
                            "EndPos": {"Offset": 11, "Line": 11, "Column": 13}
                        },
                        "StartPos": {"Offset": 10, "Line": 11, "Column": 12},
                        "EndPos": {"Offset": 11, "Line": 11, "Column": 13}
                    },
                    "StartPos": {"Offset": 13, "Line": 14, "Column": 15},
                    "EndPos": {"Offset": 11, "Line": 11, "Column": 13}
               }
            ],
            "TypeArgumentsStartPos": {"Offset": 16, "Line": 17, "Column": 18},
            "StartPos": {"Offset": 1, "Line": 2, "Column": 3},
            "EndPos": {"Offset": 19, "Line": 20, "Column": 21}
        }
        `,
		string(actual),
	)
}
