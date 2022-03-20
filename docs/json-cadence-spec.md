---
title: JSON-Cadence Data Interchange Format
---

> Version 0.2.0

JSON-Cadence is a data interchange format used to represent Cadence values as language-independent JSON objects.

This format includes less type information than a complete [ABI](https://en.wikipedia.org/wiki/Application_binary_interface), and instead promotes the following tenets:

- **Human-readability** - JSON-Cadence is easy to read and comprehend, which speeds up development and debugging.
- **Compatibility** - JSON is a common format with built-in support in most high-level programming languages, making it easy to parse on a variety of platforms.
- **Portability** - JSON-Cadence is self-describing and thus can be transported and decoded without accompanying type definitions (i.e. an ABI).

---

## Void

```json
{
  "type": "Void"
}
```

### Example

```json
{
  "type": "Void"
}
```

---

## Optional

```json
{
  "type": "Optional",
  "value": null | <value>
}
```

### Example

```json
// Non-nil

{
  "type": "Optional",
  "value": {
    "type": "UInt8",
    "value": "123"
  }
}

// Nil

{
  "type": "Optional",
  "value": null
}
```

---

## Bool

```json
{
  "type": "Bool",
  "value": true | false
}
```

### Example

```json
{
  "type": "Bool",
  "value": true
}
```

---

## String

```json
{
  "type": "String",
  "value": "..."
}

```

### Example

```json
{
  "type": "String",
  "value": "Hello, world!"
}
```

---

## Address

```json
{
  "type": "Address",
  "value": "0x0" // as hex-encoded string with 0x prefix
}
```

```json
{
  "type": "Address",
  "value": "Fx0" // as hex-encoded string with Fx prefix
}
```


### Example

```json
{
  "type": "Address",
  "value": "0x1234"
}
```

---

## Integers

`[U]Int`, `[U]Int8`, `[U]Int16`, `[U]Int32`,`[U]Int64`,`[U]Int128`, `[U]Int256`,  `Word8`, `Word16`, `Word32`, or `Word64`

Although JSON supports integer literals up to 64 bits, all integer types are encoded as strings for consistency.

While the static type is not strictly required for decoding, it is provided to inform client of potential range.

```json
{
  "type": "<type>",
  "value": "<decimal string representation of integer>"
}
```

### Example

```json
{
  "type": "UInt8",
  "value": "123"
}
```

---

## Fixed Point Numbers

`[U]Fix64`

Although fixed point numbers are implemented as integers, JSON-Cadence uses a decimal string representation for readability.

```json
{
    "type": "[U]Fix64",
    "value": "<integer>.<fractional>"
}
```

### Example

```json
{
    "type": "Fix64",
    "value": "12.3"
}
```

---

## Array

```json
{
  "type": "Array",
  "value": [
    <value at index 0>,
    <value at index 1>
    // ...
  ]
}
```

### Example

```json
{
  "type": "Array",
  "value": [
    {
      "type": "Int16",
      "value": "123"
    },
    {
      "type": "String",
      "value": "test"
    },
    {
      "type": "Bool",
      "value": true
    }
  ]
}
```

---

## Dictionary

Dictionaries are encoded as a list of key-value pairs to preserve the deterministic ordering implemented by Cadence.

```json
{
  "type": "Dictionary",
  "value": [
    {
      "key": "<key>",
      "value": <value>
    },
    ...
  ]
}
```

### Example

```json
{
  "type": "Dictionary",
  "value": [
    {
      "key": {
        "type": "UInt8",
        "value": "123"
      },
      "value": {
        "type": "String",
        "value": "test"
      }
    }
  ],
  // ...
}
```

---

## Composites (Struct, Resource, Event, Contract, Enum)

Composite fields are encoded as a list of name-value pairs in the order in which they appear in the composite type declaration.

```json
{
  "type": "Struct" | "Resource" | "Event" | "Contract" | "Enum",
  "value": {
    "id": "<fully qualified type identifier>",
    "fields": [
      {
        "name": "<field name>",
        "value": <field value>
      },
      // ...
    ]
  }
}
```

### Example

```json
{
  "type": "Resource",
  "value": {
    "id": "0x3.GreatContract.GreatNFT",
    "fields": [
      {
        "name": "power",
        "value": {"type": "Int", "value": "1"}
      }
    ]
  }
}
```

---

## Path

```json
{
  "type": "Path",
  "value": {
    "domain": "storage" | "private" | "public",
    "identifier": "..."
  }
}
```

### Example

```json
{
  "type": "Path",
  "value": {
    "domain": "storage",
    "identifier": "flowTokenVault"
  }
}
```

---

## Type

```json
{
  "type": "Type",
  "value": {
    "staticType": "..."
  }
}
```

### Example

```json
{
  "type": "Type",
  "value": {
    "staticType": "Int"
  }
}
```

---

## Capability

```json
{
  "type": "Capability",
  "value": {
    "path": <path>,
    "address": "0x0",  // as hex-encoded string with 0x prefix
    "borrowType": "<type ID>",
  }
}
```

### Example

```json
{
  "type": "Capability",
  "value": {
    "path": "/public/someInteger",
    "address": "0x1",
    "borrowType": "Int",
  }
}
```
