pub struct Test {
    /// This is a test function.
    pub fun nothing() {}

    /// This is a test function with parameters.
    pub fun params(a: Int, _ b: String) {}

    /// This is a test function with a return type.
    pub fun returnBool(): Bool {}

    /// This is a test function with parameters and a return type.
    pub fun paramsAndReturn(a: Int, _ b: String): Bool {}

    /// This is a test function with a type parameter.
    // TODO: pub fun typeParam<T>()
}
