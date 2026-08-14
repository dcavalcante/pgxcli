package app

import (
	"os"
	"path/filepath"
	"testing"
	"unicode/utf8"

	"github.com/balajz/bubbline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompleteLocalMetaCommands(t *testing.T) {
	t.Parallel()

	_, completions := completeMetaCommand(`\c`, 2, 0, 2, maxCompletions)
	require.NotNil(t, completions)
	assert.Contains(t, completionReplacements(completions), `\cd`)

	_, completions = completeMetaCommand(`\e`, 2, 0, 2, maxCompletions)
	require.NotNil(t, completions)
	assert.Contains(t, completionReplacements(completions), `\e`)

	_, completions = completeMetaCommand(`\edi`, 4, 0, 4, maxCompletions)
	require.NotNil(t, completions)
	assert.Contains(t, completionReplacements(completions), `\edit`)

	_, completions = completeMetaCommand(`\i`, 2, 0, 2, maxCompletions)
	require.NotNil(t, completions)
	assert.Contains(t, completionReplacements(completions), `\i`)

	_, completions = completeMetaCommand(`\inc`, 4, 0, 4, maxCompletions)
	require.NotNil(t, completions)
	assert.Contains(t, completionReplacements(completions), `\include`)
}

func TestCompleteIncludeFile(t *testing.T) {
	originalDirectory, err := os.Getwd()
	require.NoError(t, err)
	root := t.TempDir()
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalDirectory))
	})

	require.NoError(t, os.Mkdir(filepath.Join(root, "queries"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "query.sql"), nil, 0o600))
	require.NoError(t, os.Chdir(root))

	t.Run("short command", func(t *testing.T) {
		completions, handled := completeIncludeAtEnd(`\i que`)
		require.True(t, handled)
		require.NotNil(t, completions)
		replacements := completionReplacements(completions)
		assert.Contains(t, replacements, "queries"+string(os.PathSeparator))
		assert.Contains(t, replacements, "query.sql")
	})

	t.Run("long alias", func(t *testing.T) {
		completions, handled := completeIncludeAtEnd(`\include query.s`)
		require.True(t, handled)
		require.NotNil(t, completions)
		assert.Equal(t, []string{"query.sql"}, completionReplacements(completions))
	})
}

func TestCompleteEditFile(t *testing.T) {
	originalDirectory, err := os.Getwd()
	require.NoError(t, err)
	root := t.TempDir()
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalDirectory))
	})

	require.NoError(t, os.Mkdir(filepath.Join(root, "queries"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "queries", "nested.sql"), nil, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "query.sql"), nil, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "query notes.sql"), nil, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".hidden.sql"), nil, 0o600))
	require.NoError(t, os.Chdir(root))
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)

	t.Run("files and directories", func(t *testing.T) {
		completions, handled := completeEditAtEnd(`\e que`)
		require.True(t, handled)
		require.NotNil(t, completions)
		replacements := completionReplacements(completions)
		assert.Contains(t, replacements, "queries"+string(os.PathSeparator))
		assert.Contains(t, replacements, "query.sql")
		assert.Contains(t, replacements, `'query notes.sql'`)
		assert.NotContains(t, replacements, ".hidden.sql")
	})

	t.Run("long alias", func(t *testing.T) {
		completions, handled := completeEditAtEnd(`\edit query.s`)
		require.True(t, handled)
		require.NotNil(t, completions)
		assert.Equal(t, []string{"query.sql"}, completionReplacements(completions))
	})

	t.Run("nested file", func(t *testing.T) {
		completions, handled := completeEditAtEnd(`\e queries/n`)
		require.True(t, handled)
		require.NotNil(t, completions)
		assert.Equal(t, []string{"queries/nested.sql"}, completionReplacements(completions))
	})

	t.Run("quoted file", func(t *testing.T) {
		completions, handled := completeEditAtEnd(`\e "query n`)
		require.True(t, handled)
		require.NotNil(t, completions)
		assert.Equal(t, []string{`"query notes.sql"`}, completionReplacements(completions))
	})

	t.Run("tilde file", func(t *testing.T) {
		completions, handled := completeEditAtEnd(`\e ~/query.s`)
		require.True(t, handled)
		require.NotNil(t, completions)
		assert.Equal(t, []string{"~/query.sql"}, completionReplacements(completions))
	})

	t.Run("explicit hidden file", func(t *testing.T) {
		completions, handled := completeEditAtEnd(`\e .h`)
		require.True(t, handled)
		require.NotNil(t, completions)
		assert.Equal(t, []string{".hidden.sql"}, completionReplacements(completions))
	})
}

