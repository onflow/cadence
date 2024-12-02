# Syntax Notation

The syntax file is a machine-readable file that describes the scanner and parser of a compiler frontend. This description is utilized to generate the frontend and the syntax tree.


## Scanner Section

The start of the section is denoted by a line containing only the text `%scanner`.

### Token Rule

Tokens are defined and referenced using angled brackets (`<` and `>`) surrounding their names. They are the sole scanner rules permitted in the parser. Tokens can have multiple productions separated by a pipe character (`|`). They can be marked with an asterisk (`*`) preceding the first angle bracket to indicate that when the token is matched, it is discarded.


```c
*
<spaceToken>:
        whitespace
    ;
```

### Normal Rule

Normal rules can have any number of productions separated by a pipe character (|) with each production containing one or more elements.


```c
whitespaceItem:
        singleLineComment
    |   multipleLineComment
    |   whitespaceCharacter
    |   lineBreak
    ;
```

### Set Rule

Set rules consist of a sequence of intervals separated by a comma (`,`). These intervals can either represent a single Unicode code point or a range of code points, separated by a hyphen (`-`).


```c
whitespaceCharacter:
        U+0000,U+0009,U+000B-U+000C,U+0020
    ;
```

## Scanner Production Elements

Scanner productions comprise one or more elements. Scanner elements can be either a nonterminal or a set. A repeat phrase is used
to create an optional or list element. It has a minimum and maximum count separated by an asterisk (`*`). The tilde (`~`) prefix of a scanner element matches one character as long as the element is not matched.

```c
<someToken>:
        *1optional
    |   1*oneOrMore
    |   *zeroOrMore
    |   2*3explictRange
    |   ~notThis
    ;
```

### Set

Scanner sets consist of a string of characters such that the element can be matched by any singular character in that string.

```c
identifierFollower:
        "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz"
    ;
```

### Nonterminal

Used to refer to any normal scanner rule.

```c
<whitespaceToken>:
        whitespace
    ;

whitespace:
        ...
    ;
```

## Parser Section

The start of the section is denoted by a line containing only the text `%parser`.

```c
%parser
```

### Terminal Rule

Parser terminal rules match the given token.

```c
identifier:
        <identifierToken>
    ;
```

### Select Rule

Parser select rules consist of a list of nonterminals separated by a pipe (`|`). Matches one of the listed nonterminals.

```c
fullType:
        fullTypeNormal
    |   fullTypeNested
    ;
```

### Empty Rule

Parser empty rules always match. Has no productions.

```c
empty:
    ;
```

### Nonterminal Rule

Parser nonterminal rules may only have one production with multiple elements that are matched in sequence. Element names must be unique.

```c
importDeclaration:
        importKeyword *1importDeclarationFromPhrase importDeclarationSource
    ;
```

## Parser Production Elements

### Literal

Parser literals match the given text.

```c
importKeyword:
        "import"
    ;

emptyBody:
        "{" "}"
    ;
```

### Nonterminal

Parser nonterminal elements match one occurrence of the nonterminal.

```c
single:
        item
    ;
```

### List

Parser nonterminal elements may be prefixed with `<minimum>*<maximum>` to match the nonterminal multiple times. The first number can be omitted for zero, and the second can be omitted for no maximum limit.

```c
oneOrMore:
        1*item
    ;

zeroOrMore:
        *item
    ;

fewerThanFive:
        *5item
    ;

betweenFourAndEight:
        4*8item
    ;
```

### Optional

Use `*1` before a parser nonterminal element to indicate that it is optional.

```c
dictionaryLiteral:
        "{" *1dictionaryEntryList "}"
    ;
```

### Separated

Lists can specify a separator that must be matched between multiple nonterminal matches. The separator may be either a literal or a nonterminal.

```c
orExpression:
        1*andExpression( "||" )
    ;

multiplicativeExpression:
        1*moveExpression( multiplicativeOp )
    ;
```

### End of Input

Use a dollar sign (`$`) to match the end of input.

```c
*
program:
         *declaration( declarationSeparator ) $
    ;
```

### Adjacent

Use a period (`.`) to match if the following token starts immediately after the preceding token.

```c
shiftRight:
        ">" . ">"
    ;
```

### Not Adjacent

Use an underscore (`_`) to match if the following token **does not** start immediately after the preceding token.

```c
infixAdd:
        _ "+" _
    ;
```

### Same Line

Use a backslash (`\`) to match if the following token is on the same line as the preceding token.

```c
forceUnwrapOp:
        \ "!"
    ;
```

### Different Line

Use a forward slash (`/`) to match if the following token **is not** on the same line as the preceding token.

```c
newLine:
        /
    ;
```
