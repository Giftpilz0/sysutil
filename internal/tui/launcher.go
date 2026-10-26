package tui

import (
	"context"
	"errors"
	"fmt"
	"os"

	"charm.land/huh/v2"
)

type Tool struct {
	Name        string
	Description string
	Run         func(ctx context.Context) error
}

func Launcher(ctx context.Context, tools []Tool) error {
	width := 0
	for _, t := range tools {
		width = max(width, len(t.Name))
	}

	opts := make([]huh.Option[int], 0, len(tools)+1)
	for i, t := range tools {
		opts = append(opts, huh.NewOption(fmt.Sprintf("%-*s  %s", width, t.Name, t.Description), i))
	}
	opts = append(opts, huh.NewOption("quit", -1))

	choice := 0
	for {
		err := huh.NewForm(huh.NewGroup(
			huh.NewSelect[int]().
				Title("sysutil").
				Description("Pick a tool · ↑/↓ to move, / to filter, enter to run, esc to quit").
				Options(opts...).
				Value(&choice),
		)).RunWithContext(ctx)
		if errors.Is(err, huh.ErrUserAborted) || choice < 0 {
			return nil
		}
		if err != nil {
			return err
		}

		if err := tools[choice].Run(ctx); err != nil && !errors.Is(err, huh.ErrUserAborted) {
			PrintError(os.Stderr, err)
		}
		fmt.Println()
	}
}
