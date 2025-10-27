package formatter

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Options captures the configuration for the formatter. Values mirror the
// original VS Code extension to maintain compatibility.
type Options struct {
	StartLine      int
	EndLine        int
	IndentWidth    int
	SeparateBlocks bool
	IndentMode     string
	AddSpaces      string
	MatrixIndent   string
}

// DefaultOptions returns the default formatter configuration.
func DefaultOptions() Options {
	return Options{
		StartLine:      1,
		EndLine:        0,
		IndentWidth:    4,
		SeparateBlocks: true,
		IndentMode:     "all_functions",
		AddSpaces:      "exclude_pow",
		MatrixIndent:   "aligned",
	}
}

// Formatter applies MATLAB formatting rules ported from the VS Code extension.
type Formatter struct {
	opts          Options
	indentMode    int
	operatorSep   float64
	matrixIndent  bool
	iwidth        int
	separateBlock bool

	ctrl1Line         *regexp.Regexp
	fcnStart          *regexp.Regexp
	ctrlStart         *regexp.Regexp
	ctrlIgnore        *regexp.Regexp
	ctrlStartSwitch   *regexp.Regexp
	ctrlCont          *regexp.Regexp
	ctrlEnd           *regexp.Regexp
	lineComment       *regexp.Regexp
	ellipsis          *regexp.Regexp
	blockCommentOpen  *regexp.Regexp
	blockCommentClose *regexp.Regexp
	blockClose        *regexp.Regexp
	ignoreCommand     *regexp.Regexp

	pString      *regexp.Regexp
	pStringDQ    *regexp.Regexp
	pComment     *regexp.Regexp
	pBlank       *regexp.Regexp
	pNumSci      *regexp.Regexp
	pNumRational *regexp.Regexp
	pIncrement   *regexp.Regexp
	pSign        *regexp.Regexp
	pColon       *regexp.Regexp
	pEllipsis    *regexp.Regexp
	pOpDot       *regexp.Regexp
	pPowDot      *regexp.Regexp
	pPow         *regexp.Regexp
	pOpComb      *regexp.Regexp
	pNot         *regexp.Regexp
	pOp          *regexp.Regexp
	pFunc        *regexp.Regexp
	pOpen        *regexp.Regexp
	pClose       *regexp.Regexp
	pComma       *regexp.Regexp
	pMultiWS     *regexp.Regexp

	initialIndent *regexp.Regexp

	ilvl           int
	istep          []int
	fstep          []int
	matrix         int
	cell           int
	isBlockComment int
	isLineComment  int
	longLine       int
	continueLine   int
	isComment      int
	ignoreLines    int
}

var (
	indentModes = map[string]int{
		"all_functions":         1,
		"only_nested_functions": -1,
		"classic":               0,
	}
	operatorSpaces = map[string]float64{
		"all_operators": 1.0,
		"exclude_pow":   0.5,
		"no_spaces":     0.0,
	}
	matrixIndentation = map[string]bool{
		"aligned": true,
		"simple":  false,
	}
	blockCommentSentinel = 1 << 30
)

