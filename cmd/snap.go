package cmd

import (
	"encoding/csv"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
)

// Global variables used by the application.
var (
	app             *tview.Application // TUI application instance
	table           *tview.Table       // Table to display snapshots
	selectedConfig  string             // Currently selected Snapper configuration
	snapshotName    string             // User-entered snapshot name
	snapshotUpdated bool               // Flag indicating whether the snapshot table needs updating
)

// Initialize the 'snap' subcommand.
func init() {
	snapshotUpdated = true
	rootCmd.AddCommand(snapCmd)
}

// Function to cycle focus between UI elements.
func cycleFocus(elements []tview.Primitive, reverse bool) {
	for i, element := range elements {
		if !element.HasFocus() {
			continue
		}

		if reverse {
			i = (i - 1 + len(elements)) % len(elements)
		} else {
			i = (i + 1) % len(elements)
		}

		app.SetFocus(elements[i])
		return
	}
}

// Function to retrieve Snapper configurations.
func getSnapperConfigurations() ([]string, error) {
	cmd := exec.Command("snapper", "--csvout", "list-configs", "--columns", "config")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var configurations []string
	configReader := csv.NewReader(strings.NewReader(string(output)))
	records, err := configReader.ReadAll()
	if err != nil {
		return nil, err
	}

	for row, data := range records {
		if row == 0 {
			continue
		}
		configurations = append(configurations, data[0])
	}
	return configurations, nil
}

// Function to retrieve snapshots for a given configuration.
func getSnapshotsForConfig(config string) ([][]string, error) {
	cmd := exec.Command("snapper", "--csvout", "-c", config, "list", "--columns", "config,number,date,description")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	snapshots := csv.NewReader(strings.NewReader(string(output)))
	records, err := snapshots.ReadAll()
	if err != nil {
		return nil, err
	}
	return records, nil
}

// Function to create a new Snapper snapshot.
func createSnapshot(config, description string) error {
	cmd := exec.Command("snapper", "-c", config, "create", "-d", description)
	err := cmd.Run()
	if err != nil {
		return err
	}
	snapshotUpdated = true
	return nil
}

// Function to delete a Snapper snapshot.
func deleteSnapshot(config, id string) error {
	cmd := exec.Command("snapper", "-c", config, "delete", id)
	err := cmd.Run()
	if err != nil {
		return err
	}
	snapshotUpdated = true
	return nil
}

// Function to revert a Snapper snapshot.
func revertSnapshot(config, id string) error {
	cmd := exec.Command("snapper", "-c", config, "undochange", id+"..0")
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

// Function to redraw the snapshot table if needed.
func updateTableIfNeeded(config string) {
	// Continuously check for updates to the snapshot table
	for {
		// Check if the snapshot table needs updating
		if snapshotUpdated {
			// Retrieve the latest snapshot information for the selected configuration
			snapshots, err := getSnapshotsForConfig(config)
			if err != nil {
				log.Fatal("Error getting snapshots:", err)
			}

			// Update the snapshot table on the main thread
			app.QueueUpdateDraw(func() {
				// Clear the existing table contents
				table.Clear()

				// Create the table header with appropriate styling
				header := []string{"Delete", "Rollback", "Config", "ID", "Timestamp", "Description"}
				headerColor := tcell.ColorYellow
				for col, title := range header {
					cell := tview.NewTableCell(title).SetAlign(tview.AlignLeft).SetSelectable(false).SetTextColor(headerColor)
					table.SetCell(0, col, cell)
				}

				// Populate the table with snapshot data
				for row, data := range snapshots {
					// Skip the header row
					if row == 0 {
						continue
					}

					// Extract snapshot data from the retrieved information
					conf := data[0]
					id := data[1]
					timestamp := data[2]
					description := data[3]

					table.SetCell(row, 0, tview.NewTableCell("Delete").SetAlign(tview.AlignLeft).SetSelectable(true))
					table.SetCell(row, 1, tview.NewTableCell("Rollback").SetAlign(tview.AlignLeft).SetSelectable(true))
					table.SetCell(row, 2, tview.NewTableCell(conf).SetAlign(tview.AlignLeft).SetSelectable(false))
					table.SetCell(row, 3, tview.NewTableCell(id).SetAlign(tview.AlignLeft).SetSelectable(false))
					table.SetCell(row, 4, tview.NewTableCell(timestamp).SetAlign(tview.AlignLeft).SetSelectable(false))
					table.SetCell(row, 5, tview.NewTableCell(description).SetAlign(tview.AlignRight).SetSelectable(false))
				}
			})
			snapshotUpdated = false
			table.Select(0, 0).SetDoneFunc(func(key tcell.Key) {
				if key == tcell.KeyEnter {
					table.SetSelectable(true, true)
				}
			}).SetSelectedFunc(func(row, column int) {
				table.GetCell(row, column).SetTextColor(tcell.ColorRed)
				if column == 1 {
					err := revertSnapshot(config, snapshots[row][1])
					if err != nil {
						log.Fatal("Error reverting snapshot:", err)
					}
				}
				if column == 0 {
					err := deleteSnapshot(config, snapshots[row][1])
					if err != nil {
						log.Fatal("Error deleting snapshot:", err)
					}
				}
			})
		}
		time.Sleep(time.Second)
	}
}

// Define the 'snap' subcommand.
var snapCmd = &cobra.Command{
	Use:   "snap",
	Short: "Manage Snapper snapshots",
	Args:  cobra.MaximumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		// Check if the script is run as root
		if !isRoot() {
			log.Fatal("You need to run this script as root")
		}

		// Retrieve available Snapper configurations
		snapperConfigs, err := getSnapperConfigurations()
		if err != nil {
			log.Fatal("Error getting Snapper configurations:", err)
		}

		// Create a TUI application for managing snapshots
		app = tview.NewApplication()

		// Create a table to display snapshot information
		table = tview.NewTable()

		// Create a form for user input (configuration selection, snapshot name, and actions)
		form := tview.NewForm().
			AddDropDown("Configuration: ", snapperConfigs, 0, func(optionText string, optionIndex int) {
				// Update the selected configuration and trigger snapshot table update
				selectedConfig = snapperConfigs[optionIndex]
				snapshotUpdated = true
				go updateTableIfNeeded(selectedConfig)
			}).
			AddInputField("Snapshot Name: ", "", 12, nil, func(text string) { snapshotName = text }).
			AddButton("Create Snapshot", func() { createSnapshot(selectedConfig, snapshotName) }).
			AddButton("Quit", func() { app.Stop() })

		// Create a flexible layout for the UI
		flex := tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(form, 0, 2, true).
			AddItem(table, 0, 4, false)

		// Define UI elements for cycling focus
		inputs := []tview.Primitive{
			table,
			form,
		}

		// Set borders and titles for UI elements
		table.SetBorder(true).SetTitle("Snapshots").SetTitleAlign(tview.AlignLeft)
		form.SetBorder(true).SetTitle("Input").SetTitleAlign(tview.AlignLeft)

		// Handle input events for cycling focus
		flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyPgUp {
				cycleFocus(inputs, false)
			} else if event.Key() == tcell.KeyPgDn {
				cycleFocus(inputs, true)
			}
			return event
		})

		// Run the TUI application with mouse support
		if err := app.SetRoot(flex, true).EnableMouse(true).Run(); err != nil {
			panic(err)
		}
	},
}
