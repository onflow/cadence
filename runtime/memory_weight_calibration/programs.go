package main

var bool_test = `
pub fun main() {
	var i = 0
	while i < 10000 {
		let b = true
		i = i + 1
	}
}
`

var void_test = `
pub fun main() {
	var i = 0
	while i < 10000 {
		let v = nil
		i = i + 1
	}
}
`

var dict_test = `
pub fun main() {
	var i = 0
	while i < 10000 {
		let v: {String: String} = {}
		i = i + 1
	}
}
`

var array_test = `
pub fun main() {
	var i = 0
	while i < 10000 {
		let a: [Int8] = []
		i = i + 1
	}
}
`

var test_programs = []struct {
	name string
	code string
}{
	{name: "bool", code: bool_test},
	{name: "void", code: void_test},
	{name: "array", code: array_test},
	{name: "dict", code: dict_test},
}
