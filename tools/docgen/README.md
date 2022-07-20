# Cadence Documentation Generator

A tool to generate human-readable documentation for Cadence programs.
Supports generating documentation for following declarations:
- Composite types (Contracts, Structs, Resources, Enums)
- Interfaces (Contract interfaces, Struct interfaces, Resource Interfaces)
- Functions and Parameters
- Event declarations


The tool currently supports generating documentation in Markdown format.

## How To Run
Navigate to `<cadence_dir>/tools/docgen/cmd` directory and run:
```
go run main.go <path_to_cadence_file> <output_dir>
```

## Documentation Comments Format
The documentation comments ("doc-strings" / "doc-comments": line comments starting with `///`,
or block comments starting with `/**`) available in Cadence programs are processed by the tool,
to produce human-readable documentations.

### Markdown Support
Standard Markdown format is supported in doc-comments, with a bit of Cadence flavour.
This means, any Markdown syntax used within the comments would be honoured and rendered like a standard Markdown snippet.
It gives the flexibility for the developers to write well-structured documentations.

e.g: A set of bullet points added using Markdown bullets syntax would be rendered as bullet points in the
generated documentation as well.

Documentation Comment:
```
/// This is the description of the function. You can use markdown syntax here.
/// Can use **bold** or _italic_ texts, or even bullet-points:
///   - Here's the first point.
///   - Can also use code snippets (eg: `a + b`)
  ```
Output:

>This is the description of the function. You can use markdown syntax here.<br/>
>Can use **bold** or _italic_ texts, or even bullet-points:
>   - Here's the first point.
>   - Can also use code snippets (eg: `a + b`)


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
- Use inline-codes (within backticks `` `foo` ``) when referring to names/identifiers (such as function names,
  parameter names, etc.) in the code.
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