// New constructs a formatter with the given options.
func New(o Options) (*Formatter, error) {
	if o.IndentWidth <= 0 {
		return nil, errors.New("indentWidth must be greater than zero")
	}

	mode, ok := indentModes[o.IndentMode]
	if !ok {
		mode = indentModes["all_functions"]
	}

	operatorSep, ok := operatorSpaces[o.AddSpaces]
	if !ok {
		operatorSep = operatorSpaces["exclude_pow"]
	}

	matIndent, ok := matrixIndentation[o.MatrixIndent]
	if !ok {
		matIndent = matrixIndentation["aligned"]
	}

	formatter := &Formatter{
		opts:              o,
		indentMode:        mode,
		operatorSep:       operatorSep,
		matrixIndent:      matIndent,
		iwidth:            o.IndentWidth,
		separateBlock:     o.SeparateBlocks,
		ctrl1Line:         regexp.MustCompile(`^(\s*)(if|while|for|try)(\W\s*\S.*\W)((end|endif|endwhile|endfor);?)(\s+\S.*|\s*$)`),
		fcnStart:          regexp.MustCompile(`^(\s*)(function|classdef)\s*(\W\s*\S.*|\s*$)`),
		ctrlStart:         regexp.MustCompile(`^(\s*)(if|while|for|parfor|try|methods|properties|events|arguments|enumeration|spmd)\s*(\W\s*\S.*|\s*$)`),
		ctrlIgnore:        regexp.MustCompile(`^(\s*)(import|clear|clearvars)(.*$)`),
		ctrlStartSwitch:   regexp.MustCompile(`^(\s*)(switch)\s*(\W\s*\S.*|\s*$)`),
		ctrlCont:          regexp.MustCompile(`^(\s*)(elseif|else|case|otherwise|catch)\s*(\W\s*\S.*|\s*$)`),
		ctrlEnd:           regexp.MustCompile(`^(\s*)((end|endfunction|endif|endwhile|endfor|endswitch);?)(\s+\S.*|\s*$)`),
		lineComment:       regexp.MustCompile(`^(\s*)%.*$`),
		ellipsis:          regexp.MustCompile(`^.*\.\.\..*$`),
		blockCommentOpen:  regexp.MustCompile(`^(\s*)%\{\s*$`),
		blockCommentClose: regexp.MustCompile(`^(\s*)%\}\s*$`),
		blockClose:        regexp.MustCompile(`^\s*[\)\]\}].*$`),
		ignoreCommand:     regexp.MustCompile(`^.*formatter\s+ignore\s+(\d*).*$`),
		pString:           regexp.MustCompile(`^(.*?[\(\[\{,;=\+\-\*\/\|\&\s]|^)\s*(\'([^\']|\'\')+\')([\)\}\]\+\-\*\/=\|\&,;].*|\s+.*|$)`),
		pStringDQ:         regexp.MustCompile(`^(.*?[\(\[\{,;=\+\-\*\/\|\&\s]|^)\s*(\"([^\"])*\")([\)\}\]\+\-\*\/=\|\&,;].*|\s+.*|$)`),
		pComment:          regexp.MustCompile(`^(.*\S|^)\s*(%.*)`),
		pBlank:            regexp.MustCompile(`^\s+$`),
		pNumSci:           regexp.MustCompile(`^(.*?\W|^)\s*(\d+\.?\d*)([eE][+-]?)(\d+)(.*)`),
		pNumRational:      regexp.MustCompile(`^(.*?\W|^)\s*(\d+)\s*(\/)\s*(\d+)(.*)`),
		pIncrement:        regexp.MustCompile(`^(.*?\S|^)\s*(\+|\-)\s*(\+|\-)\s*([\)\]\},;].*|$)`),
		pSign:             regexp.MustCompile(`^(.*?[\(\[\{,;:=\*/\s]|^)\s*(\+|\-)(\w.*)`),
		pColon:            regexp.MustCompile(`^(.*?\S|^)\s*(:)\s*(\S.*|$)`),
		pEllipsis:         regexp.MustCompile(`^(.*?\S|^)\s*(\.\.\.)\s*(\S.*|$)`),
		pOpDot:            regexp.MustCompile(`^(.*?\S|^)\s*(\.)\s*(\+|\-|\*|/|\^)\s*(=)\s*(\S.*|$)`),
		pPowDot:           regexp.MustCompile(`^(.*?\S|^)\s*(\.)\s*(\^)\s*(\S.*|$)`),
		pPow:              regexp.MustCompile(`^(.*?\S|^)\s*(\^)\s*(\S.*|$)`),
		pOpComb:           regexp.MustCompile(`^(.*?\S|^)\s*(\.|\+|\-|\*|\\|/|=|<|>|\||\&|!|~|\^)\s*(<|>|=|\+|\-|\*|/|\&|\|)\s*(\S.*|$)`),
		pNot:              regexp.MustCompile(`^(.*?\S|^)\s*(!|~)\s*(\S.*|$)`),
		pOp:               regexp.MustCompile(`^(.*?\S|^)\s*(\+|\-|\*|\\|/|=|!|~|<|>|\||\&)\s*(\S.*|$)`),
		pFunc:             regexp.MustCompile(`^(.*?\w)(\()\s*(\S.*|$)`),
		pOpen:             regexp.MustCompile(`^(.*?)(\(|\[|\{)\s*(\S.*|$)`),
		pClose:            regexp.MustCompile(`^(.*?\S|^)\s*(\)|\]|\})(.*|$)`),
		pComma:            regexp.MustCompile(`^(.*?\S|^)\s*(,|;)\s*(\S.*|$)`),
		pMultiWS:          regexp.MustCompile(`^(.*?\S|^)(\s{2,})(\S.*|$)`),
		initialIndent:     regexp.MustCompile(`^(\s*)(.*)$`),
	}

	return formatter, nil
}

