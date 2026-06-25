# Cadence Feature Inventory

A non-exhaustive, non-authoritative inventory of Cadence language features,
automatically generated from the codebase.

Last updated: 38729bff7536b82974ac9ab9e924345fe9f5ef14

---

## 1. Lexical & Syntactic Foundations

### 1.1 Comments & doc-strings
- Line comments; block comments `/* */` with **nesting** 
- Doc-strings

### 1.2 Literals
- Nil (`nil`)
- Boolean (`true`/`false`)
- Integer literals: decimal, hex `0x`, binary `0b`, octal `0o`, digit grouping `_`
- Fixed-point literals
- String literals; escape sequences `\0 \n \r \t \" \' \\` and Unicode `\u{...}`
- String templates / interpolation `"... \(expr) ..."`
- Array literals `[...]`, dictionary literals `{k: v}`
- Path literals `/domain/identifier`
- Void/unit `()`
- Parser depth limit: 16 levels for expressions and for types

### 1.3 Keywords

- **Active**: `if else while for in break continue return switch case default guard`,
  `fun let var`, `struct resource contract interface event enum attachment`,
  `entitlement mapping`, `import from as`, `transaction prepare execute pre post`,
  `create destroy init`, `emit`, `attach remove to`, `access all account self auth view`,
  `true false nil include`.
- **Soft keywords** (usable as identifiers): `from account all view attach remove to type`.
- **Reserved, unimplemented** (rejected as identifiers, no feature behind them):
  `require requires`, `try catch finally throw throws`, `goto repeat`,
  `const export internal final`, `typealias type where`, `is` (defined as a keyword but
  not consumed by the parser/checker — no type-test operator exists).

---

## 2. Declarations

### 2.1 Composite declarations
- `struct`, `resource`, `contract`, `event`, `enum`, `attachment`

### 2.2 Interfaces
- Struct / resource / contract interfaces
- Interface conformance 
- Interface inheritance (conformance to other interfaces)
- Default function implementations (at most one; cannot override a default)
- Conditions in interfaces; condition linearization across conformances

### 2.3 Members
- Fields (`let`/`var` + access modifier)
- Functions
- Initializer `init` (may be `view`-annotated: `view init(...)`)
- Enum cases
- Nested type declarations: contracts may nest structs/resources/interfaces/events/
  enums/attachments; **interfaces may declare nested events** (and nested interfaces);
  other composites cannot nest composites

### 2.4 Functions
- Named functions (top-level & member)
- Generics: type parameters `<T: Bound>`; type-parameter inference
- `view` purity annotation; purity enforcement
- Argument labels vs. parameter names; `_` (no label)
- Default arguments (currently only the `ResourceDestroyed` default-destroy event)
- Function expressions / closures
- `pre`/`post` conditions
- Conditions: test conditions (optional string message), `emit` conditions;
  implicitly **`view`** (read-only)


### 2.5 Variables & constants
- `let` (constant), `var` (mutable)
- Transfer forms: `=` (copy), `<-` (move), `<-!` (force move)
- Dual-transfer (two-value) declarations (`let x <- a <- b`)

### 2.6 Imports
- Import all / import specific identifiers
- Import by address (`0x...`), by string, by identifier
- Aliased imports `import X as Y from ...`; alias uniqueness
- Cyclic-import detection
- Wildcard-address import prohibited

### 2.7 Transactions
- Parameters, importability requirement
- Fields, access-modifier restrictions
- `prepare` phase (params must be `&Account` references)
- `pre` conditions; `execute` phase; `post` conditions
- `execute`/`post` orderable
- `execute` required, `prepare` optional unless fields present

### 2.8 Entitlements
- `entitlement E`
- `entitlement mapping M { A -> B, include OtherMap }` (with cyclic/duplicate-inclusion checks)

### 2.9 Pragmas 
- `#pragma` directives (literal / identifier / invocation forms)
- `#removedType(T)`

---

## 3. Statements 

