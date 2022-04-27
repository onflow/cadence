# v0.19.1 (2021-09-13)

## 🛠 Improvements

- Remove obsolete script/transaction parameter check (#1114) @SupunS 

## 🐞 Bug Fixes

- Fix export errors by properly wrapping in runtime errors (#1139) @SupunS

# v0.19.0 (2021-09-13)

## 💥 Breaking Changes

- Fix export of values (#1067) @turbolent
- Add encoding/decoding for array value static type (#1035) @SupunS

## ⭐ Features

- Add public account contracts (#1090) @SupunS
- Add `names` field to auth account contracts (#1089) @SupunS
- Report a hint when casting to a same type as the expected type (#1056) @SupunS
- Add encoding/decoding for array value static type (#1035) @SupunS
- Add walker for values (#1037) @turbolent
- Add encoding/decoding dictionary static type info (#1036) @SupunS
- Store static type in array/dictionary values (#1041) @SupunS
- Infer static-type from imported array/dictionary values (#1052) @SupunS

## 🛠 Improvements

- Add type inferring for func-args when there are no generics (#1033) @SupunS
- Add container static types (#1125) @SupunS
- Add container mutation check for arrays and dictionaries (#1103) @SupunS
- flow-go sync tweaks (#1104) @janezpodhostnik
- Return a dedicated error when decoding fails due to an unsupported CBOR tag (#1064) @turbolent
- Remove obsolete fields (#1053) @turbolent
- Include and consider static types of arrays and dictionaries (#1043) @turbolent
- Infer array and dictionary static types from expected type during import (#1038) @turbolent
- Finish storage format changes (#1042) @turbolent
- Add static types to array values and dictionary values (#1034) @turbolent
- FlowKit API Update (#1028) @sideninja
- Produce valid Cadence string literals when formatting strings (#1023) @turbolent
- Directly declare base values in activation without interpreter (#1022) @turbolent
- Reuse base activation across all interpreters (#1018) @SupunS
- Extend the source compatibility suite (#980) @turbolent
- Update the language server to the latest Cadence version (#1015) @turbolent
- Convert the maps in the type file to simple arrays (#1086) @jwinkler2083233
- Update Security to point to website (#1074) @jkan2
- Gracefully handle the optional-chaining invocation on non-optional member (#1071) @turbolent
- Add static type sanity check for imported values (#1048) @SupunS
- Add tests for decoding from old format and encoding in new format (#1046) @SupunS
- Add more tests for imported array/dictionary value type conformance (#1045) @SupunS
- Add encoder/decoder v5 (#1039) @SupunS
- Remove additional check for builtin values during import resolving (#1026) @SupunS
- Update NPM packages (#1011) @turbolent

## 🐞 Bug Fixes

- Cointainer variances fixes (#1087) @turbolent
- Check member reads and writes (#1085) @turbolent
- Include receiver in function type and check it in invocations  (#1084) @turbolent
- Fix race condition in hash and sign algorithm values (#1096) @SupunS
- Check run-time type of the argument of getCapability calls (#1083) @turbolent
- Fix interpretation of optional binding with second value (#1082) @turbolent
- Check resource construction (#1081) @turbolent
- Fix address conversion (#1080) @turbolent
- Gracefully handle errors during transaction and script argument validation (#1070) @turbolent
- Fix map keys (#1069) @turbolent
- Fix import of values (#1068) @turbolent
- Fix export of values (#1067) @turbolent
- Fix get and revoke key functions (#1065) @turbolent
- Finish storage format changes (#1042) @turbolent
- Fix close-brace for struct definition docs in docgen tool (#1019) @SupunS
- Don't check missing program (#1013) @turbolent

## 📖 Documentation

- Give Resources their own page. (#1130) @10thfloor
- Fix and improve documentation (#1122) @turbolent
- Add presentation "Programming Language Implementation / Cadence Implementation" (#990) @turbolent
- Remove Crypto contract status callouts (#1012) @turbolent
- Improve the documentation for dictionaries and arrays (#1020) @turbolent
- Fix operator documentation (#1017) @turbolent
- Updating broken links in ReadMe (#1097) @kimcodeashian

# v0.18.0 (2021-06-15)

## 💥 Breaking Changes

- Add HashAlgorithm hash and hashWithTag functions (#1002) @turbolent
- Update the Crypto contract (#999) @turbolent

## ⭐ Features

- Add HashAlgorithm hash and hashWithTag functions (#1002) @turbolent
- Add npm package for the docgen-tool (#1008) @SupunS
- Add wasm generation for docgen tool (#1005) @SupunS

## 🛠 Improvements

- Make `PublicKey` type importable (#995) @SupunS
- Update the Crypto contract (#999) @turbolent
- Add dynamic type importability check for arguments (#1007) @SupunS
- Language Server NPM package: Add support for Node environment, add tests (#1006) @turbolent
- Embed markdown templates to docgen tool at compile time (#1004) @SupunS
- No need for checking resource loss when function definitely halted (#1000) @turbolent
- Remove the result declaration kind (#1001) @turbolent
- Update for changes in Flowkit API (#962) @sideninja
- Add Supun as code owner (#997) @turbolent
- Disable the wasmtime VM for now (#991) @turbolent

## 🐞 Bug Fixes

- Add dynamic type importability check for arguments (#1007) @SupunS
- Fix dictionary deferred owner (#992) @turbolent
- Fix AST walk for transaction declaration (#998) @turbolent

## 📖 Documentation

- Add documentation for the Cadence documentation generator (#1003) @SupunS


# v0.17.0 (2021-06-08)

## ⭐ Features

- Add deferred decoding support for dictionary values (#925) @SupunS
- Track origin's occurrences (#907) @turbolent
- Declare more ranges (#882) @turbolent
- Language Server: Suggest completion items for declaration ranges (#881) @turbolent
- Provide code action to declare field and function (#961) @turbolent
- Provide code action to declare variable/constant (#946) @turbolent
- Add code action to implement missing members (#942) @turbolent
- Provide code actions / quick fixes (#941) @turbolent
- Provide documentation for range completion items (#923) @turbolent
- Record docstrings in variables  (#922) @turbolent
- Add a `String.utf8` field (#954) @turbolent
- Provide signature help (#911) @turbolent
- Add a String constructor function and a String.encodeHex function (#953) @turbolent
- Record position information for function invocations (#910) @turbolent
- Add function docs formatting support (#938) @SupunS
- Add Cadence documentation generator (#927) @SupunS
- Rename all occurrences of a symbol in the document (#909) @turbolent
- Enable DocumentSymbol capability for Outline (#662) @MaxStalker
- Add an AST walking function (#939) @turbolent
- Add HexToAddress utility function, add tests for address functions (#932) @turbolent
- Highlight all occurrences of a symbol in the document (#908) @turbolent

## 🛠 Improvements

- Add deferred decoding support for dictionary values (#925) @SupunS
- Improve type inference for binary expressions (#957) @turbolent
- Update language server to Cadence v0.16.0 (#926) @turbolent
- [Doc-Gen Tool] Add documentation generation support for event declarations (#985) @SupunS
- [Doc-Gen Tool] Group declarations based on their kind (#984) @SupunS
- Check 'importability' instead of 'storability' for transaction arguments (#983) @SupunS
- Allow native composite types to be passed as script arguments (#973) @SupunS
- Validate UTF-8 compatibility in string value constructor (#972) @SupunS
- Cache type ID for composite types (#950) @SupunS
- [Optimization] Make dynamic types singleton for simple types (#963) @SupunS
- Include docstrings in value and type variables, improve hover markup (#945) @turbolent
- Always check arguments and record the argument types in the elaboration (#951) @turbolent
- Optimize qualified identifier creation (#948) @SupunS
- Optimize `Type` function declaration (#949) @turbolent
- [Optimization] Re-use converter functions across interpreters (#947) @SupunS
- Update and extend the source compatibility suite (#977) @turbolent
- Update to Go 1.16.3's libexec/misc/wasm/wasm_exec.js (#929) @turbolent

## 🐞 Bug Fixes

- Track origin's occurrences (#907) @turbolent
- Bring back fmt.Stringer implementation for interpreter.Value (#969) @turbolent
- Fix the initialization order in the interpreter (#958) @turbolent
- Fix parsing of negative fractional fixed-point strings (#935) @mikeylemmon
- capture computation used for all transactions (#895) @ramtinms
- Only wrap a type in an optional if it is not nil (#937) @turbolent

## 🧪 Tests

- Add test case for invalid utf8 string import (#968) @SupunS
- Test contract member initialization (#931) @turbolent

## 📖 Documentation

- Update account docs to not to pass PublicKey as an argument (#967) @SupunS
- Improve the crypto doc (#960) @tarakby

## 🏗 Chore

- Use actions/setup-go@v2 to speed up CI by ~10 secs and specify Go 1.15.x (#971) @fxamacker
- Remove italics from auto flow-go PR (#988) @janezpodhostnik
- Improve sync flow_go action (#987) @janezpodhostnik
- Fix for auto-update flow-go: no fail on tidy (#986) @janezpodhostnik
- Auto update flow-go github action (#981) @janezpodhostnik
- Remove coverage of empty functions from report (#944) @turbolent
- Improve coverage calculation (#933) @turbolent

# v0.16.1 (2021-05-23)

## 🐞 Bug Fixes

- Only wrap a type in an optional if it is not nil (#936) @SupunS 

# v0.16.0 (2021-04-21)

## 💥 Breaking Changes

- Add new crypto features (#852) @SupunS

  Renamed the `ECDSA_Secp256k1` signature algorithm to `ECDSA_secp256k1`.

  Please update existing contracts, transactions, and scripts to use the new name.

- Remove the high-level storage API (#877) @turbolent

  The internal interface's `SetCadenceValue` function was removed.

  This change does not affect Cadence programs (contracts, transactions, scripts).

## ⭐ Features

- Add non-local type inference for expressions (#875) @SupunS

  The type of most declarations and expressions is now inferred in a uni-directional way, instead of only locally.

  For example, this allows the following declarations to type-check without having to add static casts:

  ```kotlin
  let numbers: [Int8] = [1, 2, 3]
  let nestedNumbers: [[Int8]] = [[1, 2], [3, 4]]
  let numberNames: {Int64: String} = {0: "zero", 1: "one"}
  let sum: Int8 = 1 + 2
  ```

- Add new crypto features (#852) @SupunS

  **Hash Algorithms**

  - Added [KMAC128_BLS_BLS12_381 hashing algorithm](https://docs.onflow.org/cadence/language/crypto/#hashing-algorithms)): `HashAlgorithm.KMAC128_BLS_BLS12_381`

  **Signature Algorithms**

  - Added [BLS_BLS12_381 signature algorithm](https://docs.onflow.org/cadence/language/crypto/#signature-algorithms): `SignatureAlgorithm.BLS_BLS12_381`

  **`PublicKey` Type**

  - Added the field `let isValid: Bool`. For example:

    ```kotlin
    let publicKey = PublicKey(publicKey: [], signatureAlgorithm: SignatureAlgorithm.ECDSA_P256)

    let valid = publicKey.isValid
    ```

  - Added the function `verify`, which allows validating signatures:

    ```kotlin
    pub fun verify(
        signature: [UInt8],
        signedData: [UInt8],
        domainSeparationTag: String,
        hashAlgorithm: HashAlgorithm
    ): Bool
    ```

    For example:

    ```kotlin
    let valid = publicKey.verify(
        signature: [],
        signedData: [],
        domainSeparationTag: "something",
        hashAlgorithm: HashAlgorithm.SHA2_256
    )
    ```

  - Added the function `hashWithTag`, which allows hashing with a tag:

    ```kotlin
    pub fun hashWithTag(
        _ data: [UInt8],
        tag: string,
        algorithm: HashAlgorithm
    ): [UInt8]
    ```

- Direct contract function invoke (#878) @janezpodhostnik

  Cadence now supports an additional function to allow the host environment to call contract functions directly.

- Language Server: Add support for access check mode (#868) @turbolent
- Record variable declaration ranges (#880) @turbolent

## ⚡️ Performance

The performance of decoding and encoding values was significantly improved by @fxamacker and @SupunS:

- Optimize encoding by using CBOR lib's StreamEncoder (#830) @fxamacker
- Optimize decoding by using CBOR StreamDecoder (#885) @fxamacker
- Add deferred decoding for array values (#871) @SupunS
- Add deferred decoding support for composite values (#896) @SupunS
- Pre-allocate and reuse value-path for encoding (#858) @fxamacker
- Pre-allocate and reuse value-path for decoding (#869) @turbolent
- Optimize Address.Bytes() to inline fast path (#848) @fxamacker
- Remove call to Valid() during encoding to boost speed by about 13-18% (#857) @fxamacker
- Directly create int subtype value at interpreter (#913) @SupunS

## 🛠 Improvements

- Refactor Language Server to use CLI shared library (#751) @MaxStalker
- Make PublicKey value immutable (#879) @SupunS
- Language Server: update flow-cli (#894) @psiemens
- Update language server to latest Cadence and Go SDK (#874) @psiemens
- Make the interpreter location optional (#918) @turbolent

## 🔒 Security

This release contains major security fixes:

- Declare post-condition result variable as reference if return type is a resource (#905) @turbolent

  We would like to thank Deniz Mert Edincik for finding and reporting this critical issue responsibly through our [Responsible Disclosure Policy](https://github.com/onflow/cadence/blob/master/SECURITY.md).

  See https://forum.onflow.org/t/fixed-cadence-vulnerability-2021-04-13

- Fix dynamic subtype test for reference values (#914) @turbolent

  We would like to thank Deniz Mert Edincik for finding and reporting this critical issue responsibly through our [Responsible Disclosure Policy](https://github.com/onflow/cadence/blob/master/SECURITY.md).

  We would also like to thank Mikey Lemmon for finding this issue independently and reporting it responsibly, too.

  See https://forum.onflow.org/t/fixed-cadence-vulnerability-2021-04-20

- Check storability when value is written (before it is encoded) (#915) @turbolent

  We would like to thank Deniz Mert Edincik for finding and reporting this medium issue responsibly through our [Responsible Disclosure Policy](https://github.com/onflow/cadence/blob/master/SECURITY.md).

  See https://forum.onflow.org/t/fixed-cadence-vulnerability-2021-04-20

- Check the index bounds for arrays (#917) @turbolent

  We would like to thank Mário Silva for finding and reporting this medium issue responsibly through our [Responsible Disclosure Policy](https://github.com/onflow/cadence/blob/master/SECURITY.md).



## 🐞 Bug Fixes

- Improve reference values' dynamic type, static type, copy, equal, and conformance functions (#921) @turbolent
- Implement `StaticType` for `StorageReferenceValue` and `EphemeralReferenceValue` (#920) @turbolent
- Fix strings (#919) @turbolent
- PublicKeyValue stringer fix (#903) @janezpodhostnik
- Handle recursive values (due to references) in string function (#906) @turbolent
- Handle case with no arguments (#886) @MaxStalker
- Fix link function return value (#865) @turbolent
- Fix optional type ID, add tests (#864) @turbolent
- Fix function type members  (#867) @turbolent
- Language Server:  Reuse existing checkers instead of re-parsing and re-checking imports  (#855) @turbolent
- Only consider the program invalid if there are error diagnostics (#854) @turbolent

## 🧪 Tests

- Add test for TopShot moment batch transfer (#826) @turbolent
- Test equal for nil value, in comparison and switch (#866) @turbolent
- Test storing capabilities (#856) @turbolent
- Test getCapability after unlink (#849) @turbolent
- Add interpreter tests for common resource reference usages (#916) @turbolent
- Add extra test for deterministic JSON export (#893) @m4ksio

## 📖 Documentation

- Add a reference to PublicKey in the crypto docs (#912) @SupunS
- Tutorial typo fixes (#825) @joshuahannan
- Update values-and-types.mdx (#827) @FeiyangTan

## 🏗 Chores

- Enable more linters and lint (#902) @turbolent
- Measure and report code coverage (#899) @turbolent
- Fix CI linter failure (#904) @SupunS
- Fix Codecov "file not found at ./coverage.txt" (#901) @fxamacker

# v0.15.1 (2021-04-21)

## 🛠 Improvements

- Include wrapped error in InvalidEntryPointArgumentError message (#823) @mikeylemmon
- Remove encoding and decoding support for storage references (#819) @turbolent

## 📖 Documentation

- Document saturation arithmetic and min/max fields (#817) @turbolent

# v0.15.0 (2021-04-20)

## ⭐ Features

- Added balance fields on accounts (#808) @janezpodhostnik
- Add address field to capabilities (#736) @ceelo777
- Add array appendAll function (#727) @lkadian
- Validate arguments passed into cadence programs (#724) @SupunS
- Implement equality for storable values and static types (#790) @turbolent
- Declare min/max fields for integer and fixed-point types (#803) @turbolent
- Add saturation arithmetic (#804) @turbolent
- Add functions to read stored and linked values (#812) @turbolent

## 🛠 Improvements

- Optimize storage format (#797) @fxamacker
- Paralleize encoding (#731) @zhangchiqing
- Deny removing contracts if contract update validation is enabled (#792) @SupunS
- Simplify function types (#802) @turbolent
- Cache capability type members (#799) @turbolent
- Use force expression's end position in force nil error (#789) @turbolent
- Handle cyclic references in dynamic type conformance check (#787) @SupunS
- Benchmark CBOR encoding/decoding using mainnet data (#783) @fxamacker
- Extend defensive run-time checking to all value transfers (#784) @turbolent
- Remove obsolete code in decode.go (#788) @fxamacker
- Optimize encoding of 12 types still using CBOR map (#778) @fxamacker
- Reject indirect incompatible updates to composite declarations (#772) @SupunS
- Panic with a dedicated error when a member access results in no value (#768) @turbolent
- Update language server to Cadence v0.14.4 (#767) @turbolent
- Improve value type conformance check (#776) @turbolent
- Add validation for enum cases during contract updates (#762) @SupunS
- Optimize encoding of composites (#761) @fxamacker
- Improve state decoding tool (#759) @turbolent
- Optimize encoding of dictionaries (#752) @fxamacker
- Prepare Decoder for storage format v4 (#746) @fxamacker
- Make numeric types singleton (#732) @SupunS
- Validate arguments passed into cadence programs (#724) @SupunS
- Use location ID as key for maps holding account codes (#723) @turbolent
- Make native composite types non-storable (#713) @turbolent
- Improve state decoding tool (#798) @turbolent
- Remove unused storage key handler (#811) @turbolent

## 🐞 Bug Fixes

- Extend defensive run-time checking to all value transfers (#784) @turbolent
- Fix borrowing (#782) @turbolent
- Prevent cyclic imports (#809) @turbolent
- Clean up "storage removal" index expression (#769) @turbolent
- Get resource owner from host environment (#770) @SupunS
- Handle field initialization using force-move assignment (#741) @turbolent
- Fix error range when reporting a type error for an indexing reference expression (#719) @turbolent

## 📖 Documentation

- Improve documentation for block timestamp (#775) @turbolent
- Document run-time type identifiers (#750) @turbolent
- Add documentation for the getType() builtin method  (#737) @SupunS


# v0.14.5 (2021-04-08)

## 🐞 Bug Fixes

- Fix borrowing (#785) @turbolent

# v0.14.4 (2021-03-25)

## ⭐ Features

- Add dictionary contains function (#716) @lkadian

## 🐞 Bug Fixes

- Fix runtime representation of native HashAlgorithm/SignAlgorithm enums (#725) @SupunS
- Fix error range when reporting a type error for an indexing reference expression (#719) @turbolent

## 🧪 Tests

- Benchmark real fungible token transfers (#722) @turbolent
- Use location ID as key for maps holding account codes (#723) @turbolent


# v0.13.10 (2021-03-25)

## 🐞 Bug Fixes

- Add support for access(account) and multiple contracts per account (#730) @turbolent

# v0.13.9 (2021-03-22)

## 🛠 Improvements

- Lazily load contract values (#720) @turbolent

# v0.14.3 (2021-03-22)

## 🛠 Improvements

- Lazily load contract values (#715) @turbolent
- Make native composite types non-storable (#713) @turbolent
- Ensure imported checkers/elaborations are reused in tests' import handlers (#711) @turbolent
- Use require.ErrorAs (#714) @turbolent

## 🐞 Bug Fixes

- Add support for access(account) and multiple contracts per account (#710) @turbolent

# v0.14.2 (2021-03-18)

## ⭐ Features

- Replace parser demo with AST explorer (#693) @turbolent
- Report a hint when a dynamic cast is statically known to always succeed (#688) @turbolent

## 🛠 Improvements

- Revert the addition of the unsupported signing and hash algorithms (#705) @turbolent

## 🐞 Bug Fixes

- Fix cadence-parser NPM package, update versions (#694) @turbolent
- Properly handle recursive types when checking validity of event parameter types (#709) @turbolent
- Make all path sub-types externally returnable (#707) @turbolent

## 📖 Documentation

- Fix the syntax of the signature algorithm enum in docs (#701) @SupunS


# v0.13.8 (2021-03-18)

## 🐞 Bug Fixes

- Properly handle recursive types when checking validity of event parameter types (#708) @turbolent

# v0.14.1 (2021-03-16)

## 🛠 Improvements

- Add unknown constants for signature algorithm and hash algorithm (#699) @turbolent
- Optimize checker activations (#674) @turbolent
- Return nil when revoking a non-existing key (#697) @SupunS

## 🐞 Bug Fixes

- Fix nested error pretty printing (#695) @turbolent

# v0.14.0 (2021-03-15)

This release introduced a new [high-level Account Key API](https://docs.onflow.org/cadence/language/accounts/) and improves the interpreter performance.
The current low-level Account Key API is now deprecated and will be removed in a future release. Please switch to the new one.

The following example transaction demonstrates the functionality:

```kotlin
transaction {
	prepare(signer: AuthAccount) {
		let newPublicKey = PublicKey(
			publicKey: "010203".decodeHex(),
			signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
		)

		// Add a key
		let addedKey = signer.keys.add(
			publicKey: newPublicKey,
			hashAlgorithm: HashAlgorithm.SHA3_256,
			weight: 100.0
		)


		// Retrieve a key
		let sameKey = signer.keys.get(keyIndex: addedKey.keyIndex)


		// Revoke a key
		signer.keys.revoke(keyIndex: addedKey.keyIndex)
	}
}
```

## ⭐ Features

- Introduce a new [high-level Account Key API](https://docs.onflow.org/cadence/language/accounts/) (#633) @SupunS
- Allow the Language Server to be debugged (#663) @turbolent

## 💥 Breaking Changes

The `Crypto` contract has changed in a backwards-incompatible way, so the types and values it declared could be used in the new Account Key API:

- The struct `Crypto.PublicKey` was replaced by the new built-in global struct `PublicKey`. 
  There is no need anymore to import the `Crypto` contract to work with public keys.

- The struct `Crypto.SignatureAlgorithm` was replaced with the new built-in global enum `SignatureAlgorithm`.
  There is no need anymore to import the `Crypto` contract to work with signature algorithms.

- The struct `Crypto.HashAlgorithm` was replaced with the new built-in global enum `HashAlgorithm`.
  There is no need anymore to import the `Crypto` contract to work with hash algorithms.

- The signature algorithm value `Crypto.ECDSA_Secp256k1` was replaced with the new built-in enum case `SignatureAlgorithm.ECDSA_Secp256k1`

- The signature algorithm value `Crypto.ECDSA_P256` was replaced with the new built-in enum case `SignatureAlgorithm.ECDSA_P256`

- The hash algorithm `Crypto.SHA3_256` was replaced with the new built-in enum case `HashAlgorithm.SHA3_256`

- The hash algorithm `Crypto.SHA2_256` was replaced with the new built-in enum case `HashAlgorithm.SHA2_256`

## 🛠 Improvements

- Add support for importing enum values (#672) @SupunS
- Optimize interpreter: Make invocation location ranges lazy (#685) @turbolent
- Optimize interpreter activations (#673) @turbolent
- Optimize integer conversion (#677) @turbolent
- Optimize interpreter: Evaluate statements and declarations directly, remove trampolines (#684) @turbolent
- Optimize interpreter: Move statement evaluation code  (#683) @turbolent
- Optimize interpreter: Move evaluation of expressions and statements to separate files (#682) @turbolent
- Optimize interpreter: Evaluate expressions directly without trampolines (#681) @turbolent
- Optimize interpreter: Refactor function invocations to return a value instead of a trampoline  (#680) @turbolent
- Optimize interpreter: Evaluate binary expressions directly (#679) @turbolent

## 🐞 Bug Fixes

- Add support for exporting enums (#669) @SupunS
- Ensure code is long enough when extracting excerpt (#690) @turbolent

## 📖 Documentation

- Add documentation for the new account key API (#635) @SupunS

# v0.13.7 (2021-03-12)

## 🐞 Bug Fixes

- Ensure code is long enough when extracting excerpt (#690) @turbolent

# v0.13.6 (2021-03-09)

## 🐞 Bug Fixes

- Revert "Cache qualified identifier and type ID for composite types and interface types" (#670) @turbolent

# v0.13.5 (2021-03-05)

## 🛠 Improvements

- Add computed fields to the composite value (#664) @SupunS
- Improve naming: Rename nominal type to simple type (#659) @turbolent
- Cache qualified identifier and type ID for composite types and interface types (#658) @turbolent

## 🐞 Bug Fixes

- Fix handling of capability values without borrow type (#666) @turbolent
- Improve type equality check in the contract update validation (#654) @SupunS

# v0.13.4 (2021-03-04)

## ⭐ Features

- Add parser for pragma signers (#656) @MaxStalker

## 🐞 Bug Fixes

- Fix updating contracts with reference typed fields (#649) @SupunS

## 📖 Documentation

- Move remaining Cadence docs to Cadence repo. (#653) @10thfloor

# v0.13.3 (2021-03-03)

## 🛠 Improvements

- Optimize checker construction (#630) @turbolent

# v0.13.2 (2021-03-03)

## 🛠 Improvements

- Make contract update validation optional and disable it by default (#646) @turbolent
- Delay contract code and value updates to the end of execution (#636) @turbolent
- Optimize encoding of positive bigints (#637) @turbolent
- Optimize deferred dictionary keys (#638) @turbolent
- Refactor `String`, `AuthAccount.Contracts`, and `DeployedContract` type to singleton (#625) @turbolent
- Refactor `AuthAccount`, `PublicAccount`, and `Block` type to singleton (#624) @turbolent
- Only record member accesses when origins and occurrences are enabled (#627) @turbolent
- Cache members for array and dictionary types (#626) @turbolent

## 🐞 Bug Fixes

- Support nested file imports in LSP (#616) @psiemens

## 📖 Documentation

- Fix code examples and improve documentation in language reference (#634) @jeroenlm

# v0.13.1 (2021-02-24)

## 🛠 Improvements

- Remove prepare and decode callbacks (#622) @turbolent
- Update language server to Cadence v0.13.0 and Go SDK v0.15.0 (#623) @turbolent
- Refactor Any type, AnyResource type, and AnyStruct type to singleton (#618) @turbolent
- Refactor Type, Bool, and Character type to singletons (#617) @turbolent

## 🐞 Bug Fixes

- Fix AuthAccount nested types (#619) @turbolent

# v0.13.0 (2021-02-22)

## ⭐ Features

- Validate contract updates (#593) @SupunS
- WebAssembly: Use reference types (#448) @turbolent
- Debug log decode calls (#585) @turbolent

## 🛠 Improvements

- Make CompositeType.Members field an ordered map (#581) @SupunS
- Add ordered map for nested types in the checker (#580) @SupunS
- Use elaborations instead of checkers (#576) @turbolent
- Check ranges over maps (#450) @turbolent
- Improve resource info merging (#606) @turbolent
- Make dictionary entries and composite fields deterministic (#614) @turbolent
- Write cached items in deterministic order (#613) @turbolent
- Add tests for KeyString function (#612) @turbolent
- Use ordered map for field initialization check (#609) @turbolent
- Use ordered maps for type parameters  (#608) @turbolent
- Use ordered maps for imported values and types (#607) @turbolent
- Cache members of composite types and interface types (#611) @turbolent
- Reuse enc mode and dec mode (#610) @turbolent
- Make resource tracking deterministic  (#603) @turbolent
- Make base values and base types deterministic (#604) @turbolent
- Make elaboration globals deterministic (#605) @turbolent
- Make activations deterministic (#601) @turbolent
- Make member set deterministic (#602) @turbolent
- Make interface sets ordered and thread-safe (#600) @turbolent
- Improve error reporting in contract update validation (#597) @SupunS
- Optimize resource tracking (#592) @turbolent
- Make iteration deterministic (#596) @turbolent
- Optimize member set by using plain Go maps and parent-child chaining (#589) @turbolent

## 🐞 Bug Fixes

- Improve Virtual Machine (#598) @turbolent
- Fix parsing 'from' keyword in imported-identifiers (#577) @SupunS

## 📖 Documentation

- Minor additions to the docs (#594) @janezpodhostnik
- Add compilation to runtime diagram (#591) @turbolent

# v0.12.11 (2021-02-15)

## 🐞 Bug Fixes

- Fix `String()` function of `DicitonaryValue` when values are deferred (#587)

# v0.12.10 (2021-02-15)

## 🐞 Bug Fixes

- Revert dictionary key string format for address values (#586)

# v0.12.9 (2021-02-15)

## 🐞 Bug Fixes

- Fix log statement ([1328925](https://github.com/onflow/cadence/commit/1328925a8221d76c41e51f1df81424566262f10c))

# v0.12.8 (2021-02-15)

## ⭐ Features

- Add an optional callback for encoding prepare function calls and debug log them to the runtime interface (#584)

# v0.12.7 (2021-02-04)

## ⭐ Features

- WebAssembly: Add support for start section (#430)
- WebAssembly: Add support for memory exports (#427)

## 🛠 Improvements

- Return a dedicated error for encoding an unsupported value (#583)
- Update language server to Cadence v0.12.6 and Go SDK v0.14.3 (#579)


# v0.12.6 (2021-02-02)

## 🛠 Improvements

- Optimize activations (#571)
- Make occurrences and origins optional (#570)
- Update to hamt 37930cf9f7d8, which contains ForEach, optimize activations (#569)
- Move global values, global types, and transaction types to elaboration (#558)
- Improve AST indices (#556)
- Cache programs before checking succeeded, properly wrap returned error (#557)

Performance of checking and interpretation has been improved, 
e.g. for `BenchmarkCheckContractInterfaceFungibleTokenConformance`:

| Commit   | Description                                    |  Ops |          Time |
|----------|------------------------------------------------|------:|--------------:|
| 21764d89 | Baseline, v0.12.5                              |   595 | 2018162 ns/op |
| df2ba05d | Update hamt, use ForEach everywhere            |  1821 |  658515 ns/op |
| 6088ce01 | Optional occurrences and origins recording     |  2258 |  530121 ns/op |
| 429a7796 | Optimize activations, replace use of HAMT map  |  3667 |  327368 ns/op |


## ⭐ Features

- Optionally run checker tests concurrently (#554)

## 🐞 Bug Fixes

- Fix runtime type of Block.timestamp (#575)
- Use correct CommandSubmitTransaction command in entrypoint code lens (#568)
- Revert "always find the declared variable (#408)" (#561)
- Make parameter list thread-safe (#555)

## 📖 Documentation

- More developer documentation (#535)
- Update the roadmap (#553)
- Document how to inject value declarations (#547)



# v0.10.6 (2021-01-30)

## 🛠 Improvements

- Update to hamt 37930cf9f7d8, use `ForEach` instead of `FirstRest` (#566)
- Make checker occurrences and origins optional (#567)

# v0.12.5 (2021-01-22)

## ⭐ Features

- WebAssembly: Add support for memory and data sections (#425)
- Start of a compiler, with IR and WASM code generator (#409)
- Language Server: Parse pragma arguments declarations and show codelenses (#432)

## 🛠 Improvements

- Update the parser NPM package (#544)
- Update the language server and Monaco client NPM packages (#545)
- Interpreter: Always find the declared variable (#408)

## 🐞 Bug Fixes

- Fix the export of type values with restricted static types (#551)
- Fix nested enum declarations (#549)
- Language Server: Don't panic when notifications fail (#543)

# v0.12.4 (2021-01-20)

## ⭐ Features

- Argument list parsing and transaction declaration docstrings (#528)
- Add support for name section in WASM binary (#388)
- Add more WASM instructions (#381)
- Add support for export section in WASM binary writer and reader (#377)
- Generate code for instructions (#372)

## 🛠 Improvements

- WebAssembly package improvements (#542)
- Generate code for instructions (#372)

## 🐞 Bug Fixes

- Add support for multiple contracts per account to the language server (#540)


# v0.12.3 (2021-01-13)

## 🛠 Improvements

- Improve the parsing / checking error (#525)

# v0.12.2 (2021-01-13)

## 🛠 Improvements

- Export all checkers (#524)

## 📖 Documentation

- Add grammar for Cadence  (#522)
- Add developer documentation (#523)
- Fix typos in documentation (#519)


# v0.12.1 (2021-01-07)

## 🛠 Improvements

- Improve the contract deployment error message (#507)

## 🐞 Bug Fixes

- Gracefully handle type loading failure (#510)

## 📖 Documentation

- Remove completed items from roadmap (#515)
- Add presentation about implementation (#506)


# v0.10.5 (2021-01-07)

## 🛠 Improvements

- Improve fixed-point multiplication and division (#508)
- Make AST thread-safe (#440)

## 🐞 Bug Fixes

- Gracefully handle type loading failure (#511)


# v0.12.0 (2020-12-15)

## ⭐ Features

- Add `getType` function (#493): It is now possible to get the [run-time type](https://docs.onflow.org/cadence/language/run-time-types/) of a value
- Flush the cache of the storage before querying the used storage amount (#480)
- Structured type identifiers (#477)
- Allow host environment to predeclare values, predicated on location (#472)
- Add a visitor for interpreter values (#449)
- Add support for imports in WASM writer and reader (#368)
- Add support for coverage reports (#465)
- Add storage fields to accounts (#439)
- Implement `fmt.Stringer` for `cadence.Value` (#434)
- Add a function to parse a literal with a given target type (#417)

## 🛠 Improvements

- Extend event parameter types and dictionary key types (#497)
- Optimize composite and interface static types (#489)
- Optimize composite values (#488)
- Fix the export of static types (#487)
- Improve error pretty printing (#481)
- Improve fixed-point multiplication and division (#490)
- Improve error messages for contract deployment name argument checks (#475)
- Add a test for decoding a struct with an address location without name (#469)
- Refactor address locations, make composite decoding backwards-compatible (#457)
- Improve error message when there are constructor argument (#455)
- Make AST thread-safe (#440)
- Add position information to interpreter errors (#424)

## 🐞 Bug Fixes

- Declare new contract's nested values before evaluating initializer (#504)
- Infer address location name from type ID for static types (#468)
- Fix optional value's type function (#458)
- Don't use the cache when deploying or updating account code (#447)
- Properly handle unspecified variable kind (#441)
- Prevent resource loss in failable downcasts (#426)

## 💥 Breaking Changes

This release contains no source-breaking changes for Cadence programs, but the following breaking changes when embedding Cadence:

- Structured type identifiers (#477)
- Add error return value to all interface methods (#470)

## 📖 Documentation

- Document that references are not storable and suggest using capabilities (#478)
- Update the diagram illustrating the architecture of the runtime (#476)
- Document the current options for syntax highlighting (#444)


# v0.10.4 (2020-12-09)

## 🛠 Improvements

- Allow non-fatal errors for all interface functions (#494, #495)
- Panic with array index out of bounds error (#496)


# v0.10.3 (2020-12-04)

## ⭐ Features

- Add storage fields to accounts (#485)

## 🛠 Improvements

- Flush the cache of the storage before querying the used storage amount (#486)


# v0.11.2 (2020-11-30)

## ⭐ Features

- Extended debug (#464)

## 🛠 Improvements

- Refactor address locations, make composite decoding backwards-compatible (#461)


# v0.10.2 (2020-11-23)

## ⭐ Features

- Extended debug (#463)

## 🛠 Improvements

- Refactor address locations, make composite decoding backwards-compatible (#460)


# v0.9.3 (2020-11-19)

## ⭐ Features

- Wrap errors to provide additional information (#451)


# v0.11.1 (2020-11-09)

## 🐞 Bug Fixes

- Don't use the cache when deploying or updating account code (#447)


# v0.10.1 (2020-11-06)

## 🐞 Bug Fixes

- Don't use the cache when deploying or updating account code (#447)


# v0.11.0 (2020-10-13)

## 💥 Breaking Changes

### Typed Paths (#403)

Paths are now typed. Paths in the storage domain have type `StoragePath`, in the private domain `PrivatePath`, and in the public domain `PublicPath`.  `PrivatePath` and `PublicPath` are subtypes of `CapabilityPath`. Both `StoragePath` and `CapabilityPath` are subtypes of `Path`.

<table>
  <tr>
    <td colspan="3">Path</td>
  </tr>
  <tr>
    <td colspan="2">CapabilityPath</td>
    <td colspan="2" rowspan="2">StoragePath</td>
  </tr>
  <tr>
    <td>PrivatePath</td>
    <td>PublicPath</td>
  </tr>
</table>

### Storage API (#403)

With paths being typed, it was possible to make the Storage API type-safer and easier to use: It is now statically checked if the correct type of path is given to a function, instead of at run-time, and therefore capability return types can now be non-optional.

The changes are as follows:

For `PublicAccount`:

- old: `fun getCapability<T>(_ path: Path): Capability<T>?` <br/>
  new: `fun getCapability<T>(_ path: PublicPath): Capability<T>`

- old: `fun getLinkTarget(_ path: Path): Path?` <br />
  new: `fun getLinkTarget(_ path: CapabilityPath): Path?`

For `AuthAccount`:

- old: `fun save<T>(_ value: T, to: Path)` <br />
  new: `fun save<T>(_ value: T, to: StoragePath)`

- old: `fun load<T>(from: Path): T?` <br />
  new: `fun load<T>(from: StoragePath): T?`

- old: `fun copy<T: AnyStruct>(from: Path): T?` <br />
  new: `fun copy<T: AnyStruct>(from: StoragePath): T?`

- old: `fun borrow<T: &Any>(from: Path): T?` <br />
  new: `fun borrow<T: &Any>(from: StoragePath): T?`

- old: `fun link<T: &Any>(_ newCapabilityPath: Path, target: Path): Capability<T>?` <br />
  new: `fun link<T: &Any>(_ newCapabilityPath: CapabilityPath, target: Path): Capability<T>?`

- old: `fun getCapability<T>(_ path: Path): Capability<T>?` <br/>
  new: `fun getCapability<T>(_ path: CapabilityPath): Capability<T>`

- old: `fun getLinkTarget(_ path: Path): Path?` <br />
  new: `fun getLinkTarget(_ path: CapabilityPath): Path?`

- old: `fun unlink(_ path: Path)` <br />
  new: `fun unlink(_ path: CapabilityPath)`

## ⭐ Features

- Add a hash function to the crypto contract (#379)
- Added npm packages for components of Cadence. This eases the development of developer tools for Cadence:

  - `cadence-language-server`: The Cadence Language Server
  - `monaco-languageclient-cadence`: Language Server Protocol client for the the Monaco editor
  - `cadence-parser`: The Cadence parser

  In addition, there is also [an example for the language server](https://github.com/onflow/cadence/tree/master/npm-packages/cadence-language-server-demo), and an [AST exploration tool](https://github.com/onflow/cadence/tree/master/tools/astexplorer) that demonstrate the use of the parser package.

- Add a command to the language server that allows getting the entry point (transaction or script) parameters (#406)

## 🛠 Improvements

- Allow references to be returned from from scripts (#400)
- Panic with a dedicated error for out of bounds array index (#396)

## 📖 Documentation

- Document resource identifiers (#394)
- Document iteration over dictionary entries (#399)

## 📦 Dependencies

- The changes to the [CBOR library](https://github.com/fxamacker/cbor) have been [merged](https://github.com/fxamacker/cbor/pull/249), so [the `replace` statement that was necessary in the last release](https://github.com/onflow/cadence/releases/tag/v0.10.0) must be removed.

# v0.10.0 (2020-10-01)

## 💥 Breaking Changes

### Contract Deployment

This release adds support for deploying multiple contracts per account.
The API for deploying has changed:

- The functions `AuthAccount.setCode` and `AuthAccount.unsafeNotInitializingSetCode` were removed (#390).
- A [new contract management API](https://docs.onflow.org/cadence/language/contracts/#deploying-updating-and-removing-contracts) has been added, which allows adding, updating, and removing multiple contracts per account (#333, #352).

  See the [updated documentation](https://docs.onflow.org/cadence/language/contracts/#deploying-updating-and-removing-contracts) for details and examples.

## ⭐ Features

### Enumerations

This release adds support for enumerations (#344).

For example, a days-of-the-week enum can be declared though:

```swift
enum Day: UInt8 {
    case Monday
    case Tuesday
    case Wednesday
    case Thursday
    case Friday
    case Saturday
    case Sunday
}
```

See the [documentation](https://docs.onflow.org/cadence/language/enumerations/) for further details and examples.

### Switch Statement

This release adds support for switch statements (#365).

```swift
fun describe(number: Int): String {
    switch number {
    case 1:
        return "one"
    case 2:
        return "two"
    default:
        return "other"
    }
}
```

See the [documentation](https://docs.onflow.org/cadence/language/control-flow/#switch) for further details and examples.

### Code Formatter

Development of a code formatter has started, in form of a plugin for Prettier (#348).
If you would like to contribute, please let us know!

## 🛠 Improvements

- Limitations on data structure complexity are now enforced when decoding and also when encoding storage data, e.g number of array elements, number of dictionary entries, etc. (#370)
- Using the force-unwrap operator on a non-optional is no longer an error, but a hint is reported suggesting the removal of the unnecessary operation (#347)
- Language Server: The features requiring a Flow client connection are now optional. This allows using the language server in editors other than Visual Studio Code, such as Emacs, Vim, etc. (#303)

## 🐞 Bug Fixes

- Fixed the encoding of bignums in storage (#370)
- Fixed the timing of recording composite-type explicit-conformances (#356)
- Added support for exporting and JSON encoding/decoding of type values and capabilities (#374)
- Added support for exporting/importing and JSON encoding/decoding of path values (#319)
- Fixed the handling of empty integer literals in the checker (#354)
- Fixed the non-storable fields error (#350)

## 📖 Documentation

- Added a [roadmap describing planned features and ideas](https://github.com/onflow/cadence/blob/master/ROADMAP.md) (#367)
- Various documentation improvements (#385). Thanks to @andrejtokarcik for contributing this!


# v0.8.2 (2020-08-28)

## 🐞 Bug Fixes

- Copy values on return (#355)

# v0.8.1 (2020-08-24)

## 🐞 Bug Fixes

- Validate script argument count (#316)

# v0.8.0 (2020-08-10)

This release focuses on improvements, bug fixes, and bringing the documentation up-to-date.

## 🛠 Improvements

- Improved support on Windows: Treat carriage returns as space (#304)
- Improved JSON marshalling for more AST elements (#292, #286)
- Recursively check if assignment target expression is valid, don't copy returned values (#288)
- Consider existing prefix when suggesting completion items (#285)
- Include available declarations in "not declared" error (#280)
- Enforce the requirement for types to be storable in more places:
  - Transaction/script parameter types (#305)
  - Script return type (#291)
  - Arguments passed to the account `load` and `store` functions (#251)

  Previously, using non-storable failed at run-time, now the programs are rejected statically.

## ⭐ Features

- Add a version constant (#289)
- Language Server: Include error notes as related information in diagnostic (#276)

## 🐞 Bug Fixes

- Don't return an error when a location type is not supported (#290)
- Handle incomplete types in checker (#284)
- Fix the suggestion in the error when an interface is used as a type (#281)

## 📖 Documentation

- Brought the documentation up-to-date, include all new language features and API changes (#275, #277, #279)

## 💥 Breaking Changes

- Arguments passed to `cadence.Value#WithType` are now typed as pointers (`*cadence.Type`) rather than values (`cadence.Type`)

  Example:

  ```go
  myEventType := cadence.EventType{...})

  // this 👇 
  myEvent := cadence.NewEvent(...).WithType(myEventType)

  // changes to this 👇 
  myEvent := cadence.NewEvent(...).WithType(&myEventType)
  ```

# v0.7.0 (2020-08-05)

This release contains a lot of improvements to the language server, which improve the development experience in the Visual Studio Code extension, and will also soon be integrated into the Flow Playground.

## ⭐ Features

- Added member completion to the language server (#257)
  - Insert the argument list when completing functions (#268)
  - Offer keyword snippets (#268)
- Enabled running the language server in the browser by compiling it to WebAssembly (#242)
- Added support for docstrings to the parser (#246, #255)
- Added support for pragmas to the parser (#239)
- Started a source compatibility suite (#226, #270).

  Cadence will be regularly checked for compatibility with existing code to ensure it does not regress in source compatibility and performance. As a start, the [Fungible Token standard](https://github.com/onflow/flow-ft) and [Non-Fungible Token standard](https://github.com/onflow/flow-ft) repositories are checked.

  Consider requesting your repository to be included in the compatibility suite!
- Added a simple minifier (#252)

## 🛠 Improvements

- Made embedding the language server easier (#274, #262)
- Extended the JSON serialization for Cadence values and types (#260, #264, #265, #266, #271, #273)
- Improved conformance checking (#245). Made return types covariant, i.e. allow subtypes for return types of functions to satisfy interface conformance.

## 💥 Breaking Changes

- Made strings immutable (#269)
- Moved the Visual Studio Code extension to a new repository: <https://github.com/onflow/vscode-flow> (#244)
- Added a magic prefix to stored data (#236).
  This allows versioning the data. Existing data without a prefix is migrated automatically
- Changed the order in which fields in values and types are exported to declaration order (#272)

## 🐞 Bug Fixes

- Removed the service account from the list of usable accounts in the language server (#261)
- Fixed a parsing ambiguity for return types (#267)
- Fixed non-composite reference subtyping (#253)

## 🧰 Maintenance

- Removed the old parser (#249)

# v0.6.0 (2020-07-14)

This is a small release with some bug fixes, internal improvements, and one breaking change for code that embeds Cadence.

## 💥 Breaking Changes

- Removed the unused `controller` parameter from the storage interface methods:

  - `func GetValue(owner, controller, key []byte) (value []byte, err error)`
    → `func GetValue(owner, key []byte) (value []byte, err error)`

  - `SetValue(owner, controller, key, value []byte) (err error)`
    → `SetValue(owner, key, value []byte) (err error)`

  - `ValueExists(owner, controller, key []byte) (exists bool, err error)`
    → `ValueExists(owner, key []byte) (exists bool, err error)`

  This is only a breaking change in the implementation of Cadence, i.e for code that embeds Cadence, not in the Cadence language, i.e. for users of Cadence.

## ⭐ Features

- Added an optional callback for writes with high-level values to the interface:

  ```go
  type HighLevelStorage interface {

	  // HighLevelStorageEnabled should return true
	  // if the functions of HighLevelStorage should be called,
	  // e.g. SetCadenceValue
	  HighLevelStorageEnabled() bool

	  // SetCadenceValue sets a value for the given key in the storage, owned by the given account.
	  SetCadenceValue(owner Address, key string, value cadence.Value) (err error)
  }
  ```

  This is a feature in the implementation of Cadence, i.e for code that embeds Cadence, not in the Cadence language, i.e. for users of Cadence.

## 🛠 Improvements

- Don't report an error for a restriction with an invalid type
- Record the occurrences of types. This enables the "Go to definition" feature for types in the language server
- Parse member expressions without a name. This allows the checker to run. This will enable adding code completion support or members in the future.

## 🐞 Bug Fixes

- Fixed a crash when checking the invocation of a function on an undeclared variable
- Fixed handling of functions in composite values in the JSON-CDC encoding
- Fixed a potential stack overflow when checking member storability

# v0.5.0 (2020-07-10)

## ⭐ Features and Improvements

### Crypto

A new built-in contract `Crypto` was added for performing cryptographic operations.

The contract can be imported using `import Crypto`.

This first iteration provides support for validating signatures, with an API that supports multiple signatures and weighted keys.

For example, to verify two signatures with equal weights for some signed data:

```swift
let keyList = Crypto.KeyList()

let publicKeyA = Crypto.PublicKey(
    publicKey:
        "db04940e18ec414664ccfd31d5d2d4ece3985acb8cb17a2025b2f1673427267968e52e2bbf3599059649d4b2cce98fdb8a3048e68abf5abe3e710129e90696ca".decodeHex(),
    signatureAlgorithm: Crypto.ECDSA_P256
)
keyList.add(
    publicKeyA,
    hashAlgorithm: Crypto.SHA3_256,
    weight: 0.5
)

let publicKeyB = Crypto.PublicKey(
    publicKey:
        "df9609ee588dd4a6f7789df8d56f03f545d4516f0c99b200d73b9a3afafc14de5d21a4fc7a2a2015719dc95c9e756cfa44f2a445151aaf42479e7120d83df956".decodeHex(),
    signatureAlgorithm: Crypto.ECDSA_P256
)
keyList.add(
    publicKeyB,
    hashAlgorithm: Crypto.SHA3_256,
    weight: 0.5
)

let signatureSet = [
    Crypto.KeyListSignature(
        keyIndex: 0,
        signature:
            "8870a8cbe6f44932ba59e0d15a706214cc4ad2538deb12c0cf718d86f32c47765462a92ce2da15d4a29eb4e2b6fa05d08c7db5d5b2a2cd8c2cb98ded73da31f6".decodeHex()
    ),
    Crypto.KeyListSignature(
        keyIndex: 1,
        signature:
            "bbdc5591c3f937a730d4f6c0a6fde61a0a6ceaa531ccb367c3559335ab9734f4f2b9da8adbe371f1f7da913b5a3fdd96a871e04f078928ca89a83d841c72fadf".decodeHex()
    )
]

// "foo", encoded as UTF-8, in hex representation
let signedData = "666f6f".decodeHex()

let isValid = keyList.isValid(
    signatureSet: signatureSet,
    signedData: signedData
)
```

### Run-time Types

The type `Type` was added to represent types at run-time.

To create a type value, use the constructor function `Type<T>()`, which accepts the static type as a type argument.

This is similar to e.g. `T.self` in Swift, `T::class` in Kotlin, and `T.class` in Java.

For example, to represent the type `Int` at run-time:

```swift
let intType: Type = Type<Int>()
```

This works for both built-in and user-defined types. For example, to get the type value for a resource:

```swift
resource Collectible {}

let collectibleType = Type<@Collectible>()
```

The function `fun isInstance(_ type: Type): Bool` can be used to check if a value has a certain type:

```swift
let collectible <- create Collectible()
let collectibleType = Type<@Collectible>()
let result = collectible.isInstance(collectibleType)
```

For example, this allows implementing a marketplace sale resource:

```swift
pub resource SimpleSale {

    pub var objectForSale: @AnyResource?
    pub let priceForObject: UFix64
    pub let requiredCurrency: Type
    pub let paymentReceiver: Capability<&{FungibleToken.Receiver}>

    init(
        objectForSale: @AnyResource,
        priceForObject: UFix64,
        requiredCurrency: Type,
        paymentReceiver: Capability<&{FungibleToken.Receiver}>
    ) {
        self.objectForSale <- objectForSale
        self.priceForObject = priceForObject
        self.requiredCurrency = requiredCurrency
        self.paymentReceiver = paymentReceiver
    }

    destroy() {
        destroy self.objectForSale
    }

    pub fun buyObject(purchaseAmount: @FungibleToken.Vault): @AnyResource {
        pre {
            self.objectForSale != nil
            purchaseAmount.balance >= self.priceForObject
            purchaseAmount.isInstance(self.requiredCurrency)
        }

        let receiver = self.paymentReceiver.borrow()
            ?? panic("failed to borrow payment receiver capability")

        receiver.deposit(from: <-purchaseAmount)
        let objectForSale <- self.objectForSale <- nil
        return <-objectForSale
    }
}
```

### New Parser

The existing parser was replaced by a new implementation, completely written from scratch, producing the same result as the old parser.

It is significantly faster than the old parser. For example, the following benchmark shows the difference for parsing all fungible and non-fungible token contracts and example transactions:

```sh
$ go run ./cmd/parse -bench ../../flow-{n,}ft/{contracts,transactions}/*
flow-nft/contracts/ExampleNFT.cdc:          [old]        9  111116110 ns/op
flow-nft/contracts/ExampleNFT.cdc:          [new]     2712     463483 ns/op
flow-nft/contracts/NonFungibleToken.cdc:    [old]      393    3097172 ns/op
flow-nft/contracts/NonFungibleToken.cdc:    [new]     3489     348496 ns/op
flow-nft/transactions/mint_nft.cdc:         [old]      700    1574730 ns/op
flow-nft/transactions/mint_nft.cdc:         [new]    12770      94070 ns/op
flow-nft/transactions/read_nft_data.cdc:    [old]      994    1222887 ns/op
flow-nft/transactions/read_nft_data.cdc:    [new]    15242      79295 ns/op
flow-nft/transactions/setup_account.cdc:    [old]     1281     879751 ns/op
flow-nft/transactions/setup_account.cdc:    [new]    16675      71759 ns/op
flow-nft/transactions/transfer_nft.cdc:     [old]       72   16417568 ns/op
flow-nft/transactions/transfer_nft.cdc:     [new]    10000     109380 ns/op
flow-ft/contracts/CustodialDeposit.cdc:     [old]       18   64938763 ns/op
flow-ft/contracts/CustodialDeposit.cdc:     [new]     3482     354662 ns/op
flow-ft/contracts/FlowToken.cdc:            [old]        7  177111544 ns/op
flow-ft/contracts/FlowToken.cdc:            [new]     1920     640557 ns/op
flow-ft/contracts/FungibleToken.cdc:        [old]      232    5324962 ns/op
flow-ft/contracts/FungibleToken.cdc:        [new]     2947     419529 ns/op
flow-ft/contracts/TokenForwarding.cdc:      [old]       44   25136749 ns/op
flow-ft/contracts/TokenForwarding.cdc:      [new]     7183     172917 ns/op
flow-ft/transactions/burn_tokens.cdc:       [old]       37   31475393 ns/op
flow-ft/transactions/burn_tokens.cdc:       [new]    11361     105932 ns/op
flow-ft/transactions/create_forwarder.cdc:  [old]      733    1636347 ns/op
flow-ft/transactions/create_forwarder.cdc:  [new]     8127     147520 ns/op
flow-ft/transactions/create_minter.cdc:     [old]     1306     923201 ns/op
flow-ft/transactions/create_minter.cdc:     [new]    15240      77666 ns/op
flow-ft/transactions/custodial_deposit.cdc: [old]       69   16504795 ns/op
flow-ft/transactions/custodial_deposit.cdc: [new]     7940     144228 ns/op
flow-ft/transactions/get_balance.cdc:       [old]     1094    1111272 ns/op
flow-ft/transactions/get_balance.cdc:       [new]    18741      65745 ns/op
flow-ft/transactions/get_supply.cdc:        [old]     1392     740989 ns/op
flow-ft/transactions/get_supply.cdc:        [new]    46008      26719 ns/op
flow-ft/transactions/mint_tokens.cdc:       [old]       72   17435128 ns/op
flow-ft/transactions/mint_tokens.cdc:       [new]     8841     124117 ns/op
flow-ft/transactions/setup_account.cdc:     [old]     1219    1089357 ns/op
flow-ft/transactions/setup_account.cdc:     [new]    13797      84948 ns/op
flow-ft/transactions/transfer_tokens.cdc:   [old]       74   17011751 ns/op
flow-ft/transactions/transfer_tokens.cdc:   [new]     9578     125829 ns/op
```

The new parser also provides better error messages and will allow us to provide even better error messages in the future.

The new parser is enabled by default – if you discover any problems or regressions, please report them!

### Typed Capabilities

Capabilities now accept an optional type argument, the reference type the capability should be borrowed as.

If a type argument is provided, it will be used for `borrow` and `check` calls, so there is no need to provide a type argument for the calls anymore.

The function `getCapability` now also accepts an optional type argument, and returns a typed capability if one is provided.

For example, the following two functions have the same behaviour:

```swift
fun cap1(): &Something? {
  // The type annotation for `cap` is only added for demonstration purposes,
  // it can also be omitted, because it can be inferred from the value
  let cap: Capability = getAccount(0x1).getCapability(/public/something)!
  return cap.borrow<&Something>()
}

fun cap2(): &Something? {
  // The type annotation for `cap` is only added for demonstration purposes,
  // it can also be omitted, because it can be inferred from the value
  let cap: Capability<&Something> =
      getAccount(0x1).getCapability<&Something>(/public/something)!
  return cap.borrow()
}
```

Prefer a typed capability over a non-typed one to reduce uses / borrows to a simple `.borrow()`, instead of having to repeatedly mention the borrowed type.

### Parameters for Scripts

Just like transactions, the `main` functions of scripts can now have parameters.

For example, a script that can be passed an integer and a string, and which logs both values, can be written as:

```swift
pub fun main(x: Int, y: String) {
    log(x)
    log(y)
}
```

### Standard Library

The function `fun toString(): String` was added to all number types and addresses. It returns the textual representation of the type. A future version of Cadence will add this function to more types.

The function `fun toBytes(): [UInt8]` was added to `Address`. It returns the byte representation of the address.

The function `fun toBigEndianBytes(): [UInt8]` was added to all number types. It returns the big-endian byte representation of the number value.

## 🐞 Bug Fixes

- Fixed the checking of return statements:
  A missing return value is now properly reported
- Fixed the checking of function invocations:
  Resources are temporarily moved into the invoked function
- Disabled caching of top-level programs (transactions, scripts)
- Fixed the comparison of optional values
- Fixed parsing of unpadded fixed point numbers
  in the JSON encoding of Cadence values (JSON-CDC)

## 💥 Breaking Changes

### Field Types

Fields which have a non-storable type are now rejected:

Non-storable types are:

- Functions (e.g. `((Int): Bool)`)
- Accounts (`AuthAccount` / `PublicAccount`)
- Transactions
- `Void`

A future version will also make references non-storable. Instead of storing a reference, store a capability and borrow it to acquire a reference.

### Byte Arrays

Cadence now represents all byte arrays as `[UInt8]` rather than `[Int]`. This affects the following functions:

- `String.decodeHex(): [Int]` => `String.decodeHex(): [UInt8]`
- `AuthAccount.addPublicKey(publicKey: [Int])` => `AuthAccount.addPublicKey(publicKey: [UInt8])`
- `AuthAccount.setCode(code: [Int])` => `AuthAccount.setCode(code: [UInt8])`

However, array literals such as the following will still be inferred as `[Int]`, meaning that they can't be used directly:

```swift
myAccount.addPublicKey([1, 2, 3])
```

Consider using `String.decodeHex()` for now until type inference has been improved.


# v0.4.0 (2020-06-02)

## 💥 Breaking Changes

- The `AuthAccount` constructor now requires a `payer` parameter, from which the account creation fee is deducted. The `publicKeys` and `code` parameters were removed.

## ⭐ Features

- Added the function `unsafeNotInitializingSetCode` to `AuthAccount` which, like `setCode`, updates the account's code but does not create a new contract object or invoke its initializer. This function can be used to upgrade account code without overwriting the existing stored contract object.

  ⚠️ This is potentially unsafe because no data migration is performed. For example, changing a field's type results in undefined behaviour (i.e. could lead to the execution being aborted).

## 🐞 Bug Fixes

- Fixed variable shadowing for-in identifier to shadow outer scope (#72)
- Fixed order of declaring conformances and usages
- Fixed detection of resource losses in invocation of optional chaining result

# v0.3.0 (2020-05-26)

## 💥 Breaking Changes

- Dictionary values which are resources are now stored in separate storage keys. This is transparent to programs, so is not breaking on a program level, but on the integration level.
- The state encoding was switched from Gob to CBOR. This is transparent to programs, so is not breaking on a program level, but on the integration level.
- The size of account addresses was reduced from 160 bits to 64 bits

## ⭐ Features

- Added bitwise operations
- Added support for caching of program ASTs
- Added metrics
- Block info can be queried now

## 🛠 Improvements

- Unnecessary writes are now avoided by tracking modifications
- Improved access modifier checks
- Improved the REPL
- Decode and validate transaction arguments
- Work on a new parser has started and is almost complete

## 🐞 Bug Fixes

- Fixed account code updated
- Fixed the destruction of arrays
- Fixed type requirements
- Implemented modulo for `[U]Fix64`, fix modulo with negative operands
- Implemented division for `[U]Fix64`
- Fixed the `flow.AccountCodeUpdated` event
- Fixed number conversion
- Fixed function return checking
