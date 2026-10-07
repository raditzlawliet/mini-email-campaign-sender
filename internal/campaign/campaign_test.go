package campaign

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCSV(t *testing.T) {
	t.Run("valid CSV with name and email", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		csv := "name,email\nAlice,alice@example.com\nBob,bob@example.com"
		recipients, err := ParseCSV(csv)
		require.NoError(err)
		assert.Len(recipients, 2)
		assert.Equal("alice@example.com", recipients[0].Email)
		assert.Equal("Alice", recipients[0].Data["name"])
		assert.Equal("bob@example.com", recipients[1].Email)
		assert.Equal("Bob", recipients[1].Data["name"])
	})

	t.Run("case-insensitive email column", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		csv := "Name,EMAIL\nAlice,alice@example.com"
		recipients, err := ParseCSV(csv)
		require.NoError(err)
		assert.Len(recipients, 1)
		assert.Equal("alice@example.com", recipients[0].Email)
	})

	t.Run("no email column", func(t *testing.T) {
		_, err := ParseCSV("name,age\nAlice,30")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "email")
	})

	t.Run("no data rows", func(t *testing.T) {
		_, err := ParseCSV("name,email")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one data row")
	})

	t.Run("skips rows with empty email", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		csv := "name,email\nAlice,alice@example.com\nBob,\nCharlie,charlie@example.com"
		recipients, err := ParseCSV(csv)
		require.NoError(err)
		assert.Len(recipients, 2)
	})

	t.Run("empty input", func(t *testing.T) {
		_, err := ParseCSV("")
		assert.Error(t, err)
	})

	t.Run("headers with whitespace trimmed", func(t *testing.T) {
		assert := assert.New(t)
		require := require.New(t)

		csv := " name , email \nAlice,alice@example.com"
		recipients, err := ParseCSV(csv)
		require.NoError(err)
		assert.Len(recipients, 1)
		assert.Equal("Alice", recipients[0].Data["name"])
		assert.Equal("alice@example.com", recipients[0].Data["email"])
	})
}

func TestStreamingCSV(t *testing.T) {
	const csvData = "name,email\nAlice,a@x.com\nBob,b@x.com\nCara,c@x.com\n"
	path := filepath.Join(t.TempDir(), "list.csv")
	require.NoError(t, os.WriteFile(path, []byte(csvData), 0o644))

	t.Run("ParseCSVFile reads all rows", func(t *testing.T) {
		assert := assert.New(t)
		recipients, err := ParseCSVFile(path, 0)
		require.NoError(t, err)
		assert.Len(recipients, 3)
		assert.Equal("c@x.com", recipients[2].Email)
		assert.Equal(2, recipients[2].Index)
	})

	t.Run("limit stops reading early", func(t *testing.T) {
		assert := assert.New(t)
		recipients, err := ParseCSVReader(strings.NewReader(csvData), 2)
		require.NoError(t, err)
		assert.Len(recipients, 2)
	})

	t.Run("limit ignores malformed rows after the limit", func(t *testing.T) {
		assert := assert.New(t)
		recipients, err := ParseCSVReader(strings.NewReader("name,email\nA,a@x.com\n\"broken"), 1)
		require.NoError(t, err)
		assert.Len(recipients, 1)
	})

	t.Run("ScanCSVFile returns ordered headers and count", func(t *testing.T) {
		assert := assert.New(t)
		headers, count, err := ScanCSVFile(path)
		require.NoError(t, err)
		assert.Equal([]string{"name", "email"}, headers)
		assert.Equal(3, count)
	})

	t.Run("missing file", func(t *testing.T) {
		assert := assert.New(t)
		_, err := ParseCSVFile(filepath.Join(t.TempDir(), "nope.csv"), 0)
		assert.Error(err)
		_, _, err = ScanCSVFile(filepath.Join(t.TempDir(), "nope.csv"))
		assert.Error(err)
	})

	t.Run("Preview streams from CSVPath", func(t *testing.T) {
		assert := assert.New(t)
		results, err := Preview(CampaignRequest{CSVPath: path, To: "{email}", Subject: "Hi {name}"}, 2)
		require.NoError(t, err)
		require.Len(t, results, 2)
		assert.Equal("Hi Bob", results[1].Subject)
	})
}
