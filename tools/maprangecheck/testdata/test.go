package testdata

func testVariable() {
	var m map[string]int

	for range m {}  // want "range statement over map: map\\[string\\]int"
}


func returnMap() map[int]string {
	return nil
}

func testFunc() {
	for range returnMap() {}  // want "range statement over map: map\\[int\\]string"
}

func testTypeDef() {
	type M map[string]int
	var m M
	for range m {}  // want "range statement over map: map\\[string\\]int"
}

func testTypeAlias() {
	type M = map[string]int
	var m M
	for range m {}  // want "range statement over map: map\\[string\\]int"
}