- `if` / `else if` / `else` (test may be expression or `let`/`var` optional binding)
- `guard ... else { ... }` (else block must exit; test may be expression or `let`/`var`
  optional binding)
- `while`
- `for [index,] x in ...` (optional index variable); iterates **arrays** (elements),
  **`InclusiveRange`** (members), **dictionaries** (keys), **`String`** (characters), and
  references to these
- `switch` / `case` / `default` (no implicit fall-through; default must be last & single)
- `return`, `break`, `continue`
- Assignment (`=`, `<-`, `<-!`)
- Swap `<->`
- `emit`
- `remove <Attachment> from <value>`

---

## 4. Expressions & Operators

### 4.1 Operators
- Logical `|| && !`
- Comparison `== != < > <= >=`
- Arithmetic `+ - * / %` (and unary `-`)
- Bitwise `| ^ & << >>`
- Nil-coalescing `??`
- Force-unwrap (postfix `!`)
- Move (prefix `<-`)
- Dereference (prefix `*`) on references

### 4.2 Expression kinds
- Identifier
- member access `.x`
- Optional chaining `?.x`
- Index access `[...]`:
  - **value indexing (read)**: arrays, dictionaries (returns `V?`), `String` (returns
    `Character`), `InclusiveRange`, references (delegate)
  - **index assignment (write `v[i] = x`)**: arrays & dictionaries only (`String` is read-only)
  - **type indexing `v[A]`** (attachment access): composites, references, intersection types
- Invocation, incl. explicit type arguments `f<T>(...)`
- Conditional / ternary `c ? a : b`
- Casting: `as` (static), `as?` (failable), `as!` (force)
- Reference creation `&v as &T`
- `create` / `destroy`
- `attach A() to v`
- Function literals / closures

---

## 5. Type Syntax & Annotations

- Nominal / qualified `A.B.C`
- Optional `T?`, nested `T??`
- Variable-sized array `[T]`, constant-sized `[T; N]`
- Dictionary `{K: V}`
- Function types `fun(...): R`; `view`
- Reference `&T`; authorized `auth(E) &T`, `auth(E, F)` (conjunction),
  `auth(E | F)` (disjunction), `auth(mapping M) &T`
- Intersection `{I1, I2}`
- Generic instantiation `T<A, B>`
- Resource annotation `@T`
- Parenthesized types `(T)`
- Type-parameter lists `<T: Bound>`

---

## 6. Built-in Types & Their Members

Comparable (support `< <= > >=`): all numeric types, `String`, `Character`, and `Bool`.
Equatable (support `==` / `!=`): the comparable types above, plus `Address`, path types,
`Type`, **enums**, and containers whose elements are equatable. General structs/resources
are **not** equatable (only enums among composites are).

### 6.1 Special / top / bottom types
- `Bool`
- `Void`
- `Never`
- `AnyStruct`, `AnyResource`
- `AnyStructAttachment`, `AnyResourceAttachment`
- `HashableStruct`
- `Storable`
- `StructStringer` — **built-in interface** with `toString()`

### 6.2 Numeric types
- Arbitrary precision: `Int`, `UInt`
- Signed sized: `Int8 … Int256`
- Unsigned sized (overflow-checked): `UInt8 … UInt256`
- Machine words (wrapping, no overflow check): `Word8 … Word256`
- Fixed-point signed: `Fix64`, `Fix128`; unsigned: `UFix64`, `UFix128`
- Supertypes: `Number`, `SignedNumber`, `Integer`, `SignedInteger`,
  `FixedSizeUnsignedInteger`, `FixedPoint`, `SignedFixedPoint`
- Instance members: `toString`, `toBigEndianBytes`;
  saturating ops `saturatingAdd/Subtract/Multiply/Divide` (where supported)
- Fixed-point only: `pow` (UFix64/UFix128), `multiplyDivide` (with rounding) (Fix/UFix)
- Static members: `fromString`, `fromBigEndianBytes`; static fields `min`, `max` (sized types)
- Conversion functions: every numeric type is callable as a converter —
  `Int(x)`, `UInt8(x)`, `Word64(x)`, `Fix64(x)`, `UFix64(x)`, …
