# Subtype Generator

This tool generates the `checkSubTypeWithoutEquality` function from a declarative YAML rules file.

## Files

- `generator.go` - The main generator implementation
- `main.go` - CLI interface and main program
- `rules.yaml` - YAML rules file defining subtype checking rules
- `go.mod` & `go.sum` - Go module dependencies

## Usage

```bash
# Generate to stdout
go run generator.go -rules rules.yaml -stdout

# Generate to file
go run generator.go -rules rules.yaml -out generated_code.go

# Specify package name
go run generator.go -rules rules.yaml -pkg sema -out generated_code.go
```

## Command Line Options

- `-rules` - Path to YAML rules file (default: rules.yaml)
- `-out` - Output file path or '-' for stdout (default: -)
- `-pkg` - Target Go package name (default: sema)
- `-stdout` - Write to stdout

## YAML Rules Format

The rules file defines subtype checking rules in a declarative DSL format. Each rule specifies:
- `super` - The supertype pattern
- `sub` - The subtype pattern  
- `rule` - The condition that must be satisfied

Supported rule types:
- `always` - Always returns true
- `isResource`, `isAttachment`, `isHashableStruct`, `isStorable` - Type checks
- `equals` - Type equality checks with `oneOf` support
- `and` - Logical AND conditions
- `not` - Logical NOT conditions
- `permits` - Authorization checks
- `purity` - Function purity checks
- `typeParamsEqual`, `paramsContravariant`, `returnCovariant`, `constructorEqual` - Function type checks
- `contains` - Set containment checks
