# Wire to Kessoku Migration Tool Architecture

## CLI Integration
Add new `migrate` subcommand to existing kessoku CLI:
```bash
kessoku migrate [flags] <wire_files...>
```

### CLI Flags
- `--dry-run`: Show changes without writing files
- `--output-dir`: Specify output directory for migrated files
- `--backup`: Create backup of original files
- `--force`: Overwrite existing files without confirmation

## Module Structure
```
internal/
├── migrate/
│   ├── migrator.go       # Main migration orchestrator
│   ├── parser.go         # Wire pattern AST parser
│   ├── transformer.go    # Wire -> Kessoku transformations
│   ├── writer.go         # File writing and backup logic
│   └── patterns.go       # Wire pattern definitions
```

## Core Components

### 1. Migrator (orchestrator)
```go
type Migrator struct {
    parser      *Parser
    transformer *Transformer
    writer      *Writer
    options     MigrationOptions
}

func (m *Migrator) MigrateFiles(files []string) error
```

### 2. Parser (AST analysis)
```go
type Parser struct {
    pkg *packages.Package
}

func (p *Parser) ParseWireFile(filename string) (*WireFile, error)
func (p *Parser) FindWirePatterns(file *ast.File) ([]WirePattern, error)
```

### 3. Transformer (pattern conversion)
```go
type Transformer struct{}

func (t *Transformer) Transform(patterns []WirePattern) ([]KessokuPattern, error)
func (t *Transformer) ConvertBuild(pattern *WireBuildPattern) *KessokuInjectPattern
func (t *Transformer) ConvertSet(pattern *WireSetPattern) *KessokuSetPattern
```

### 4. Writer (file output)
```go
type Writer struct {
    options WriteOptions
}

func (w *Writer) WriteKessokuFile(file *KessokuFile) error
func (w *Writer) CreateBackup(filename string) error
```

## Pattern Definitions

### Wire Patterns to Detect
1. `//go:build wireinject` build constraints
2. `wire.Build()` function calls
3. `wire.NewSet()` variable declarations
4. `wire.Bind()` interface bindings
5. `wire.Value()` value injections
6. `wire.Struct()` struct providers
7. Injector function signatures

### Kessoku Output Patterns  
1. `//go:generate go tool kessoku $GOFILE` directives
2. `kessoku.Inject[]` variable declarations
3. `kessoku.Set()` provider sets
4. `kessoku.Bind[]()` interface bindings
5. `kessoku.Value()` value injections
6. `kessoku.Provide()` wrapper functions

## Migration Process Flow
1. **Parse**: Load and analyze wire.go files
2. **Extract**: Identify wire patterns using AST
3. **Transform**: Convert to equivalent kessoku patterns
4. **Generate**: Create new Go source code
5. **Write**: Output transformed files with proper naming
6. **Cleanup**: Remove wire build constraints and placeholder code