package snapper

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type Config struct {
	Name      string
	Subvolume string
}

type Snapshot struct {
	Number      int
	Type        string
	PreNumber   string
	Date        string
	User        string
	Cleanup     string
	Description string
}

var Bin = "snapper"

func Configs(ctx context.Context) ([]Config, error) {
	records, err := runCSV(ctx, "list-configs", "--columns", "config,subvolume")
	if err != nil {
		return nil, err
	}

	configs := make([]Config, 0, len(records))
	for _, r := range records {
		configs = append(configs, Config{Name: r["config"], Subvolume: r["subvolume"]})
	}

	return configs, nil
}

// List omits the pseudo-snapshot 0 ("current").
func List(ctx context.Context, config string) ([]Snapshot, error) {
	records, err := runCSV(ctx, "-c", config, "list", "--columns", "number,type,pre-number,date,user,cleanup,description")
	if err != nil {
		return nil, err
	}

	snaps := make([]Snapshot, 0, len(records))
	for _, r := range records {
		n, err := strconv.Atoi(r["number"])
		if err != nil {
			return nil, fmt.Errorf("unexpected snapshot number %q", r["number"])
		}
		if n == 0 {
			continue
		}

		snaps = append(snaps, Snapshot{
			Number:      n,
			Type:        r["type"],
			PreNumber:   r["pre-number"],
			Date:        r["date"],
			User:        r["user"],
			Cleanup:     r["cleanup"],
			Description: r["description"],
		})
	}

	return snaps, nil
}

func Create(ctx context.Context, config, description string) (int, error) {
	out, err := run(ctx, "-c", config, "create", "--print-number", "--description", description)
	if err != nil {
		return 0, err
	}

	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, fmt.Errorf("unexpected output from snapper create: %q", out)
	}

	return n, nil
}

func Delete(ctx context.Context, config string, numbers ...int) error {
	if len(numbers) == 0 {
		return errors.New("no snapshots given")
	}

	args := []string{"-c", config, "delete"}
	for _, n := range numbers {
		if n <= 0 {
			return fmt.Errorf("invalid snapshot number %d", n)
		}
		args = append(args, strconv.Itoa(n))
	}

	_, err := run(ctx, args...)

	return err
}

func UndoChange(ctx context.Context, config string, number int) error {
	if number <= 0 {
		return fmt.Errorf("invalid snapshot number %d", number)
	}

	_, err := run(ctx, "-c", config, "undochange", fmt.Sprintf("%d..0", number))

	return err
}

func run(ctx context.Context, args ...string) ([]byte, error) {
	var stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, Bin, args...)
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, fmt.Errorf("%s not found in PATH, is snapper installed?", Bin)
		}

		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("snapper %s: %s", subcommand(args), msg)
		}

		return nil, fmt.Errorf("snapper: %w", err)
	}

	return out, nil
}

func subcommand(args []string) string {
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "-c":
			i++
		case !strings.HasPrefix(args[i], "-"):
			return args[i]
		}
	}

	return ""
}

func runCSV(ctx context.Context, args ...string) ([]map[string]string, error) {
	out, err := run(ctx, append([]string{"--csvout"}, args...)...)
	if err != nil {
		return nil, err
	}

	return parseCSV(out)
}

func parseCSV(data []byte) ([]map[string]string, error) {
	rows, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parsing snapper output: %w", err)
	}

	if len(rows) == 0 {
		return nil, nil
	}

	header := rows[0]
	records := make([]map[string]string, 0, len(rows)-1)
	for _, row := range rows[1:] {
		rec := make(map[string]string, len(header))
		for i, col := range header {
			if i < len(row) {
				rec[col] = row[i]
			}
		}
		records = append(records, rec)
	}

	return records, nil
}
