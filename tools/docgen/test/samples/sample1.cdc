/// This is a foo function
fun foo(a: Int, b: String) {
}

/// This is a bar function
fun bar(name: String, bytes: [Int8]): bool {
}

/// This is some struct. It has
/// @field x: a string field
/// @field y: a map of int and any-struct
struct Some {
	var x: String
	var y: {Int: AnyStruct}
}

/// This is an Enum without type conformance.
enum Direction {
	case LEFT
	case RIGHT
}

/// This is an Enum, with explicit type conformance.
enum Color: Int8 {
	case Red
	case Blue
}