func TestCompleteChangeDirectory(t *testing.T) {
	originalDirectory, err := os.Getwd()
	require.NoError(t, err)
	root := t.TempDir()
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalDirectory))
	})

	require.NoError(t, os.Mkdir(filepath.Join(root, "alpha"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(root, "alpha", "nested"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(root, "two words"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(root, ".hidden"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "alpha.sql"), nil, 0o600))
	require.NoError(t, os.Chdir(root))
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)

	t.Run("relative path", func(t *testing.T) {
		completions, handled := completeDirectoryAtEnd(`\cd al`)
		require.True(t, handled)
		require.NotNil(t, completions)
		assert.Equal(t, []string{"alpha" + string(os.PathSeparator)}, completionReplacements(completions))
	})

	t.Run("nested path", func(t *testing.T) {
		completions, handled := completeDirectoryAtEnd(`\cd alpha/n`)
		require.True(t, handled)
		require.NotNil(t, completions)
		assert.Equal(t, []string{"alpha/nested/"}, completionReplacements(completions))
	})

	t.Run("tilde path", func(t *testing.T) {
		completions, handled := completeDirectoryAtEnd(`\cd ~/al`)
		require.True(t, handled)
		require.NotNil(t, completions)
		assert.Equal(t, []string{"~/alpha/"}, completionReplacements(completions))
	})

	t.Run("quoted path", func(t *testing.T) {
		completions, handled := completeDirectoryAtEnd(`\cd "two`)
		require.True(t, handled)
		require.NotNil(t, completions)
		assert.Equal(t, []string{`"two words` + string(os.PathSeparator) + `"`}, completionReplacements(completions))
	})

	t.Run("adds quotes when needed", func(t *testing.T) {
		completions, handled := completeDirectoryAtEnd(`\cd two`)
		require.True(t, handled)
		require.NotNil(t, completions)
		assert.Equal(t, []string{`'two words` + string(os.PathSeparator) + `'`}, completionReplacements(completions))
	})

	t.Run("filters files and hidden directories", func(t *testing.T) {
		completions, handled := completeDirectoryAtEnd(`\cd `)
		require.True(t, handled)
		require.NotNil(t, completions)
		replacements := completionReplacements(completions)
		assert.Contains(t, replacements, "alpha"+string(os.PathSeparator))
		assert.NotContains(t, replacements, "alpha.sql")
		assert.NotContains(t, replacements, ".hidden"+string(os.PathSeparator))
	})

	t.Run("shows explicitly requested hidden directories", func(t *testing.T) {
		completions, handled := completeDirectoryAtEnd(`\cd .`)
		require.True(t, handled)
		require.NotNil(t, completions)
		assert.Contains(t, completionReplacements(completions), ".hidden"+string(os.PathSeparator))
	})
}

func TestCompleteChangeDirectoryIgnoresOtherInput(t *testing.T) {
	t.Parallel()

	testCases := []string{
		"select 1",
		`\clear `,
		`\c database`,
		`\connect database`,
		`\cdx `,
	}

	for _, input := range testCases {
		completions, handled := completeDirectoryAtEnd(input)
		assert.False(t, handled, input)
		assert.Nil(t, completions, input)
	}
}

func completeDirectoryAtEnd(input string) (bubbline.Completions, bool) {
	return completeChangeDirectory(
		[][]rune{[]rune(input)},
		0,
		utf8.RuneCountInString(input),
		maxCompletions,
	)
}

func completeEditAtEnd(input string) (bubbline.Completions, bool) {
	return completeEditFile(
		[][]rune{[]rune(input)},
		0,
		utf8.RuneCountInString(input),
		maxCompletions,
	)
}

func completeIncludeAtEnd(input string) (bubbline.Completions, bool) {
	return completeIncludeFile(
		[][]rune{[]rune(input)},
		0,
		utf8.RuneCountInString(input),
		maxCompletions,
	)
}

func completionReplacements(completions bubbline.Completions) []string {
	var replacements []string
	for category := 0; category < completions.NumCategories(); category++ {
		for entryIndex := 0; entryIndex < completions.NumEntries(category); entryIndex++ {
			entry := completions.Entry(category, entryIndex)
			replacements = append(replacements, completions.Candidate(entry).Replacement())
		}
	}
	return replacements
}
