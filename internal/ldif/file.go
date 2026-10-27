package ldif

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Direction string

const (
	ToLDIF Direction = "csv2ldif"
	ToCSV  Direction = "ldif2csv"
)

func DefaultOutput(input string, dir Direction) string {
	ext := ".csv"
	if dir == ToLDIF {
		ext = ".ldif"
	}

	if input == "" || input == "-" {
		return "-"
	}

	return strings.TrimSuffix(input, filepath.Ext(input)) + ext
}

// ConvertFile accepts "-" for stdin/stdout.
func ConvertFile(dir Direction, input, output string, sep rune, m Modify) (int, error) {
	if input != "-" && output != "-" && sameFile(input, output) {
		return 0, errors.New("input and output must be different files")
	}

	var r io.Reader = os.Stdin
	if input != "-" {
		f, err := os.Open(input)
		if err != nil {
			return 0, err
		}
		defer func() { _ = f.Close() }()
		r = f
	}

	var w io.Writer = os.Stdout
	var outFile *os.File
	if output != "-" {
		f, err := os.Create(output)
		if err != nil {
			return 0, err
		}
		outFile = f
		w = f
	}

	var (
		n   int
		err error
	)
	switch dir {
	case ToLDIF:
		n, err = CSVToLDIF(r, w, sep, m)
	case ToCSV:
		n, err = LDIFToCSV(r, w, sep)
	default:
		err = fmt.Errorf("unknown direction %q", dir)
	}

	if outFile != nil {
		if cerr := outFile.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			_ = os.Remove(output)
		}
	}

	return n, err
}

func CSVHeader(path string, sep rune) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	cr := newCSVReader(f, sep)
	header, err := cr.Read()
	if err != nil {
		return nil
	}

	return header
}

func sameFile(a, b string) bool {
	sa, err := os.Stat(a)
	if err != nil {
		return false
	}

	sb, err := os.Stat(b)
	if err != nil {
		return false
	}

	return os.SameFile(sa, sb)
}
