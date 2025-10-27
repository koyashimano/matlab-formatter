package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/koyashimano/matlab-formatter/internal/formatter"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	filename := os.Args[1]

	opts := formatter.DefaultOptions()

	fs := flag.NewFlagSet("matlabformatter", flag.ExitOnError)
	startLine := fs.Int("startLine", opts.StartLine, "Start line (1-based)")
	endLine := fs.Int("endLine", opts.EndLine, "End line (inclusive, 0 for end of file)")
	indentWidth := fs.Int("indentWidth", opts.IndentWidth, "Number of spaces per indentation level")
	separateBlocks := fs.Bool("separateBlocks", opts.SeparateBlocks, "Insert blank lines between blocks")
	indentMode := fs.String("indentMode", opts.IndentMode, "Indentation mode: all_functions, only_nested_functions, classic")
	addSpaces := fs.String("addSpaces", opts.AddSpaces, "Operator spacing: all_operators, exclude_pow, no_spaces")
	matrixIndent := fs.String("matrixIndent", opts.MatrixIndent, "Matrix indentation: aligned, simple")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	options := formatter.Options{
		StartLine:      *startLine,
		EndLine:        *endLine,
		IndentWidth:    *indentWidth,
		SeparateBlocks: *separateBlocks,
		IndentMode:     *indentMode,
		AddSpaces:      *addSpaces,
		MatrixIndent:   *matrixIndent,
	}

	f, err := formatter.New(options)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := f.FormatFile(filename, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "usage: matlabformatter filename [options...]\n")
	fmt.Fprintf(os.Stderr, "  OPTIONS:\n")
	opts := formatter.DefaultOptions()
	fmt.Fprintf(os.Stderr, "    --startLine=int (default %d)\n", opts.StartLine)
	fmt.Fprintf(os.Stderr, "    --endLine=int (default %d)\n", opts.EndLine)
	fmt.Fprintf(os.Stderr, "    --indentWidth=int (default %d)\n", opts.IndentWidth)
	fmt.Fprintf(os.Stderr, "    --separateBlocks=bool (default %t)\n", opts.SeparateBlocks)
	fmt.Fprintf(os.Stderr, "    --indentMode=string (default %s)\n", opts.IndentMode)
	fmt.Fprintf(os.Stderr, "    --addSpaces=string (default %s)\n", opts.AddSpaces)
	fmt.Fprintf(os.Stderr, "    --matrixIndent=string (default %s)\n", opts.MatrixIndent)
}
