struct Foo {
    /// foo
    access(all) fun foo()

    /// Bar
    access(all) let bar: Foo.Bar

    struct Bar {
        /// bar
        access(all) fun bar()
    }
}
