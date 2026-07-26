package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/brianvoe/gofakeit/v7"
)

type column struct {
	Name string
	Type string
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	var total int = 1
	var columns []column
	var outputPath string
	parseErr := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-n":
			i++
			if i >= len(args) {
				fmt.Fprintln(os.Stderr, "error: -n requires a value")
				parseErr = true
				continue
			}
			n, err := strconv.Atoi(args[i])
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: invalid -n value %q\n", args[i])
				parseErr = true
			}
			total = n
		case "-h", "-H", "--header":
			i++
			if i >= len(args) {
				fmt.Fprintln(os.Stderr, "error: -h/--header requires a value")
				parseErr = true
				continue
			}
			col, ok := parseColumn(args[i])
			if ok {
				columns = append(columns, col)
			}
		default:
			outputPath = args[i]
		}
	}

	if parseErr {
		os.Exit(1)
	}

	if len(columns) == 0 {
		fmt.Fprintln(os.Stderr, "error: at least one header is required (use -h name:Type)")
		os.Exit(1)
	}

	if total < 1 {
		fmt.Fprintln(os.Stderr, "error: total rows must be at least 1, got:", total)
		os.Exit(1)
	}

	if outputPath == "" {
		outputPath = "output.csv"
	}

	if err := generateCSV(columns, total, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %d rows to %s\n", total, outputPath)
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `gencsv - Generate CSV files with fake data

Usage:
  gencsv -n <total> -h <name:Type> [-h <name:Type> ...] [output.csv]

Options:
  -n <int>        Total rows to generate (default: 1)
  -h, --header    Column definition as name:Type (repeatable)
  output.csv      Output file path (default: output.csv)

Types:
  FirstName, LastName, Email, Phone

Examples:
  gencsv -n 100 -h "email:Email" data.csv
  gencsv -n 50 -h "first:FirstName" -h "last:LastName" -h "email:Email" users.csv`)
}

func parseColumn(raw string) (column, bool) {
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		fmt.Fprintf(os.Stderr, "warning: invalid header format %q (expected name:Type), ignoring\n", raw)
		return column{}, false
	}
	name := strings.TrimSpace(parts[0])
	typ := strings.TrimSpace(parts[1])
	if name == "" || typ == "" {
		fmt.Fprintf(os.Stderr, "warning: empty name or type in %q, ignoring\n", raw)
		return column{}, false
	}
	return column{Name: name, Type: typ}, true
}

func generateCSV(cols []column, total int, outputPath string) error {
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	headerRow := make([]string, len(cols))
	for i, c := range cols {
		headerRow[i] = c.Name
	}
	if err := w.Write(headerRow); err != nil {
		return fmt.Errorf("writing header: %w", err)
	}

	faker := gofakeit.New(0)

	for i := 0; i < total; i++ {
		row := make([]string, len(cols))
		for j, c := range cols {
			row[j] = generateValue(faker, c.Type)
		}
		if err := w.Write(row); err != nil {
			return fmt.Errorf("writing row %d: %w", i, err)
		}
	}

	return nil
}

func generateValue(faker *gofakeit.Faker, typ string) string {
	switch strings.ToLower(typ) {
	case "firstname":
		return faker.FirstName()
	case "lastname":
		return faker.LastName()
	case "email":
		return faker.Email()
	case "phone":
		return faker.Phone()
	default:
		return faker.FirstName()
	}
}
