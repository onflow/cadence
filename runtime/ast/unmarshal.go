package ast

import (
	"encoding/json"
	"fmt"
	"reflect"
)

func jsonMarshalAndVerify(i interface{}) ([]byte, error) {
	b, err := json.Marshal(i)
	if err != nil { return b, err }

	expected := string(b)

	unmarshalled := reflect.New(reflect.TypeOf(i)).Interface()
	err = json.Unmarshal(b, &unmarshalled)
	if err != nil { return b, err }

	b2, err2 := json.Marshal(unmarshalled)
	if err2 != nil { return b2, err2 }

	actual := string(b2)

	if expected != actual {
		return nil, fmt.Errorf("un/marshal failed:\n%s\n%s\n", expected, actual)
	}

	return b, err
}
