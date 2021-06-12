---
title: Syntax
---

## Comments

Comments can be used to document code.
A comment is text that is not executed.

*Single-line comments* start with two slashes (`//`).
These comments can go on a line by themselves or they can go directly after a line of code.

```cadence
// This is a comment on a single line.
// Another comment line that is not executed.

let x = 1  // Here is another comment after a line of code.
```

*Multi-line comments* start with a slash and an asterisk (`/*`)
and end with an asterisk and a slash (`*/`):

```cadence
/* This is a comment which
spans multiple lines. */
```

Comments may be nested.

```cadence
/* /* this */ is a valid comment */
```

Multi-line comments are balanced.

```cadence
/* this is a // comment up to here */ this is not part of the comment */
```

### Documentation Comments
Documentation comments (also known as "doc-strings" ro "doc-comment") are a special set of comments that would be
processed by various tools to generate human-readable documentations for cadence programs.

Single line doc-comments starts with three slashes (`///`)
```cadence
/// This is a documnetation comment on a single line.
/// Another documnetation comment line that is not executed.

let x = 1
```

Multi-line doc-comments comments start with a slash followed by two asterisks (`/**`)
```cadence
/** This is a documnetation comment
 which spans multiple lines. **/
```

## Names

Names may start with any upper or lowercase letter (A-Z, a-z)
or an underscore (`_`).
This may be followed by zero or more upper and lower case letters,
underscores, and numbers (0-9).
Names may not begin with a number.

```cadence
// Valid: title-case
//
PersonID

// Valid: with underscore
//
token_name

// Valid: leading underscore and characters
//
_balance

// Valid: leading underscore and numbers
_8264

// Valid: characters and number
//
account2

// Invalid: leading number
//
1something

// Invalid: invalid character #
_#1

// Invalid: various invalid characters
//
!@#$%^&*
```

### Conventions

By convention, variables, constants, and functions have lowercase names;
and types have title-case names.

## Semicolons

Semicolons (;) are used as separators between declarations and statements.
A semicolon can be placed after any declaration and statement,
but can be omitted between declarations and if only one statement appears on the line.

Semicolons must be used to separate multiple statements if they appear on the same line.

```cadence
// Declare a constant, without a semicolon.
//
let a = 1

// Declare a variable, with a semicolon.
//
var b = 2;

// Declare a constant and a variable on a single line, separated by semicolons.
//
let d = 1; var e = 2
```