- Arithmetic semantics: overflow/underflow **errors** on Int/UInt/fixed-point;
  **wrapping** on Word types; bitwise ops on integer types

### 6.3 Text
- `Character` — field `utf8`; method `toString`
- `String` — fields `utf8`, `length`; methods `concat`, `slice`, `decodeHex`, `split`,
  `replaceAll`, `contains`, `index`, `count`, `toLower`; statics `encodeHex`, `fromUTF8`,
  `fromCharacters`, `join`
  (UTF-8 / NFC normalization; grapheme-cluster indexing)
- `String` is indexable: `str[i]` returns `Character` (read-only; index is `Integer`)
- `StringBuilder` — type members `append`, `appendCharacter`, `clear`, `toString`,
  field `length`

### 6.4 Address & paths
- `Address` — `toBytes`, `toString`; statics `fromBytes`, `fromString`
- Path types: `Path`, `StoragePath`, `CapabilityPath`, `PublicPath`, `PrivatePath`
- Path domains: `storage`, `private`, `public`
- **Path constructor functions** (string → path, returns optional): `StoragePath(identifier:)`,
  `PublicPath(identifier:)`, `PrivatePath(identifier:)`

### 6.5 Container & structural types
- Optional `T?` — `map`
- Variable-sized array `[T]`: `length`; `contains`, `firstIndex`, `reverse`, `filter`,
  `map`, `concat`, `slice`, `toConstantSized`; mutating `append`, `appendAll`, `insert`,
  `remove`, `removeFirst`, `removeLast`
- Constant-sized array `[T; N]`: read-only members + `toVariableSized`
- Dictionary `{K: V}`: `length`, `keys`, `values`, `containsKey`, `forEachKey`;
  mutating `insert`, `remove`
- Mutation gated by container entitlements `Insert`, `Remove`, `Mutate`
- Reference array overloads: `filter`, `map` (resource-element arrays exclude most methods)

### 6.6 Reference & capability types
- Reference `&T` with authorization (unauthorized / entitlement-set / entitlement-map)
- `Capability<&T>` — fields `address`, `id`; methods `borrow`, `check`
- `StorageCapabilityController` — fields `capability`, `tag`, `borrowType`, `capabilityID`;
  methods `setTag`, `delete`, `target`, `retarget`
- `AccountCapabilityController` — fields `capability`, `tag`, `borrowType`, `capabilityID`;
  methods `setTag`, `delete`

### 6.7 Reflection / meta & run-time types
- `Type` (metatype value) — fields `identifier`, `isRecovered`, `address`, `contractName`;
  method `isSubtype(of:)`
- Universal members on all values: `isInstance(Type)`, `getType()`
- Composite/struct/resource extra: `forEachAttachment`
- `Type<T>()` — run-time type for a statically-known type
- Run-time type constructor functions (build `Type` values dynamically):
  `OptionalType`, `VariableSizedArrayType`, `ConstantSizedArrayType`, `DictionaryType`,
  `CompositeType` (by identifier), `FunctionType`, `ReferenceType`, `IntersectionType`,
  `CapabilityType`, `InclusiveRangeType`

### 6.8 Environment types
- `Block` — `height`, `view`, `timestamp`, `id` (`[UInt8; 32]`)
- `Account` and sub-objects
- `DeployedContract` — `address`, `name`, `code`; method `publicTypes()`
- `DeploymentResult` — `deployedContract: DeployedContract?`
- `PublicKey` — fields `publicKey`, `signatureAlgorithm`; methods `verify`, `verifyPoP`
- `AccountKey` — fields `keyIndex`, `publicKey`, `hashAlgorithm`, `weight`, `isRevoked`
- `InclusiveRange<T>` — `start`, `end`, `step`; `contains`
- `RoundingRule` enum (verified cases): `towardZero`, `awayFromZero`, `nearestHalfAway`,
  `nearestHalfEven`

