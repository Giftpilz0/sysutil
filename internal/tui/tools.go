package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"charm.land/huh/v2"
	"charm.land/huh/v2/spinner"

	"github.com/giftpilz0/sysutil/internal/bench"
	"github.com/giftpilz0/sysutil/internal/ldif"
	"github.com/giftpilz0/sysutil/internal/netinfo"
)

func PrintIPInfo(w io.Writer, info netinfo.IPInfo) {
	PrintKV(w, "Public IP", []KV{
		{"Address", info.IP},
		{"Hostname", info.Hostname},
		{"Location", strings.Join(nonEmpty(info.City, info.Region, info.Country), ", ")},
		{"Coordinates", info.Location},
		{"Provider", info.Org},
		{"Timezone", info.Timezone},
	})
}

func PrintMACInfo(w io.Writer, mac string, info netinfo.MACInfo) {
	if !info.Found {
		kind := "unknown vendor"
		if info.IsRandom {
			kind = "randomized (locally administered) address"
		}
		PrintKV(w, "MAC "+mac, []KV{{"Vendor", kind}})
		return
	}

	PrintKV(w, "MAC "+mac, []KV{
		{"Prefix", info.MACPrefix},
		{"Vendor", info.Company},
		{"Address", info.Address},
		{"Country", info.Country},
		{"Block", info.BlockType},
	})
}

func PrintBenchResult(w io.Writer, r bench.Result) {
	PrintKV(w, "Matrix × vector benchmark", []KV{
		{"Matrix", fmt.Sprintf("%d × %d", r.Size, r.Size)},
		{"Runs", strconv.Itoa(r.Runs)},
		{"Workers", strconv.Itoa(r.Workers)},
		{"Total time", r.Elapsed.Round(time.Millisecond).String()},
		{"Time per run", r.PerRun.String()},
		{"Throughput", fmt.Sprintf("%.0f MFLOP/s", r.MFLOPS)},
	})
}

func IP(ctx context.Context) error {
	var info netinfo.IPInfo

	err := spinner.New().
		Title("Looking up public IP…").
		Context(ctx).
		ActionWithErr(func(ctx context.Context) (err error) {
			info, err = netinfo.LookupIP(ctx)
			return err
		}).
		Run()
	if err != nil {
		return err
	}

	PrintIPInfo(os.Stdout, info)

	return nil
}

func MAC(ctx context.Context) error {
	const manual = "manual"

	ifaces, _ := netinfo.Interfaces()

	opts := make([]huh.Option[string], 0, len(ifaces)+1)
	for _, iface := range ifaces {
		state := "down"
		if iface.Up {
			state = "up"
		}
		opts = append(opts, huh.NewOption(fmt.Sprintf("%s  %s  (%s)", iface.MAC, iface.Name, state), iface.MAC))
	}
	opts = append(opts, huh.NewOption("Enter a MAC address…", manual))

	var mac string
	if def, err := netinfo.DefaultMAC(); err == nil {
		mac = def.MAC
	}

	var typed string

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Which MAC address?").
				Options(opts...).
				Value(&mac),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("MAC address or vendor prefix").
				Placeholder("00:1a:2b:3c:4d:5e").
				Validate(func(s string) error { _, err := netinfo.NormalizeMAC(s); return err }).
				Value(&typed),
		).WithHideFunc(func() bool { return mac != manual }),
	).RunWithContext(ctx)
	if err != nil {
		return err
	}

	if mac == manual {
		mac = typed
	}

	var info netinfo.MACInfo
	err = spinner.New().
		Title("Looking up " + mac + "…").
		Context(ctx).
		ActionWithErr(func(ctx context.Context) (err error) {
			info, err = netinfo.LookupMAC(ctx, mac)
			return err
		}).
		Run()
	if err != nil {
		return err
	}

	PrintMACInfo(os.Stdout, mac, info)

	return nil
}

func Bench(ctx context.Context) error {
	def := bench.DefaultConfig()
	runs, size, workers := strconv.Itoa(def.Runs), strconv.Itoa(def.Size), strconv.Itoa(def.Workers)

	err := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Matrix size").Description("Dimension of the square matrix").
			Validate(positiveInt).Value(&size),
		huh.NewInput().Title("Runs").Description("Number of multiplications").
			Validate(positiveInt).Value(&runs),
		huh.NewInput().Title("Workers").Description(fmt.Sprintf("Parallel goroutines (this machine has %d CPUs)", runtime.NumCPU())).
			Validate(positiveInt).Value(&workers),
	)).RunWithContext(ctx)
	if err != nil {
		return err
	}

	cfg := bench.Config{}
	cfg.Runs, _ = strconv.Atoi(runs)
	cfg.Size, _ = strconv.Atoi(size)
	cfg.Workers, _ = strconv.Atoi(workers)

	var res bench.Result
	err = spinner.New().
		Title(fmt.Sprintf("Running %d multiplications of a %d×%d matrix…", cfg.Runs, cfg.Size, cfg.Size)).
		Context(ctx).
		ActionWithErr(func(ctx context.Context) (err error) {
			res, err = bench.Run(ctx, cfg, nil)
			return err
		}).
		Run()
	if err != nil {
		return err
	}

	PrintBenchResult(os.Stdout, res)

	return nil
}

