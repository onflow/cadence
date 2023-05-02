pub struct Test {
    /// This is a test integer.
    let testInt: UInt64

    /// This is a test optional integer.
    let testOptInt: UInt64?

    /// This is a test integer reference.
    let testRefInt: &UInt64

    /// This is a test variable-sized integer array.
    let testVarInts: [UInt64]

    /// This is a test constant-sized integer array.
    let testConstInts: [UInt64; 2]

    /// This is a test parameterized-type field.
    let testParam: Foo<Bar>

    /// This is a test address field.
    let testAddress: Address

    /// This is a test type field.
    let testType: Type

    /// This is a test unparameterized capability field.
    let testCap: Capability

    /// This is a test parameterized capability field.
    let testCapInt: Capability<Int>

    /// This is a test restricted type (without type) field.
    let testRestrictedWithoutType: {Bar, Baz}

    /// This is a test restricted type (with type) field.
    let testRestrictedWithType: Foo{Bar, Baz}

    /// This is a test restricted type (without restrictions) field.
    let testRestrictedWithoutRestrictions: Foo{}
}
