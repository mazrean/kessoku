package migrate

import (
	"go/token"
	"strings"
	"testing"
)

func TestParseErrorError(t *testing.T) {
	tests := []struct {
		name    string
		wantSub string
		err     ParseError
	}{
		{
			name: "basic error",
			err: ParseError{
				Kind:    ParseErrorMissingConstructor,
				File:    "test.go",
				Pos:     token.Pos(100),
				Message: "missing constructor",
			},
			wantSub: "missing constructor",
		},
		{
			name: "error with file info",
			err: ParseError{
				Kind:    ParseErrorMissingConstructor,
				File:    "wire.go",
				Pos:     token.Pos(200),
				Message: "test error message",
			},
			wantSub: "test error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if !strings.Contains(got, tt.wantSub) {
				t.Errorf("ParseError.Error() = %q, want to contain %q", got, tt.wantSub)
			}
		})
	}
}

func TestWarningMessage(t *testing.T) {
	tests := []struct {
		name    string
		wantSub string
		warning Warning
	}{
		{
			name: "no wire import warning",
			warning: Warning{
				Code:    WarnNoWireImport,
				Message: "No wire import found",
			},
			wantSub: "No wire import found",
		},
		{
			name: "no patterns warning",
			warning: Warning{
				Code:    WarnNoWirePatterns,
				Message: "No wire patterns found",
			},
			wantSub: "No wire patterns found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.warning.Message
			if !strings.Contains(got, tt.wantSub) {
				t.Errorf("Warning.Message = %q, want to contain %q", got, tt.wantSub)
			}
		})
	}
}

func TestBaseWirePatternPosition(t *testing.T) {
	pattern := &WireNewSet{
		baseWirePattern: baseWirePattern{
			Pos:  token.Pos(100),
			File: "test.go",
		},
	}

	got := pattern.Position()
	if got != token.Pos(100) {
		t.Errorf("Position() = %d, want 100", got)
	}
}

func TestWirePatternMethods(t *testing.T) {
	// Test that all wire patterns implement the wirePattern method
	patterns := []WirePattern{
		&WireNewSet{},
		&WireBind{},
		&WireValue{},
		&WireInterfaceValue{},
		&WireStruct{},
		&WireFieldsOf{},
		&WireProviderFunc{},
		&WireSetRef{},
		&WireBuild{},
	}

	for _, p := range patterns {
		// Just verify they implement the interface (no panic)
		p.wirePattern()
		_ = p.Position()
	}
}

func TestKessokuPatternMethods(t *testing.T) {
	// Test that all kessoku patterns implement the kessokuPattern method
	patterns := []KessokuPattern{
		&KessokuSet{},
		&KessokuProvide{},
		&KessokuBind{},
		&KessokuValue{},
		&KessokuSetRef{},
		&KessokuInject{},
	}

	for _, p := range patterns {
		// Just verify they implement the interface (no panic)
		p.kessokuPattern()
	}
}
