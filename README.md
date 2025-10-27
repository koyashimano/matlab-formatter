# matlab-formatter

Go implementation of the MATLAB formatter used by the VS Code extension. The formatter mirrors the behaviour of the original Python script and can be invoked from the command line.

## Usage

```bash
go run ./cmd/matlabformatter <path-to-file> [--startLine=1 --endLine=0 --indentWidth=4 --separateBlocks=true --indentMode=all_functions --addSpaces=exclude_pow --matrixIndent=aligned]
```

Pass `-` as the filename to read from standard input. The defaults match the VS Code extension; adjust the flags as needed for your workflow.
