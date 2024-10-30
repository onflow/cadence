# Go

## Patterns

- To ensure interface cannot be implemented in other packages,
  add a private function (first character must be lower-case) named "is" + type name,
  which takes no arguments, returns nothing, and has an empty body.

  For example:

  ```go
  type I interface {
      isI()
  }
  ```

  See https://go.dev/doc/faq#guarantee_satisfies_interface

- To ensure a type implements an interface at compile-time,
  use the "interface guard" pattern:
  Introduce a global variable named `_`, type it as the interface,
  and assign an empty value of the concrete type to it.

  For example:

  ```go
  type T struct {
      //...
  }

  var _ io.ReadWriter = (*T)(nil)

  func (t *T) Read(p []byte) (n int, err error) {
  // ...
  ```

  See
  - https://go.dev/doc/faq#guarantee_satisfies_interface
  - https://rednafi.com/go/interface_guards/
  - https://github.com/uber-go/guide/blob/master/style.md#verify-interface-compliance
  - https://medium.com/@matryer/golang-tip-compile-time-checks-to-ensure-your-type-satisfies-an-interface-c167afed3aae