### 6.9 Implicitly-available members & special identifiers
- **`self`** — receiver in composites/attachments; transaction value; contract value
- **Resource built-in fields** (injected on every resource): `owner: &Account?`,
  `uuid: UInt64`
- **Contract built-in field**: `self.account: &Account` (the account the contract is deployed to)
- **Attachment `base`** — reference to the attached-to value inside attachment members;
  `self` referring to the attachment
- **`result`** variable in post-conditions; **`before(expr)`** in post-conditions
- **Enum implicit members** — every enum exposes a `rawValue` field, and an implicit
  lookup constructor `MyEnum(rawValue:): MyEnum?`

### 6.10 Type capability classifications (documentable constraints)
- **Storable** (can be saved to account storage): most value types and **capabilities**;
  **references and functions are NOT storable**
- **Importable** (valid transaction/script parameter types): excludes functions &
  references; enforced on transaction/script parameters
- **Exportable** (can cross the runtime boundary as a return/result value)
- **Hashable** (valid dictionary key types): numbers, `Address`, `Bool`, `Character`,
  `String`, `Type`, paths, enums, `HashableStruct`
- **Equatable** / **Comparable**

---

## 7. Access Control & Entitlements

### 7.1 Access modifiers
- Primitive: `access(all)`, `access(self)`, `access(contract)`, `access(account)`
- Entitlement-based: `access(E)`, conjunction `access(E, F)`, disjunction `access(E | F)`
- Mapped: `access(mapping M)`
- Legacy: `pub`, `priv`, `pub(set)` — **deprecated**
- Access check modes (config): Strict / NotSpecifiedRestricted /
  NotSpecifiedUnrestricted / None

### 7.2 Entitlements & mappings
- User-declared entitlements and entitlement mappings (domain/image, set algebra)
- **Built-in entitlement mappings**: `Identity` (maps every entitlement to itself),
  `Account`, `Capabilities` 
- **Container entitlements** (built-in): `Mutate`, `Insert`, `Remove`
- **Account entitlements** (built-in, gate `&Account` member access):
  `Storage`, `Contracts`, `Keys`, `Inbox`, `Capabilities`,
  `SaveValue`, `LoadValue`, `CopyValue`, `BorrowValue`,
  `AddContract`, `UpdateContract`, `RemoveContract`,
  `AddKey`, `RevokeKey`,
  `PublishCapability`, `UnpublishCapability`,
  `GetStorageCapabilityController`, `IssueStorageCapabilityController`,
  `GetAccountCapabilityController`, `IssueAccountCapabilityController`,
  `PublishInboxCapability`, `UnpublishInboxCapability`, `ClaimInboxCapability`

---

## 8. Type-System Semantic Rules (documentable behaviors)

- **Type checking & inference** — type inference (literals, arrays, dictionaries via least
  common supertype), subtyping & type hierarchy, conversions/casting (`as`/`as?`/`as!`,
  impossible-cast detection), generic unification & type bounds, type-argument
  inference.
- **Resources & move semantics** — linear typing, use-after-invalidation, resource loss,
  mandatory create/move/destroy, nested-resource move, resource fields, capturing
  restrictions, resource-in-array/dictionary/optional rules, resource in
  ternary/for-loop restrictions. 
- **References** — authorized references, entitlement enforcement on member access,
  reference-to-optional errors, nested-reference prohibition, reference invalidation,
  dereference rules.
- **Access & entitlements** — access enforcement, entitlement set/map validity, mapping
  inclusion/cyclic checks, attachment entitlements, mapped-access authorization
  computation.
- **Interfaces & conformance** — conformance checking, missing/mismatched members,
  duplicate/cyclic conformance, default-implementation rules, member conflicts.
- **Composites & construction** — initializer requirements, field initialization
  (definite-assignment), reinitialization/uninitialized-access, composite-kind
  mismatch, construction/destruction validity, nesting restrictions.
- **Enums** — raw-type rules, case validity, conformance restrictions.
- **Attachments** — attach/remove validity, base-type rules, attachment annotations,
  attachment conformances.
