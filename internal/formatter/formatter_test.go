package formatter

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestFormatterMatchesReferenceSample(t *testing.T) {
	unformatted, err := os.ReadFile("testdata/sample_unformatted.m")
	if err != nil {
		t.Fatalf("read unformatted: %v", err)
	}
	expected, err := os.ReadFile("testdata/sample_formatted.m")
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}

	fmttr, err := New(DefaultOptions())
	if err != nil {
		t.Fatalf("formatter init: %v", err)
	}

	lines, err := readLines(bytes.NewReader(unformatted))
	if err != nil {
		t.Fatalf("readLines unformatted: %v", err)
	}
	formatted, err := fmttr.FormatLines(lines)
	if err != nil {
		t.Fatalf("format lines: %v", err)
	}

	got := strings.Join(formatted, "\n") + "\n"
	want := string(expected)

	if got != want {
		t.Fatalf("formatted content mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestFormatLinesPartialRangePreservesSurroundingLines(t *testing.T) {
	lines := []string{
		"function y=foo(x)",
		"    if x>0",
		"y=x+1;",
		"    end",
		"end",
	}

	opts := DefaultOptions()
	opts.StartLine = 2
	opts.EndLine = 4

	fmttr, err := New(opts)
	if err != nil {
		t.Fatalf("formatter init: %v", err)
	}

	got, err := fmttr.FormatLines(lines)
	if err != nil {
		t.Fatalf("FormatLines: %v", err)
	}

	want := []string{
		"function y=foo(x)",
		"    if x > 0",
		"        y = x + 1;",
		"    end",
		"",
		"end",
	}

	if len(got) != len(want) {
		t.Fatalf("unexpected line count: got %d want %d\nlines: %#v", len(got), len(want), got)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d mismatch: got %q want %q", i+1, got[i], want[i])
		}
	}
}

func TestFormatLinesStartBeyondFileIsNoOp(t *testing.T) {
	lines := []string{
		"function y=foo(x)",
		"end",
	}

	opts := DefaultOptions()
	opts.StartLine = 99
	opts.EndLine = 0

	fmttr, err := New(opts)
	if err != nil {
		t.Fatalf("formatter init: %v", err)
	}

	got, err := fmttr.FormatLines(lines)
	if err != nil {
		t.Fatalf("FormatLines: %v", err)
	}

	if len(got) != len(lines) {
		t.Fatalf("unexpected line count: got %d want %d", len(got), len(lines))
	}

	for i := range lines {
		if got[i] != lines[i] {
			t.Fatalf("line %d mismatch: got %q want %q", i+1, got[i], lines[i])
		}
	}
}

func TestFormatLinesDanglingEndsReduceIndent(t *testing.T) {
	lines := []string{
		"function foo",
		"    if a",
		"        if b",
		"            disp('x');",
		"        end",
		"    end",
		"end",
	}

	opts := DefaultOptions()
	opts.StartLine = 5
	opts.EndLine = 6

	fmttr, err := New(opts)
	if err != nil {
		t.Fatalf("formatter init: %v", err)
	}

	got, err := fmttr.FormatLines(lines)
	if err != nil {
		t.Fatalf("FormatLines: %v", err)
	}

	want := []string{
		"function foo",
		"    if a",
		"        if b",
		"            disp('x');",
		"        end",
		"",
		"    end",
		"",
		"end",
	}

	if len(got) != len(want) {
		t.Fatalf("unexpected line count: got %d want %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d mismatch: got %q want %q", i+1, got[i], want[i])
		}
	}
}
