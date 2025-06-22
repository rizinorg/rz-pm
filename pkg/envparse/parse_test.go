package envparse

import (
	"testing"
)

type testStruct struct {
	Foo   string `env:"FOO"`
	Bar   int    `env:"BAR"`
	Baz   bool   `env:"BAZ"`
	NoTag string
}

func TestUnmarshal_BasicTypes(t *testing.T) {
	input := `
FOO=hello
BAR=42
BAZ=true
NOPE=should_be_ignored
`
	var ts testStruct
	err := Unmarshal(input, &ts)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if ts.Foo != "hello" {
		t.Errorf("expected Foo=hello, got %q", ts.Foo)
	}
	if ts.Bar != 42 {
		t.Errorf("expected Bar=42, got %d", ts.Bar)
	}
	if ts.Baz != true {
		t.Errorf("expected Baz=true, got %v", ts.Baz)
	}
	if ts.NoTag != "" {
		t.Errorf("expected NoTag to be empty, got %q", ts.NoTag)
	}
}

func TestUnmarshal_UnsupportedType(t *testing.T) {
	type badStruct struct {
		Foo float64 `env:"FOO"`
	}
	input := "FOO=3.14"
	var bs badStruct
	err := Unmarshal(input, &bs)
	if err == nil {
		t.Fatal("expected error for unsupported type, got nil")
	}
}

func TestUnmarshal_IntError(t *testing.T) {
	type s struct {
		Bar int `env:"BAR"`
	}
	input := "BAR=notanint"
	var st s
	err := Unmarshal(input, &st)
	if err == nil {
		t.Fatal("expected error for int parse failure, got nil")
	}
}

func TestUnmarshal_BoolError(t *testing.T) {
	type s struct {
		Baz bool `env:"BAZ"`
	}
	input := "BAZ=notabool"
	var st s
	err := Unmarshal(input, &st)
	if err == nil {
		t.Fatal("expected error for bool parse failure, got nil")
	}
}

func TestUnmarshal_NonPointer(t *testing.T) {
	input := "FOO=bar"
	var ts testStruct
	err := Unmarshal(input, ts) // not a pointer
	if err == nil {
		t.Fatal("expected error for non-pointer, got nil")
	}
}

func TestUnmarshal_NonStructPointer(t *testing.T) {
	input := "FOO=bar"
	var s string
	err := Unmarshal(input, &s)
	if err == nil {
		t.Fatal("expected error for non-struct pointer, got nil")
	}
}

func TestUnmarshal_EmptyInput(t *testing.T) {
	var ts testStruct
	err := Unmarshal("", &ts)
	if err != nil {
		t.Fatalf("Unmarshal failed on empty input: %v", err)
	}
	// All fields should be zero values
	if ts.Foo != "" || ts.Bar != 0 || ts.Baz != false {
		t.Errorf("expected zero values, got %+v", ts)
	}
}
