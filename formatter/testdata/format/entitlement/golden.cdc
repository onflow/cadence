access(all) contract Foo {
    access(all) entitlement NodeOperator

    access(all) entitlement mapping AccountMapping {
        include Identity
    }
}