func LDIF(ctx context.Context) error {
	var (
		dir        = ldif.ToLDIF
		input      string
		output     string
		sepStr     = ";"
		attribute  string
		changeType = "replace"
	)

	// Attribute suggestions are refreshed whenever the input file or separator changes.
	headerBinding := []*string{&input, &sepStr}

	header := func() []string {
		sep, err := ldif.ParseSeparator(sepStr)
		if err != nil {
			return nil
		}
		return slices.DeleteFunc(ldif.CSVHeader(input, sep), func(h string) bool { return strings.EqualFold(h, "dn") })
	}

	inputExt := func() []string {
		if dir == ldif.ToLDIF {
			return []string{".csv", ".txt"}
		}
		return []string{".ldif", ".ldf"}
	}

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[ldif.Direction]().
				Title("Conversion").
				Options(
					huh.NewOption("CSV → LDIF  (generate modify records)", ldif.ToLDIF),
					huh.NewOption("LDIF → CSV  (export entries)", ldif.ToCSV),
				).
				Value(&dir),
		),
		huh.NewGroup(
			huh.NewInput().
				TitleFunc(func() string {
					if dir == ldif.ToLDIF {
						return "CSV input file"
					}
					return "LDIF input file"
				}, &dir).
				Description("tab completes files in the current directory").
				SuggestionsFunc(func() []string { return filesWithExt(inputExt()...) }, &dir).
				Validate(existingFile).
				Value(&input),
			huh.NewInput().
				Title("CSV separator").
				Description(`single character, or "tab"`).
				Validate(func(s string) error { _, err := ldif.ParseSeparator(s); return err }).
				Value(&sepStr),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Attribute to modify").
				DescriptionFunc(func() string {
					if h := header(); len(h) > 0 {
						return "columns: " + strings.Join(h, ", ")
					}
					return "CSV column / LDAP attribute"
				}, headerBinding).
				SuggestionsFunc(header, headerBinding).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errors.New("attribute is required")
					}
					return nil
				}).
				Value(&attribute),
			huh.NewSelect[string]().
				Title("Change type").
				Options(huh.NewOptions(ldif.ChangeTypes...)...).
				Value(&changeType),
		).WithHideFunc(func() bool { return dir != ldif.ToLDIF }),
		huh.NewGroup(
			huh.NewInput().
				Title("Output file").
				DescriptionFunc(func() string {
					return "leave empty for " + ldif.DefaultOutput(input, dir) + `, "-" for stdout`
				}, &input).
				Value(&output),
		),
	).RunWithContext(ctx)
	if err != nil {
		return err
	}

	if output == "" {
		output = ldif.DefaultOutput(input, dir)
	}

	if output != "-" {
		if _, err := os.Stat(output); err == nil {
			overwrite := false
			err := huh.NewConfirm().
				Title(output + " already exists. Overwrite?").
				Affirmative("Overwrite").
				Negative("Cancel").
				Value(&overwrite).
				Run()
			if err != nil {
				return err
			}
			if !overwrite {
				return huh.ErrUserAborted
			}
		}
	}

	sep, _ := ldif.ParseSeparator(sepStr)

	n, err := ldif.ConvertFile(dir, input, output, sep, ldif.Modify{Attribute: strings.TrimSpace(attribute), ChangeType: changeType})
	if err != nil {
		return err
	}

	if output != "-" {
		PrintSuccess(os.Stdout, "Wrote %d records to %s", n, output)
	}

	return nil
}

func positiveInt(s string) error {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n <= 0 {
		return errors.New("enter a positive number")
	}

	return nil
}

func existingFile(path string) error {
	if path == "" {
		return errors.New("file is required")
	}

	st, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("cannot open %s", path)
	}
	if st.IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}

	return nil
}

func filesWithExt(exts ...string) []string {
	entries, err := os.ReadDir(".")
	if err != nil {
		return nil
	}

	var out []string
	for _, e := range entries {
		if !e.IsDir() && slices.Contains(exts, strings.ToLower(filepath.Ext(e.Name()))) {
			out = append(out, e.Name())
		}
	}

	return out
}

func nonEmpty(values ...string) []string {
	return slices.DeleteFunc(values, func(s string) bool { return s == "" })
}