- **Events** — event parameter type validity, emit rules (same location, events only,
  no imported events), emit-in-condition, default-destroy event rules.
- **Functions & calls** — argument count/labels, parameter compatibility, return
  checking, function-exit enforcement, purity.
- **Conditions & results** — pre/post conditions must be `Bool`, message must be `String`,
  `before(expr)` in post-conditions, `result` variable in post-conditions, emit conditions.
- **Control flow & reachability** — return value/statement requirements, unreachable
  statements, switch default position & non-empty cases, guard-else-must-exit, break/
  continue targeting.
- **Operators** — operand type rules (arithmetic equal types, comparable, Bool logic),
  nil-coalescing operand rules, swap-target validity, indexability/index-assignability,
  equatability, callability, optional-chaining restrictions.
- **Literals & values** — integer/fixed-point literal range & scale, address literal,
  character literal, constant-sized array literal size, dictionary key hashability,
  string-template interpolation restrictions (toString-able types only).
- **Imports/exports** — unresolved/cyclic imports, not-exported, duplicate import/alias,
  importability of transaction/script parameters.
- **Paths** — path domain & identifier validity.
- **Entry points** — scripts require a `main`-style entry function; 
  transaction parameter extraction; entry-point type validity.
- **Contract-update compatibility** — field types may not change; nested decls / interface
  conformances / enum cases may not be removed or reordered; declaration kind fixed;
  `#removedType`.

### 8.1 Behavioral & evaluation semantics (must-document behaviors)

**Evaluation order & short-circuiting**
- `&&` and `||` short-circuit (right side evaluated only when needed)
- `??` (nil-coalescing) evaluates its right side only when the left is `nil`
- Ternary `c ? a : b` evaluates only the taken branch
- Optional chaining `a?.b` stops at the first `nil` (skips the member/call, yields `nil`)
- Function arguments, and array/dictionary literal elements, evaluate left-to-right
- Arguments are evaluated at the call site, then transferred/converted to parameter types

**Optionals**
- `a?.b` (member) and `a?.b()` (call) wrap the result in an optional; calls aren't invoked if receiver is `nil`
- Force-unwrap `!` aborts at runtime if the value is `nil`
- `as?` returns `nil` on failure (no abort); `as!` aborts at runtime on mismatch; `as` is static

**Type inference**
- Array/dictionary literals infer a common supertype for elements / keys / values
- Empty literals adopt the contextually-expected type
- Generic type arguments are inferred from arguments (or given explicitly)
- Reference-typed parameters are never inferred — they need explicit annotation

**Casting**
- Failable cast `as?` of a resource requires optional binding and an identifier operand
- Casting a resource (`as`/`as!`) moves/invalidates the source
- Resource↔non-resource casts are statically rejected as always-failing
- Intersection→composite / intersection→intersection casts resolve via dynamic conformance

**References & authorization**
- References require an explicit target type; **no references-to-references**
- `&T?` parses as `(&T)?`; reference optionality must match the referenced value's
- Member/index access through a reference returns a **reference** to the element, with
  authorization **intersected/narrowed** (entitlement-set) or **mapped** (entitlement-map)
- Dereference `*r` reads through a reference; on an optional reference yields an optional;
  the referenced inner type must be a **primitive or a container of primitives**
- **Resource methods cannot be bound** to a variable — only invoked directly

**Resources**
- `create` only inside the declaring contract/location; only for direct resource types
- `destroy` only on resources; emits the resource's default `ResourceDestroyed` event
- Move `<-` is mandatory for resource expressions; it invalidates the source
- Double-transfer `let x <- a <- b` temporarily invalidates the first target, then re-validates
- Attaching to a resource moves it; resources forbidden in ternary expressions

**Control flow**
- `switch`: test & cases must be equatable; **no implicit fall-through**; each case needs
  ≥1 statement; a single `default` must be last
- `break` targets the innermost loop **or** `switch`; `continue` targets the enclosing
  **loop** (skipping any intervening `switch`)
