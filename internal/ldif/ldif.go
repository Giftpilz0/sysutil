package ldif

import (
	"bufio"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode/utf8"
)

var ChangeTypes = []string{"replace", "add", "delete"}

type Attr struct {
	Name   string
	Values []string
}

type Entry struct {
	DN    string
	Attrs []Attr
}

func (e *Entry) add(name, value string) {
	for i := range e.Attrs {
		if strings.EqualFold(e.Attrs[i].Name, name) {
			e.Attrs[i].Values = append(e.Attrs[i].Values, value)
			return
		}
	}
	e.Attrs = append(e.Attrs, Attr{Name: name, Values: []string{value}})
}

type Modify struct {
	Attribute  string
	ChangeType string
}

// ParseSeparator accepts a single character, "tab" or `\t`.
func ParseSeparator(s string) (rune, error) {
	switch s {
	case "tab", `\t`:
		return '\t', nil
	}

	r, size := utf8.DecodeRuneInString(s)
	if size == 0 || size != len(s) {
		return 0, fmt.Errorf("separator must be a single character, got %q", s)
	}

	if r == '"' || r == '\r' || r == '\n' || r == utf8.RuneError {
		return 0, fmt.Errorf("invalid separator %q", s)
	}

	return r, nil
}

// Parse handles comments, line folding and base64 values; change records are flattened.
func Parse(r io.Reader) ([]Entry, error) {
	var (
		entries []Entry
		current *Entry
	)

	flush := func() {
		if current != nil {
			entries = append(entries, *current)
			current = nil
		}
	}

	handle := func(line string) error {
		if line == "" {
			flush()
			return nil
		}

		if strings.HasPrefix(line, "#") || line == "-" {
			return nil
		}

		name, value, err := parseLine(line)
		if err != nil {
			return err
		}

		if strings.EqualFold(name, "dn") {
			flush()
			current = &Entry{DN: value}
			return nil
		}

		// Lines outside of a record (e.g. "version: 1") are ignored.
		if current != nil {
			current.add(name, value)
		}

		return nil
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	var logical strings.Builder
	pending := false
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSuffix(scanner.Text(), "\r")

		// A line starting with a single space continues the previous line.
		if strings.HasPrefix(line, " ") && pending {
			logical.WriteString(line[1:])
			continue
		}

		if pending {
			if err := handle(logical.String()); err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNo-1, err)
			}
		}

		logical.Reset()
		logical.WriteString(line)
		pending = true
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if pending {
		if err := handle(logical.String()); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNo, err)
		}
	}

	flush()

	return entries, nil
}

func parseLine(line string) (name, value string, err error) {
	name, rest, ok := strings.Cut(line, ":")
	if !ok {
		return "", "", fmt.Errorf("missing ':' in %q", line)
	}

	switch {
	case strings.HasPrefix(rest, ":"):
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(rest[1:]))
		if err != nil {
			return "", "", fmt.Errorf("invalid base64 value for %s: %w", name, err)
		}
		return name, string(decoded), nil
	case strings.HasPrefix(rest, "<"):
		// URL references are kept verbatim.
		return name, strings.TrimSpace(rest), nil
	default:
		return name, strings.TrimLeft(rest, " "), nil
	}
}

// LDIFToCSV writes multi-valued attributes as "[value1][value2]".
func LDIFToCSV(r io.Reader, w io.Writer, sep rune) (int, error) {
	entries, err := Parse(r)
	if err != nil {
		return 0, err
	}

	header := []string{"dn"}
	for _, e := range entries {
		for _, a := range e.Attrs {
			if !slices.ContainsFunc(header, func(h string) bool { return strings.EqualFold(h, a.Name) }) {
				header = append(header, a.Name)
			}
		}
	}

	cw := csv.NewWriter(w)
	cw.Comma = sep

	if err := cw.Write(header); err != nil {
		return 0, err
	}

	for _, e := range entries {
		row := make([]string, len(header))
		row[0] = e.DN
		for _, a := range e.Attrs {
			i := slices.IndexFunc(header, func(h string) bool { return strings.EqualFold(h, a.Name) })
			row[i] = joinValues(a.Values)
		}

		if err := cw.Write(row); err != nil {
			return 0, err
		}
	}

	cw.Flush()

	return len(entries), cw.Error()
}

// CSVToLDIF skips rows with an empty attribute value.
func CSVToLDIF(r io.Reader, w io.Writer, sep rune, m Modify) (int, error) {
	if !slices.Contains(ChangeTypes, m.ChangeType) {
		return 0, fmt.Errorf("invalid change type %q (want one of %s)", m.ChangeType, strings.Join(ChangeTypes, ", "))
	}

	if m.Attribute == "" {
		return 0, errors.New("no attribute given")
	}

	cr := newCSVReader(r, sep)

	header, err := cr.Read()
	if errors.Is(err, io.EOF) {
		return 0, errors.New("CSV file is empty")
	}
	if err != nil {
		return 0, err
	}

	dnIdx, err := columnIndex(header, "dn")
	if err != nil {
		return 0, err
	}

	attrIdx, err := columnIndex(header, m.Attribute)
	if err != nil {
		return 0, err
	}

	bw := bufio.NewWriter(w)
	count := 0

	for {
		row, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return count, err
		}

		if attrIdx >= len(row) || dnIdx >= len(row) || row[attrIdx] == "" || row[dnIdx] == "" {
			continue
		}

		writeLine(bw, "dn", row[dnIdx])
		fmt.Fprintln(bw, "changetype: modify")
		fmt.Fprintf(bw, "%s: %s\n", m.ChangeType, m.Attribute)
		if m.ChangeType != "delete" {
			for _, v := range splitValues(row[attrIdx]) {
				writeLine(bw, m.Attribute, v)
			}
		}
		fmt.Fprint(bw, "-\n\n")

		count++
	}

	return count, bw.Flush()
}

func newCSVReader(r io.Reader, sep rune) *csv.Reader {
	cr := csv.NewReader(r)
	cr.Comma = sep
	cr.FieldsPerRecord = -1

	return cr
}

func columnIndex(header []string, name string) (int, error) {
	for i, col := range header {
		if strings.EqualFold(strings.TrimSpace(col), name) {
			return i, nil
		}
	}

	return -1, fmt.Errorf("column %q not found in CSV header (have: %s)", name, strings.Join(header, ", "))
}

// writeLine base64 encodes values that are not a SAFE-STRING (RFC 2849).
func writeLine(w io.Writer, name, value string) {
	if isSafe(value) {
		fmt.Fprintf(w, "%s: %s\n", name, value)
		return
	}

	fmt.Fprintf(w, "%s:: %s\n", name, base64.StdEncoding.EncodeToString([]byte(value)))
}

func isSafe(s string) bool {
	if s == "" {
		return true
	}

	if c := s[0]; c == ' ' || c == ':' || c == '<' || strings.HasSuffix(s, " ") {
		return false
	}

	for i := 0; i < len(s); i++ {
		if c := s[i]; c == 0 || c == '\n' || c == '\r' || c > 127 {
			return false
		}
	}

	return true
}

func joinValues(values []string) string {
	if len(values) == 1 {
		return values[0]
	}

	var b strings.Builder
	for _, v := range values {
		b.WriteString("[" + v + "]")
	}

	return b.String()
}

func splitValues(s string) []string {
	if !strings.HasPrefix(s, "[") || !strings.HasSuffix(s, "]") {
		return []string{s}
	}

	parts := strings.Split(s[1:len(s)-1], "][")
	for _, p := range parts {
		if strings.ContainsAny(p, "[]") {
			return []string{s}
		}
	}

	return parts
}
