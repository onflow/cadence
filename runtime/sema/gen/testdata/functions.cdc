access(all) struct Test {
    /// This is a test function.
    access(all) fun nothing() {}

    /// This is a test function with parameters.
    access(all) fun params(a: Int, _ b: String) {}

    /// This is a test function with a return type.
    access(all) fun returnBool(): Bool {}

    /// This is a test function with parameters and a return type.
    access(all) fun paramsAndReturn(a: Int, _ b: String): Bool {}

    /// This is a test function with a type parameter.
    access(all) fun typeParam<T>() {}

    /// This is a test function with a type parameter and a type bound.
    access(all) fun typeParamWithBound<T: &Any>() {}

    /// This is a test function with a type parameter and a parameter using it.
    access(all) fun typeParamWithBoundAndParam<T>(t: T) {}

    /// This is a function with 'view' modifier
    access(all) view fun viewFunction() {}
}
