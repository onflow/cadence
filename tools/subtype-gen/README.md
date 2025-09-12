# Subtype Generator

A Go code generator that reads subtype checking rules from a YAML file and generates the `checkSubTypeWithoutEquality` function.

## Structure

The generator is split into three main files:

- **`main.go`** - Entry point and CLI handling
- **`parser.go`** - YAML parsing and type definitions
- **`generator.go`** - Code generation logic

## Usage

```bash
go run main.go parser.go generator.go -rules rules.yaml -stdout
```

## Flags

- `-rules` - Path to YAML rules file (default: rules.yaml)
- `-out` - Output file path or '-' for stdout (default: -)
- `-pkg` - Target Go package name (default: sema)
- `-stdout` - Write to stdout

## Files

- `rules.yaml` - Input DSL defining subtype checking rules
- `generated_code.go` - Example output (not used in build)