- Functions must return on all paths unless the return type is `Void`
- Unreachable statements (after a definite return/halt) are flagged
- `guard`'s `else` must exit; only `if`/`guard` support optional binding

**Declarations & members**
- Function parameters are constants; argument labels must be unique
- Constant (`let`) fields are assignable once, in `init`; all fields must be initialized before use
- Enum cases are always public; enums carry an implicit `rawValue`
- Imported contract values are exposed as **references**
- Assignment to an optional-chain result (`a?.b = x`) is not allowed
- Index assignment through a reference requires `Mutate` (or `Insert`/`Remove`) entitlements

**Conditions**
- Pre/post condition blocks run in a `view` (read-only) context
- Post-conditions can use `before(expr)` and the `result` variable

**Subtyping**
- `Never` is a subtype of every type; every type is a subtype of `Any` / `AnyStruct` /
  `AnyResource` (by kind)
- Optionals are covariant: `T? <: U?` when `T <: U`
- References are covariant in the referenced type; a more-authorized reference is a subtype
  of a less-authorized one (`auth(E) &T <: &T`)
- Functions: parameters are contravariant, return types covariant; a `view` function is a
  subtype of an impure function
- Intersection-type subtyping (`{Us} <: {Vs}`, and `AnyStruct{Vs}` etc. relations)

---

## 9. Account Model & Storage

### 9.1 `Account`
- Fields: `address`, `balance`, `availableBalance`, `storage`, `contracts`, `keys`,
  `inbox`, `capabilities`
- Account creation: `Account(payer: auth(BorrowValue | Storage) &Account)` → fully-entitled `&Account`
- Authorized vs. unauthorized account references (entitlement-gated)

### 9.2 `Account.Storage`
- `save`, `load`, `copy`, `borrow`, `check`, `type`
- fields `used`, `capacity`, `publicPaths`, `storagePaths`
- iteration `forEachStored`, `forEachPublic`
- domains `/storage`, `/public`, `/private` (`/private` legacy); storage domains

### 9.3 `Account.Contracts`
- `add`, `update`, `tryUpdate` (→ `DeploymentResult`), `get`, `borrow`, `remove`; field `names`

### 9.4 `Account.Keys`
- `add`, `get`, `revoke`, `forEach`; field `count`

### 9.5 `Account.Inbox`
- `publish`, `unpublish`, `claim`

### 9.6 `Account.Capabilities` (capability-controller model)
- top: `get`, `borrow`, `exists`, `publish`, `unpublish`
- `Account.Capabilities.Storage`: `issue`, `issueWithType`, `getController`,
  `getControllers`, `forEachController`
- `Account.Capabilities.Account`: `issue`, `issueWithType`, `getController`,
  `getControllers`, `forEachController`

---

## 10. Standard Library & Built-in Functions

### 10.1 Global functions
- `assert`, `panic`, `log`
- `revertibleRandom<T: FixedSizeUnsignedInteger>(modulo?)`
- `getAccount`, `getAuthAccount` (**script-only**)
- `getBlock`, `getCurrentBlock`
- `InclusiveRange<T>(start, end, step?)`
- `Comparison` contract: `min`, `max`, `clamp`
- enum constructors from raw value: `HashAlgorithm`, `SignatureAlgorithm`, `RoundingRule`
- numeric conversion functions (`Int(x)`, `UInt8(x)`, `Fix64(x)`, …)
- path constructor functions (`StoragePath(identifier:)`, …)
- `StringBuilder()` constructor
- run-time type constructor functions (`Type<T>()`, `OptionalType`, `ReferenceType`, …)

### 10.2 Cryptography
- `HashAlgorithm`: `SHA2_256`, `SHA2_384`, `SHA3_256`, `SHA3_384`, `KECCAK_256`, `KMAC128_BLS_BLS12_381`;
  methods `hash`, `hashWithTag`
