# Cadence <- Style Guide
## Table of contents
1. [Identation](#identation)
1. [Line length](#line-length)
1. [Comments](#comments)
1. [Whitespace between code lines](#whitespace-between-code-lines)
1. [Declaring variables and constants](#whitespace-between-code-lines)
1. [Bracket spaces](#bracket-spaces)
1. [Panic messages](#panic-messages)
1. [Naming and capitalization](#naming-and-capitalization)
## Identation
Use 4 spaces per indentation level.
Prefer spaces over tabs as indentation method, mixing them should be avoided.
## Line length
A line lenght of 80 characters is recomended for Cadence. This could be particurally challenging when facing capability-related lines, for instance:
```cadence
letResourceRef = account.getCapability(ContractName.SomePathName).borrow<&SomeResource{ContractName.InterfaceName}>()
```
This generic line of code that could be found in a pretty similar way in lots of transactions that need to borrow a reference to a certain object from a capability it's 118 characters long by its own without any identation. Keeping this lines readable should be a main objective of any Cadence developer. There are some simple patterns that could be observed in order to accomplish this:
+ Prefer allways to break the line before the borrow call.
```cadence
letResourceRef = account.getCapability(ContractName.SomePathName)
                  .borrow<&SomeResource{ContractName.InterfaceName}>()
```
+ If identation or a long path name makes the "getCapability" line go further than 80 characters an additional line break can be included
```cadence
letResourceRef = account
                  .getCapability(ContractName.SomeAbsurdlyLongForSomeReasonPathName)
                  .borrow<&SomeResource{ContractName.InterfaceName}>()
```
Keep allways those concatenated function calls idented from the object they are being called from.
## Comments
Comments are a vital part of smart contracts, allowing users to easily understand what they are getting involve with. 
### High level documentation at the begining of files (contracts, transactions and scripts)
Top Level comments and comments for types, fields, events, and functions should use `///` (three slashes) because there is a cadence docs generating tool that picks up three slash comments to auto-generate docs.
### Commenting functions
Functions should be commented with a:
  * Description
  * Parameter descriptions (unless it is obvious)
  * Return value descriptions (unless it is obvious)
Regular comments within functions should only use two slashes (`//`)
## Whitespace between code lines
## Declaring variables and constants
## Bracket spaces
## Panic messages
## Naming and capitalization
### Contracts
Contract and contract interfaces names should allways follow PascalCase, for instance:
```cadence
pub contract ExampleToken
```
### Fields
```cadence
pub var totalSupply
```
```cadence
pub var VaultStoragePath
```
// Different rule for regular fields than for path fields?
### Functions
Functions names should allways follow camelCase, for instance:
```cadence
pub fun mintTokens
```
### Paths