// FormatFile formats the requested range of the provided file and writes the
// result to the supplied writer. A filename of "-" reads from stdin.
func (f *Formatter) FormatFile(filename string, w io.Writer) error {
	var (
		reader io.Reader
		closer io.Closer
		err    error
	)

	if filename == "-" {
		reader = os.Stdin
	} else {
		file, openErr := os.Open(filename)
		if openErr != nil {
			return openErr
		}
		reader = file
		closer = file
	}

	if closer != nil {
		defer closer.Close()
	}

	lines, err := readLines(reader)
	if err != nil {
		return err
	}

	formatted, err := f.FormatLines(lines)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(w)
	for _, line := range formatted {
		if _, writeErr := fmt.Fprintln(writer, line); writeErr != nil {
			return writeErr
		}
	}
	return writer.Flush()
}

// FormatLines formats the configured slice of lines according to the supplied
// options.
func (f *Formatter) FormatLines(lines []string) ([]string, error) {
	start := f.opts.StartLine
	if start < 1 {
		start = 1
	}
	end := f.opts.EndLine

	startIdx := start - 1
	if startIdx > len(lines) {
		startIdx = len(lines)
	}

	endIdx := len(lines)
	if end > 0 && end <= len(lines) {
		endIdx = end
	}
	if endIdx < startIdx {
		endIdx = startIdx
	}

	if startIdx == endIdx {
		copied := append([]string{}, lines...)
		return copied, nil
	}

	segment := append([]string{}, lines[startIdx:endIdx]...)
	if len(segment) == 0 {
		segment = []string{""}
	}

	f.resetState()

	match := f.initialIndent.FindStringSubmatch(segment[0])
	if len(match) == 3 {
		f.ilvl = len(match[1]) / f.iwidth
		segment[0] = match[2]
	}

	var output []string
	blank := true

	for _, rawLine := range segment {
		if len(strings.TrimSpace(rawLine)) == 0 {
			if !blank {
				output = append(output, "")
				blank = true
			}
			continue
		}

		offset, line := f.formatLine(rawLine)
		f.ilvl += offset
		if f.ilvl < 0 {
			f.ilvl = 0
		}

		if f.separateBlock && offset > 0 && !blank && f.isLineComment == 0 {
			output = append(output, "")
		}

		output = append(output, strings.TrimRight(line, " \t\r\n"))

		if f.separateBlock && offset < 0 {
			output = append(output, "")
			blank = true
		} else {
			blank = false
		}
	}

	if endIdx == len(lines) {
		for len(output) > 0 && output[len(output)-1] == "" {
			output = output[:len(output)-1]
		}

		if len(output) == 0 {
			output = []string{""}
		}
	} else if len(output) == 0 {
		output = []string{""}
	}

	result := make([]string, 0, len(lines[:startIdx])+len(output)+len(lines[endIdx:]))
	result = append(result, lines[:startIdx]...)
	result = append(result, output...)
	result = append(result, lines[endIdx:]...)

	return result, nil
}

func (f *Formatter) resetState() {
	f.ilvl = 0
	f.istep = f.istep[:0]
	f.fstep = f.fstep[:0]
	f.matrix = 0
	f.cell = 0
	f.isBlockComment = 0
	f.isLineComment = 0
	f.longLine = 0
	f.continueLine = 0
	f.isComment = 0
	f.ignoreLines = 0
}

