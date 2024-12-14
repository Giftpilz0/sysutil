package cmd

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	csvFile    string
	ldifFile   string
	mode       string
	changeType string
	attribute  string
	separator  string
)

// Initialize the 'ldiftool' subcommand.
func init() {
	rootCmd.AddCommand(ldifToolCmd)
	ldifToolCmd.Flags().StringVarP(&csvFile, "csv", "c", "default.csv", "Name of the CSV file.")
	ldifToolCmd.Flags().StringVarP(&ldifFile, "ldif", "l", "default.ldif", "Name of the LDIF file.")
	ldifToolCmd.Flags().StringVarP(&mode, "mode", "m", "csv2ldif", "Conversion direction: csv2ldif or ldif2csv.")
	ldifToolCmd.Flags().StringVarP(&changeType, "type", "t", "replace", "LDIF modify change type (csv2ldif mode): add, replace, delete.")
	ldifToolCmd.Flags().StringVarP(&attribute, "attribute", "a", "uid", "LDIF modify attribute name (csv2ldif mode).")
	ldifToolCmd.Flags().StringVarP(&separator, "separator", "s", ";", "CSV field separator (csv2ldif mode).")
}

// Define the 'ldiftool' subcommand.
var ldifToolCmd = &cobra.Command{
	Use:   "ldiftool",
	Args:  cobra.MaximumNArgs(0),
	Short: "Convert between CSV and LDIF file formats.",
	Run: func(cmd *cobra.Command, args []string) {
		switch mode {
		case "csv2ldif":
			if err := convertCsvToLdif(); err != nil {
				fmt.Println("Error:", err)
			}
		case "ldif2csv":
			if err := convertLdifToCsv(); err != nil {
				fmt.Println("Error:", err)
			}
		default:
			fmt.Println("Invalid mode. Use 'csv2ldif' or 'ldif2csv'.")
		}
	},
}

// Convert CSV to LDIF format.
func convertCsvToLdif() error {
	csvFileHandle, err := os.Open(csvFile)
	if err != nil {
		return fmt.Errorf("could not open CSV file: %w", err)
	}
	defer csvFileHandle.Close()

	csvData, err := readCsv(csvFileHandle, rune(separator[0]))
	if err != nil {
		return fmt.Errorf("could not read CSV data: %w", err)
	}

	if err := writeLdif(ldifFile, csvData, attribute, changeType); err != nil {
		return fmt.Errorf("could not write LDIF file: %w", err)
	}

	return nil
}

// Convert LDIF to CSV format.
func convertLdifToCsv() error {
	ldifFileHandle, err := os.Open(ldifFile)
	if err != nil {
		return fmt.Errorf("could not open LDIF file: %w", err)
	}
	defer ldifFileHandle.Close()

	ldifEntries, err := readLdif(ldifFileHandle)
	if err != nil {
		return fmt.Errorf("could not read LDIF data: %w", err)
	}

	if err := writeCsv(csvFile, ldifEntries, rune(separator[0])); err != nil {
		return fmt.Errorf("could not write CSV file: %w", err)
	}

	return nil
}

// Write data to an LDIF file from CSV.
func writeLdif(ldifFile string, csvData [][]string, attributeName, changeType string) error {
	dnIndex, err := getColumnIndex(csvData, "dn")
	if err != nil {
		return err
	}
	attributeIndex, err := getColumnIndex(csvData, attributeName)
	if err != nil {
		return err
	}

	file, err := os.Create(ldifFile)
	if err != nil {
		return fmt.Errorf("could not create LDIF file: %w", err)
	}
	defer file.Close()

	for _, row := range csvData[1:] {
		if row[attributeIndex] == "" {
			continue
		}

		_, err = file.WriteString(fmt.Sprintf("dn: %s\n", row[dnIndex]))
		if err != nil {
			return err
		}

		_, err = file.WriteString("changetype: modify\n")
		if err != nil {
			return err
		}

		_, err = file.WriteString(fmt.Sprintf("%s: %s\n", changeType, attributeName))
		if err != nil {
			return err
		}

		if changeType != "delete" {
			_, err = file.WriteString(fmt.Sprintf("%s: %s\n", attributeName, row[attributeIndex]))
			if err != nil {
				return err
			}
		}

		_, err = file.WriteString("\n")
		if err != nil {
			return err
		}
	}

	return nil
}

// Get the index of a specific column in the CSV data.
func getColumnIndex(csvData [][]string, columnName string) (int, error) {
	header := csvData[0]
	for i, col := range header {
		if col == columnName {
			return i, nil
		}
	}
	return -1, fmt.Errorf("column '%s' not found", columnName)
}

// Read CSV file and return its content as a 2D slice of strings.
func readCsv(csvFile *os.File, separator rune) ([][]string, error) {
	csvReader := csv.NewReader(csvFile)
	csvReader.Comma = separator

	csvData, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("could not read CSV: %w", err)
	}

	return csvData, nil
}

// Read LDIF file and return entries as a slice of maps.
func readLdif(ldifFile *os.File) ([]map[string]string, error) {
	var ldifEntries []map[string]string
	var currentEntry map[string]string

	scanner := bufio.NewScanner(ldifFile)

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "dn:") {
			if currentEntry != nil {
				ldifEntries = append(ldifEntries, currentEntry)
			}
			currentEntry = map[string]string{"dn": strings.TrimSpace(line[3:])}
		} else {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				attr := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				if existingValue, exists := currentEntry[attr]; exists {
					currentEntry[attr] = existingValue + "\n" + value
				} else {
					currentEntry[attr] = value
				}
			}
		}
	}

	if currentEntry != nil {
		ldifEntries = append(ldifEntries, currentEntry)
	}

	return ldifEntries, nil
}

// Write data to a CSV file from LDIF.
func writeCsv(csvFile string, ldifEntries []map[string]string, separator rune) error {
	file, err := os.Create(csvFile)
	if err != nil {
		return fmt.Errorf("could not create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Comma = separator
	defer writer.Flush()

	var header []string
	for _, entry := range ldifEntries {
		for key := range entry {
			if !stringInSlice(key, header) {
				header = append(header, key)
			}
		}
	}

	if err := writer.Write(header); err != nil {
		return fmt.Errorf("could not write CSV header: %w", err)
	}

	for _, entry := range ldifEntries {
		var row []string
		for _, key := range header {
			value, exists := entry[key]
			if exists {
				if strings.Contains(value, "\n") {
					values := strings.Split(value, "\n")
					var encapsulatedValues []string
					for _, val := range values {
						encapsulatedValues = append(encapsulatedValues, fmt.Sprintf("[%s]", val))
					}
					row = append(row, strings.Join(encapsulatedValues, ""))
				} else {
					row = append(row, value)
				}
			} else {
				row = append(row, "")
			}
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("could not write CSV row: %w", err)
		}
	}

	return nil
}

// Check if a string exists in a slice.
func stringInSlice(str string, slice []string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}
