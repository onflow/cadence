pub struct Test {
    /// This is a test function.
    pub fun nothing() {}

    /// This is a test function with parameters.
    pub fun params(a: Int, _ b: String) {}

    /// This is a test function with a return type.
    pub fun return(): Bool {}

    /// This is a test function with parameters and a return type.
    pub fun paramsAndReturn(a: Int, _ b: String): Bool {}

    /// This is a test function with a type parameter.
    pub fun typeParam<T>() {}

    /// This is a test function with a type parameter and a type bound.
    pub fun typeParamWithBound<T: &Any>() {}

    /// This is a test function with a type parameter and a parameter using it.
    pub fun typeParamWithBoundAndParam<T>(t: T) {}
}
