package campaign

import (
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