func (f *Formatter) formatLine(line string) (int, string) {
	if f.ignoreLines > 0 {
		f.ignoreLines--
		return 0, f.indent(0) + strings.TrimSpace(line)
	}

	if f.lineComment.MatchString(line) {
		f.isLineComment = 2
	} else {
		if f.isLineComment > 0 {
			f.isLineComment--
		}
	}

	switch {
	case f.blockCommentOpen.MatchString(line):
		f.isBlockComment = blockCommentSentinel
	case f.blockCommentClose.MatchString(line):
		f.isBlockComment = 1
	default:
		if f.isBlockComment > 0 {
			f.isBlockComment--
		}
	}

	f.isComment = 0
	stripped := f.cleanLineFromStringsAndComments(line)
	ellipsisInComment := f.isLineComment == 2 || f.isBlockComment > 0

	if f.blockClose.MatchString(stripped) || ellipsisInComment {
		f.continueLine = 0
	} else {
		f.continueLine = f.longLine
	}

	if f.ellipsis.MatchString(stripped) && !ellipsisInComment {
		f.longLine = 1
	} else {
		f.longLine = 0
	}

	if f.isBlockComment > 0 {
		return 0, strings.TrimRight(line, " \t\r\n")
	}

	if f.isLineComment == 2 {
		if m := f.ignoreCommand.FindStringSubmatch(line); len(m) == 2 {
			if m[1] != "" {
				if v, err := strconv.Atoi(m[1]); err == nil {
					if v > 1 {
						f.ignoreLines = v
					} else {
						f.ignoreLines = 1
					}
				}
			} else {
				f.ignoreLines = 1
			}
		}
		return 0, f.indent(0) + strings.TrimSpace(line)
	}

	if f.ctrlIgnore.MatchString(line) {
		return 0, f.indent(0) + strings.TrimSpace(line)
	}

	prevMatrix := f.matrix
	if diff := f.multilineMatrix(line); diff != 0 || prevMatrix != 0 {
		return 0, f.indent(prevMatrix) + strings.TrimSpace(f.format(line))
	}

	prevCell := f.cell
	if diff := f.cellArray(line); diff != 0 || prevCell != 0 {
		return 0, f.indent(prevCell) + strings.TrimSpace(f.format(line))
	}

	if m := f.ctrl1Line.FindStringSubmatch(line); len(m) == 7 {
		return 0, f.indent(0) + m[2] + " " + strings.TrimSpace(f.format(m[3])) + " " + m[4] + " " + strings.TrimSpace(f.format(m[6]))
	}

	if m := f.fcnStart.FindStringSubmatch(line); len(m) == 4 {
		offset := f.indentMode
		f.fstep = append(f.fstep, 1)
		if f.indentMode == -1 {
			if len(f.fstep) > 1 {
				offset = 1
			} else {
				offset = 0
			}
		}
		return offset, f.indent(0) + m[2] + " " + strings.TrimSpace(f.format(m[3]))
	}

	if m := f.ctrlStart.FindStringSubmatch(line); len(m) == 4 {
		f.istep = append(f.istep, 1)
		return 1, f.indent(0) + m[2] + " " + strings.TrimSpace(f.format(m[3]))
	}

	if m := f.ctrlStartSwitch.FindStringSubmatch(line); len(m) == 4 {
		f.istep = append(f.istep, 2)
		return 2, f.indent(0) + m[2] + " " + strings.TrimSpace(f.format(m[3]))
	}

	if m := f.ctrlCont.FindStringSubmatch(line); len(m) == 4 {
		return 0, f.indent(-f.iwidth) + m[2] + " " + strings.TrimSpace(f.format(m[3]))
	}

	if m := f.ctrlEnd.FindStringSubmatch(line); len(m) == 5 {
		step := 0
		indentExtra := 0
		if l := len(f.istep); l > 0 {
			step = f.istep[l-1]
			f.istep = f.istep[:l-1]
			indentExtra = -step * f.iwidth
		} else if l := len(f.fstep); l > 0 {
			step = f.fstep[l-1]
			f.fstep = f.fstep[:l-1]
			indentExtra = -step * f.iwidth
		} else if f.ilvl > 0 {
			// When the formatter is asked to operate on a partial selection that
			// only contains closing statements (e.g. one or more "end" lines),
			// we may not have matching openers recorded on the stack. In that
			// case we still need to reduce the indent depth for subsequent lines
			// while keeping the current line aligned with its existing indent.
			step = 1
			indentExtra = 0
		}
		return -step, f.indent(indentExtra) + m[2] + " " + strings.TrimSpace(f.format(m[4]))
	}

	return 0, f.indent(0) + strings.TrimSpace(f.format(line))
}

func (f *Formatter) cellIndent(line, open, close string, indent int) (int, int) {
	pattern := regexp.MustCompile(fmt.Sprintf(`(\s*)((\S.*)?)(%s.*$)`, regexp.QuoteMeta(open)))
	cleaned := f.cleanLineFromStringsAndComments(line)
	openCount := strings.Count(cleaned, open) - strings.Count(cleaned, close)

	if openCount > 0 {
		if m := pattern.FindStringSubmatch(cleaned); len(m) >= 3 {
			n := len(m[2])
			if f.matrixIndent {
				indent = n + 1
			} else {
				indent = f.iwidth
			}
		}
	} else if openCount < 0 {
		indent = 0
	}

	return openCount, indent
}

func (f *Formatter) multilineMatrix(line string) int {
	diff, indent := f.cellIndent(line, "[", "]", f.matrix)
	f.matrix = indent
	return diff
}

func (f *Formatter) cellArray(line string) int {
	diff, indent := f.cellIndent(line, "{", "}", f.cell)
	f.cell = indent
	return diff
}