- `SignatureAlgorithm` enum: `ECDSA_P256`, `ECDSA_secp256k1`, `BLS_BLS12_381`
- `PublicKey` — fields `publicKey`, `signatureAlgorithm`; methods `verify`, `verifyPoP`
- `BLS` contract: `aggregateSignatures`, `aggregatePublicKeys` (defined here: `stdlib/bls.cdc`)
- `Crypto` contract — referenced by location only in this repo (source/members defined
  elsewhere)

### 10.3 Encoding
- RLP contract: `decodeString`, `decodeList`
- Hex encode/decode on `String` / `[UInt8]`

### 10.4 Flow core events
- Account lifecycle: `AccountCreated`, `AccountKeyAdded`, `AccountKeyRemoved`,
  `AccountContractAdded`, `AccountContractUpdated`, `AccountContractRemoved`
- Capability controllers: `StorageCapabilityControllerIssued`,
  `AccountCapabilityControllerIssued`, `StorageCapabilityControllerDeleted`,
  `AccountCapabilityControllerDeleted`, `StorageCapabilityControllerTargetChanged`
- Capability publication: `CapabilityPublished`, `CapabilityUnpublished`
- Inbox: `InboxValuePublished`, `InboxValueUnpublished`, `InboxValueClaimed`

### 10.5 Resource destruction events
- Default destruction event (`ResourceDestroyed`) emitted on resource destroy

---

## 11. Testing Framework

- `Test` contract: `assert`, `assertEqual`, `fail`, `expect`, `expectFailure`,
  `newMatcher`, `readFile`, `not`
- Built-in matchers (verified): `equal`, `beEmpty`, `haveElementCount`, `contain`,
  `beGreaterThan`, `beLessThan`, `beNil`, `beSucceeded`, `beFailed` (last three defined in
  `stdlib/contracts/test.cdc`)
- `Matcher` type — field/fn `test`; **combinators** `and`, `or` (and `Test.not`);
  custom matchers via `Test.newMatcher`
- Test value/result types:
  - `Result` interface (field `status`); `ResultStatus` enum (`succeeded`/`failed`)
  - `ScriptResult` — `status`, `returnValue`, `error`
  - `TransactionResult` — `status`, `error`, `computationUsed`
  - `Error` — `message`
  - `TestAccount` — `address`, `publicKey`
  - `Transaction` — `code`, `authorizers`, `signers`, `arguments`
  - `Matcher`, `BlockchainBackend` (interface)
- Blockchain API (authoritative source `stdlib/contracts/test.cdc`, 516 lines):
  `executeScript`, `createAccount`, `getAccount`, `addTransaction`,
  `executeNextTransaction`, `commitBlock`, `executeTransaction`, `executeTransactions`,
  `deployContract`, `logs`, `serviceAccount`, `events`, `eventsOfType`, `reset`,
  `moveTime`, `freezeTime`, `unfreezeTime`, `createSnapshot`, `loadSnapshot`
- Result/value types: `ScriptResult`, `TransactionResult`, `TestAccount`, `Transaction`, `Error`
- Coverage reporting integration

---

## 12. Runtime Semantics

### 12.1 Execution model
- Script execution: single entry function, exportable return value
- Argument decoding / import & static-type validation
- Value & type import/export across host boundary
- Program recovery: certain contracts that fails to check can be recovered
  (`RecoverProgram`, parsed via the old parser); recovered programs are flagged and
  surfaced through `Type.isRecovered`

### 12.2 Value semantics
- Copy for non-resources; move for resources; transfer between containers/storage
- Reference invalidation on move/destroy
- Resource destruction (nested teardown, `destroy` body, default-destroy events,
  attachment `base` reference)
- Iteration (arrays, dictionaries `forEachKey`, storage `forEachStored/Public`, ranges)

### 12.3 Limits & metering
- Computation metering (statements, loops, invocations, value create/transfer/destroy, 
  string ops, parsing, atree ops, etc.)
- Memory metering / limit
- Call-stack depth limit (default 2000)
- Coverage reporting (JSON/LCOV) 
- Computation profiling / pprof export

---

## 13. Encoding: value/type interchange formats

- JSON-Cadence
- CCF (Cadence Compact Format)

