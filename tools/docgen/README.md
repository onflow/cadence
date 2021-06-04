# Cadence Documentation Generator

This is a tool to generate human-readable documentation for Cadence programs. 
The tool currently supports generating documentation for following declarations:
- Composite types (Contracts, Structs, Resources, Enums)
- Interfaces (Contract interfaces, Struct interfaces, Resource Interfaces)
- Functions and Parameters
- Event declarations


The tool currently supports generating documentation in Markdown format.

## How To Run
`go run <cadence_dir>/tools/docgen/main.go <path_to_cadence_file> <output_dir>`

## Documentation Comments Format
Documentation comments (aka "docstrings": line comments starts with `///`, or block comments starting with `/**`) 
added in a Cadence code are processed by the tool.
[Standard Markdown format](https://www.markdownguide.org/basic-syntax/) is supported in documentation comments, 
with a bit of Cadence flavour.
Any Markdown syntax used within the comments would be honoured and rendered like a standard Markdown snippet.
This gives the flexibility for the developers to write well-structured documentations. 
<br/>
e.g: A set of bullet points added using Markdown bullets syntax, would be rendered as bullet points in the
generated documentation as well.

### Function Documentation
Function documentation may start with a description of the function.
It also supports a special set of tags to document parameters and return types.
Parameters can be documented using the `@param` tag, followed by the parameter name, a colon (`:`) and the parameter description.
The return type can be documented using the `@return` tag.

```
/// This is the description of the function. This function adds two values.
///
/// @param a: First integer value to add
/// @param b: Second integer value to add
/// @return Addition of the two arguments `a` and `b`
///
pub fun add(a: Int, b: Int): Int {
}
```

## Best Practices
- Avoid using headings, horizontal-lines in the documentation.
  - It could potentially conflict with the headers and lines added by the tool, when generating the documentation
  - This may cause the generated documentation to be rendered in a disorganized manner.
- Use inline-codes (within backticks `` `foo` ``) when referring to names/identifiers in the code.
  - e.g: Referring to a function name, parameter names, etc.
    ```
    /// This is the description of the function.
    /// This function adds `a` and `b` values.
    ///
    pub fun add(a: Int, b: Int): Int {
    }
    ```
- When documenting function parameters and return type, avoid mixing parameter/return-type documentations
  with the description of the function. e.g:
  ```
  /// This is the description of the function.
  ///
  /// @param a: First integer value to add
  /// @param b: Second integer value to add
  ///
  /// This function adds two values. However, this is not the proper way to document it.
  /// This part of the description is not in the proper place.
  ///
  /// @return Addition of the two arguments `a` and `b`
  ///
  pub fun add(a: Int, b: Int): Int {
  }
  ```
