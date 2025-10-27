# matlab-formatter

Go implementation of the MATLAB formatter used by the VS Code extension. The formatter mirrors the behaviour of the original Python script and can be invoked from the command line.

## Installation

Install the formatter globally to use it from any directory:

```bash
go install github.com/koyashimano/matlab-formatter/cmd/matlabformatter@latest
```

After installation, you can use `matlabformatter` command from anywhere.

## Usage

```bash
matlabformatter <path-to-file> [options...]
```

### Options

- `--startLine=int` - Start line (1-based, default: 1)
- `--endLine=int` - End line (inclusive, 0 for end of file, default: 0)
- `--indentWidth=int` - Number of spaces per indentation level (default: 4)
- `--separateBlocks=bool` - Insert blank lines between blocks (default: true)
- `--indentMode=string` - Indentation mode: `all_functions`, `only_nested_functions`, `classic` (default: all_functions)
- `--addSpaces=string` - Operator spacing: `all_operators`, `exclude_pow`, `no_spaces` (default: exclude_pow)
- `--matrixIndent=string` - Matrix indentation: `aligned`, `simple` (default: aligned)

### Examples

Format a MATLAB file:

```bash
matlabformatter myfile.m
```

Format with custom indent width:

```bash
matlabformatter myfile.m --indentWidth=2
```

Read from standard input:

```bash
cat myfile.m | matlabformatter -
```

Format specific lines:

```bash
matlabformatter myfile.m --startLine=10 --endLine=50
```

## Development

### Build

Build the binary:

```bash
go build -o matlabformatter ./cmd/matlabformatter
```

### Test

Run all tests:

```bash
go test ./...
```

Run tests with verbose output:

```bash
go test -v ./...
```

### Format

Format the code:

```bash
go fmt ./...
```

Check code quality:

```bash
go vet ./...
```

### Run without installing

For development, you can run directly:

```bash
go run ./cmd/matlabformatter <path-to-file> [options...]
```
