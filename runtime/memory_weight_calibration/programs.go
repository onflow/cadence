package main

var bool_test = `
pub fun main() {
	var i = 0
	while i < 10000 {
		let b = true || false
		i = i + 1
	}
}
`

var nil_test = `
pub fun main() {
	var i = 0
	while i < 10000 {
		let v = nil ?? nil
		i = i + 1
	}
}
`

var string_test = `
pub fun main() {
	var i = 0
	while i < 10000 {
		let v = "x".toLower()
		i = i + 1
	}
}
`

var char_test = `
pub fun main() {
	var i = 0
	while i < 10000 {
		let v: Character = "x"[0]
		i = i + 1
	}
}
`

var ephemeral_ref_test = `
pub fun main() {
	var i: Int64 = 0
	while i < 10000 {
		let v = &i as &Int64
		i = i + 1
	}
}
`

var int_test = `
pub fun main() {
	var i = 0
	while i < 10000 {
		let v = 1 + 1
		i = i + 1
	}
}
`

var float_test = `
pub fun main() {
	var i = 0
	while i < 10000 {
		let v = 3.2 + 1.1
		i = i + 1
	}
}
`

var meta_type_test = `
pub fun main() {
	var i = 0
	while i < 10000 {
		let t = i.getType()
		i = i + 1
	}
}
`

/*var block_test = `
pub fun main() {
	var i = 0
	while i < 10000 {
		let t = {}
		i = i + 1
	}
}
`*/

var path_test = `
pub fun main() {
	var i = 0
	while i < 10000 {
		let t: StoragePath = /storage/foo
		i = i + 1
	}
}
`

var address_test = `
pub fun main() {
	var i = 0
	while i < 10000 {
		let t: Address = 0x1
		i = i + 1
	}
}
`

/*var link_test = `
pub fun main() {
	var i = 0
	while i < 10000 {
		let t = Link<Int>(/storage/foo)
		i = i + 1
	}
}
`*/

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

var iteration_test = `
pub fun main() {
	var i = 0
	var a: [String] = []
	while i < 10000 {
		a.append("a")
		i = i + 1
	}
	for s in a {
		let x = s
	}
}
`

var composite_test = `
pub struct S {}

pub fun main() {
	var i = 0
	while i < 10000 {
		let s = S()
		i = i + 1
	}
}
`

var composite_field_test = `
pub struct S {
	pub let x: Bool
	init(x: Bool) {
		self.x = x
	}
}

pub fun main() {
	var i = 0
	while i < 10000 {
		let s = S(x: false)
		i = i + 1
	}
}
`

var optional_test = `
pub fun main() {
	var i = 0
	while i < 10000 {
		let b: Bool? = true
		i = i + 1
	}
}
`

var function_test = `
pub fun main() {
	var i = 0
	while i < 10000 {
		let f = fun() {}
		i = i + 1
	}
}
`

var bound_function_test = `
pub struct S {
	fun foo() {}
}
pub fun main() {
	var i = 0
	while i < 10000 {
		let s = S()
		let f = s.foo
		i = i + 1
	}
}
`

var test_programs = []struct {
	name string
	code string
}{
	{name: "bool", code: bool_test},
	{name: "nil", code: nil_test},
	{name: "string", code: string_test},
	{name: "char", code: char_test},
	{name: "int", code: int_test},
	{name: "float", code: float_test},
	{name: "path", code: path_test},
	{name: "address", code: address_test},
	{name: "function", code: function_test},
	//{name: "link", code: link_test},
	{name: "ephemeral ref", code: ephemeral_ref_test},
	{name: "meta type", code: meta_type_test},
	//{name: "block", code: block_test},
	{name: "array", code: array_test},
	{name: "dict", code: dict_test},
	{name: "optional bool", code: optional_test},
	{name: "empty composite", code: composite_test},
	{name: "bound function", code: bound_function_test},
	//{name: "iteration", code: iteration_test},
	//{name: "composite with field", code: composite_field_test},
}
