package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/koyashimano/matlab-formatter/internal/formatter"
)

var errMissingFilename = errors.New("missing filename")

func main() {
	opts := formatter.DefaultOptions()

	fs := flag.NewFlagSet("matlabformatter", flag.ExitOnError)
	write := fs.Bool("w", false, "Write result to source file instead of stdout")
	startLine := fs.Int("startLine", opts.StartLine, "Start line (1-based)")
	endLine := fs.Int("endLine", opts.EndLine, "End line (inclusive, 0 for end of file)")
	indentWidth := fs.Int("indentWidth", opts.IndentWidth, "Number of spaces per indentation level")
	separateBlocks := fs.Bool("separateBlocks", opts.SeparateBlocks, "Insert blank lines between blocks")
	indentMode := fs.String("indentMode", opts.IndentMode, "Indentation mode: all_functions, only_nested_functions, classic")
	addSpaces := fs.String("addSpaces", opts.AddSpaces, "Operator spacing: all_operators, exclude_pow, no_spaces")
	matrixIndent := fs.String("matrixIndent", opts.MatrixIndent, "Matrix indentation: aligned, simple")

	filename, err := parseFilename(fs, os.Args[1:])
	if err != nil {
		if errors.Is(err, errMissingFilename) {
			printUsage()
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
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

	// If -w flag is set and not reading from stdin, write to file
	if *write && filename != "-" {
		var buf bytes.Buffer
		if err := f.FormatFile(filename, &buf); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		// Write to file with same permissions as original
		info, err := os.Stat(filename)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if err := os.WriteFile(filename, buf.Bytes(), info.Mode()); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	} else {
		if err := f.FormatFile(filename, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "usage: matlabformatter [options...] filename\n")
	fmt.Fprintf(os.Stderr, "  OPTIONS:\n")
	fmt.Fprintf(os.Stderr, "    -w (default false) - Write result to source file instead of stdout\n")
	opts := formatter.DefaultOptions()
	fmt.Fprintf(os.Stderr, "    --startLine=int (default %d)\n", opts.StartLine)
	fmt.Fprintf(os.Stderr, "    --endLine=int (default %d)\n", opts.EndLine)
	fmt.Fprintf(os.Stderr, "    --indentWidth=int (default %d)\n", opts.IndentWidth)
	fmt.Fprintf(os.Stderr, "    --separateBlocks=bool (default %t)\n", opts.SeparateBlocks)
	fmt.Fprintf(os.Stderr, "    --indentMode=string (default %s)\n", opts.IndentMode)
	fmt.Fprintf(os.Stderr, "    --addSpaces=string (default %s)\n", opts.AddSpaces)
	fmt.Fprintf(os.Stderr, "    --matrixIndent=string (default %s)\n", opts.MatrixIndent)
}

func parseFilename(fs *flag.FlagSet, args []string) (string, error) {
	if err := fs.Parse(args); err != nil {
		return "", err
	}

	if fs.NArg() == 0 {
		return "", errMissingFilename
	}

	if fs.NArg() > 1 {
		return "", fmt.Errorf("too many arguments")
	}

	return fs.Arg(0), nil
}
