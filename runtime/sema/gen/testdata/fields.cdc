access(all) struct Test {
    /// This is a test integer.
    pub let testInt: UInt64

    /// This is a test optional integer.
    pub let testOptInt: UInt64?

    /// This is a test integer reference.
    pub let testRefInt: &UInt64

    /// This is a test variable-sized integer array.
    pub let testVarInts: [UInt64]

    /// This is a test constant-sized integer array.
    pub let testConstInts: [UInt64; 2]

    /// This is a test parameterized-type field.
    pub let testParam: Foo<Bar>

    /// This is a test address field.
    pub let testAddress: Address

    /// This is a test type field.
    pub let testType: Type

    /// This is a test unparameterized capability field.
    pub let testCap: Capability

    /// This is a test parameterized capability field.
    pub let testCapInt: Capability<Int>

    /// This is a test restricted type (without type) field.
    pub let testRestrictedWithoutType: {Bar, Baz}

    /// This is a test restricted type (with type) field.
    pub let testRestrictedWithType: Foo{Bar, Baz}

    /// This is a test restricted type (without restrictions) field.
    pub let testRestrictedWithoutRestrictions: Foo{}
}