func (f *Formatter) cleanLineFromStringsAndComments(line string) string {
	left, _, right, ok := f.extractStringOrComment(line)
	if ok {
		return f.cleanLineFromStringsAndComments(left) + " " + f.cleanLineFromStringsAndComments(right)
	}
	return line
}

func (f *Formatter) extractStringOrComment(part string) (string, string, string, bool) {
	m := f.pString.FindStringSubmatch(part)
	m2 := f.pStringDQ.FindStringSubmatch(part)
	if m2 != nil && (m == nil || len(m[2]) < len(m2[2])) {
		m = m2
	}
	if m != nil {
		return m[1], m[2], m[4], true
	}

	if m = f.pComment.FindStringSubmatch(part); m != nil {
		f.isComment = 1
		return m[1] + " ", m[2], "", true
	}

	return "", "", "", false
}

func (f *Formatter) extract(part string) (string, string, string, bool) {
	if f.pBlank.MatchString(part) {
		return "", " ", "", true
	}

	if left, mid, right, ok := f.extractStringOrComment(part); ok {
		return left, mid, right, true
	}

	if m := f.pNumSci.FindStringSubmatch(part); m != nil {
		return m[1] + m[2], m[3], m[4] + m[5], true
	}

	if m := f.pNumRational.FindStringSubmatch(part); m != nil {
		return m[1] + m[2], m[3], m[4] + m[5], true
	}

	if m := f.pIncrement.FindStringSubmatch(part); m != nil {
		return m[1], m[2] + m[3], m[4], true
	}

	if m := f.pSign.FindStringSubmatch(part); m != nil {
		return m[1], m[2], m[3], true
	}

	if m := f.pColon.FindStringSubmatch(part); m != nil {
		return m[1], m[2], m[3], true
	}

	if m := f.pOpDot.FindStringSubmatch(part); m != nil {
		sep := ""
		if f.operatorSep > 0 {
			sep = " "
		}
		return m[1] + sep, m[2] + m[3] + m[4], sep + m[5], true
	}

	if m := f.pPowDot.FindStringSubmatch(part); m != nil {
		sep := ""
		if f.operatorSep > 0.5 {
			sep = " "
		}
		return m[1] + sep, m[2] + m[3], sep + m[4], true
	}

	if m := f.pPow.FindStringSubmatch(part); m != nil {
		sep := ""
		if f.operatorSep > 0.5 {
			sep = " "
		}
		return m[1] + sep, m[2], sep + m[3], true
	}

	if m := f.pOpComb.FindStringSubmatch(part); m != nil {
		sep := ""
		if f.operatorSep > 0 {
			sep = " "
		}
		return m[1] + sep, m[2] + m[3], sep + m[4], true
	}

	if m := f.pNot.FindStringSubmatch(part); m != nil {
		return m[1] + " ", m[2], m[3], true
	}

	if m := f.pOp.FindStringSubmatch(part); m != nil {
		sep := ""
		if f.operatorSep > 0 {
			sep = " "
		}
		return m[1] + sep, m[2], sep + m[3], true
	}

	if m := f.pFunc.FindStringSubmatch(part); m != nil {
		return m[1], m[2], m[3], true
	}

	if m := f.pOpen.FindStringSubmatch(part); m != nil {
		return m[1], m[2], m[3], true
	}

	if m := f.pClose.FindStringSubmatch(part); m != nil {
		return m[1], m[2], m[3], true
	}

	if m := f.pComma.FindStringSubmatch(part); m != nil {
		return m[1], m[2], " " + m[3], true
	}

	if m := f.pEllipsis.FindStringSubmatch(part); m != nil {
		return m[1] + " ", m[2], " " + m[3], true
	}

	if m := f.pMultiWS.FindStringSubmatch(part); m != nil {
		return m[1], " ", m[3], true
	}

	return "", "", "", false
}

func (f *Formatter) format(part string) string {
	left, mid, right, ok := f.extract(part)
	if !ok {
		return part
	}
	return f.format(left) + mid + f.format(right)
}

func (f *Formatter) indent(extra int) string {
	width := (f.ilvl + f.continueLine) * f.iwidth
	width += extra
	if width < 0 {
		width = 0
	}
	if width == 0 {
		return ""
	}
	return strings.Repeat(" ", width)
}

func readLines(r io.Reader) ([]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	lines := strings.Split(content, "\n")

	// strings.Split always adds an empty string after a trailing delimiter.
	// Remove it to avoid outputting an extra newline, except for empty files.
	if len(lines) > 1 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return lines, nil
}
