access(all) struct Test {
    /// This is a test integer.
    access(all) let testInt: UInt64

    /// This is a test optional integer.
    access(all) let testOptInt: UInt64?

    /// This is a test integer reference.
    access(all) let testRefInt: &UInt64

    /// This is a test variable-sized integer array.
    access(all) let testVarInts: [UInt64]

    /// This is a test constant-sized integer array.
    access(all) let testConstInts: [UInt64; 2]

    /// This is a test parameterized-type field.
    access(all) let testParam: Foo<Bar>

    /// This is a test address field.
    access(all) let testAddress: Address

    /// This is a test type field.
    access(all) let testType: Type

    /// This is a test unparameterized capability field.
    access(all) let testCap: Capability

    /// This is a test parameterized capability field.
    access(all) let testCapInt: Capability<Int>

    /// This is a test intersection type (without type) field.
    access(all) let testIntersectionWithoutType: {Bar, Baz}

    /// This is a test intersection type (with type) field.
    access(all) let testIntersectionWithType: Foo{Bar, Baz}

    /// This is a test intersection type (without types) field.
    access(all) let testIntersectionWithoutTypes: Foo{}
}
