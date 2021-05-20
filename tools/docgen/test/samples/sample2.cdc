/// NFT is a dummy non-fungible token contract.
///
pub contract NFT {

    event TestEvent(x: Int, y: Int)

    /// A variable fields
    var field1: Int

    /// A constant field
    let field2: String

    /// This is a foo function,
    /// This doesn't have a return type.
    fun foo(a: Int, b: String) {
    }

    /// This is a bar function, with a return type
    /// @param name: The name. Must be a string
    /// @param bytes: Content
    /// @returns Validity
    fun bar(name: String, bytes: [Int8]): bool {
    }

    fun noDocsFunction() {
    }

    /// This is some struct. It has
    /// @field x: a string field
    /// @field y: a map of int and any-struct
    struct SomeStruct {
        var x: String
        var y: {Int: AnyStruct}

        /// Can be used to construct a 'SomeStruct'
        init() {
        }

        /// This is a nested struct.
        struct InnerStruct {
            var a: Int
            var b: String
        }
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
}
