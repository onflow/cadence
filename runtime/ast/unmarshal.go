package ast

import (
	"encoding/json"
)

func jsonMarshalAndVerify(i interface{}) ([]byte, error) {
	return json.Marshal(i)
}
