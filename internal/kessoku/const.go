package kessoku

const (
	kessokuPkgPath  = "github.com/mazrean/kessoku"
	errgroupPkgPath = "golang.org/x/sync/errgroup"
	errgroupPkgName = "errgroup"
	contextPkgPath  = "context"
	contextPkgName  = "context"
	contextTypeName = "Context"
)

var (
	goPredeclaredIdentifiers = [44]string{
		// Types
		"any", "bool", "byte", "comparable",
		"complex64", "complex128", "error", "float32", "float64",
		"int", "int8", "int16", "int32", "int64", "rune", "string",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",

		// Constants
		"true", "false", "iota",

		// Zero value
		"nil",

		// Functions
		"append", "cap", "clear", "close", "complex", "copy", "delete", "imag", "len",
		"make", "max", "min", "new", "panic", "print", "println", "real", "recover",
	}
	goReservedKeywords = [25]string{
		"break", "default", "func", "interface", "select",
		"case", "defer", "go", "map", "struct",
		"chan", "else", "goto", "package", "switch",
		"const", "fallthrough", "if", "range", "type",
		"continue", "for", "import", "return", "var",
	}
)
