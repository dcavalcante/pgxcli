package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunLocalMatchesClearExactly(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		query string
		match bool
	}{
		{name: "backslash clear", query: `\clear`, match: true},
		{name: "forward slash clear", query: `/clear`, match: false},
		{name: "uppercase", query: `\CLEAR`, match: false},
		{name: "leading whitespace", query: ` \clear`, match: false},
		{name: "trailing whitespace", query: `\clear `, match: false},
		{name: "argument", query: `\clear now`, match: false},
		{name: "connect short form", query: `\c database`, match: false},
		{name: "connect long form", query: `\connect database`, match: false},
		{name: "empty", query: "", match: false},
	}

	cli := &pgxCLI{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			execution, matched := cli.runLocal(context.Background(), testCase.query)
			assert.Equal(t, testCase.match, matched)
			if testCase.match {
				require.NotNil(t, execution.cmd)
				assert.False(t, execution.delegatesExecution)
				assert.Equal(t, tea.ClearScreen(), execution.cmd())
				return
			}
			assert.Nil(t, execution.cmd)
		})
	}
}

func TestSplitLocalCommand(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		query         string
		wantCommand   string
		wantArguments string
		wantSeparator bool
	}{
		{name: "command only", query: `\cd`, wantCommand: `\cd`},
		{name: "space separator", query: `\cd migrations`, wantCommand: `\cd`, wantArguments: "migrations", wantSeparator: true},
		{name: "tab separator", query: "\\cd\t../sql", wantCommand: `\cd`, wantArguments: "../sql", wantSeparator: true},
		{name: "empty argument", query: `\cd   `, wantCommand: `\cd`, wantSeparator: true},
		{name: "leading whitespace", query: ` \cd migrations`, wantArguments: `\cd migrations`, wantSeparator: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			command, arguments, separator := splitLocalCommand(testCase.query)
			assert.Equal(t, testCase.wantCommand, command)
			assert.Equal(t, testCase.wantArguments, arguments)
			assert.Equal(t, testCase.wantSeparator, separator)
		})
	}
}

func TestParsePathArgument(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		raw     string
		want    string
		wantErr string
	}{
		{name: "relative", raw: "migrations", want: "migrations"},
		{name: "parent", raw: "../sql", want: "../sql"},
		{name: "single quoted", raw: `'directory with spaces'`, want: "directory with spaces"},
		{name: "double quoted", raw: `"directory with spaces"`, want: "directory with spaces"},
		{name: "unquoted whitespace", raw: "directory with spaces", wantErr: "quote paths containing whitespace"},
		{name: "unterminated single quote", raw: `'migrations`, wantErr: "unterminated quoted path"},
		{name: "unterminated double quote", raw: `"migrations`, wantErr: "unterminated quoted path"},
		{name: "text after closing quote", raw: `'migrations' extra`, wantErr: "unexpected text after quoted path"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := parsePathArgument(testCase.raw)
			if testCase.wantErr != "" {
				require.ErrorContains(t, err, testCase.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.want, got)
		})
	}
}

func TestChangeWorkingDirectory(t *testing.T) {
	originalDirectory, err := os.Getwd()
	require.NoError(t, err)
	root := t.TempDir()
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalDirectory))
	})

	home := filepath.Join(root, "home")
	quotedTarget := filepath.Join(root, "directory with spaces")
	tildeTarget := filepath.Join(home, "migrations")
	require.NoError(t, os.Mkdir(home, 0o755))
	require.NoError(t, os.Mkdir(quotedTarget, 0o755))
	require.NoError(t, os.Mkdir(tildeTarget, 0o755))
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	t.Run("local command dispatch", func(t *testing.T) {
		require.NoError(t, os.Chdir(root))
		execution, matched := (&pgxCLI{}).runLocal(context.Background(), `\cd "`+quotedTarget+`"`)
		assert.True(t, matched)
		assert.Nil(t, execution.cmd)
		assert.False(t, execution.delegatesExecution)
		assertWorkingDirectory(t, quotedTarget)
	})

	t.Run("quoted absolute path", func(t *testing.T) {
		require.NoError(t, os.Chdir(root))
		require.NoError(t, changeWorkingDirectory(`"`+quotedTarget+`"`))
		assertWorkingDirectory(t, quotedTarget)
	})

	t.Run("home by default", func(t *testing.T) {
		require.NoError(t, os.Chdir(root))
		require.NoError(t, changeWorkingDirectory(""))
		assertWorkingDirectory(t, home)
	})

	t.Run("tilde path", func(t *testing.T) {
		require.NoError(t, os.Chdir(root))
		require.NoError(t, changeWorkingDirectory("~/migrations"))
		assertWorkingDirectory(t, tildeTarget)
	})

	t.Run("missing path", func(t *testing.T) {
		require.NoError(t, os.Chdir(root))
		err := changeWorkingDirectory("missing")
		require.ErrorContains(t, err, `change directory to "missing"`)
		assertWorkingDirectory(t, root)
	})
}

func assertWorkingDirectory(t *testing.T, want string) {
	t.Helper()

	got, err := os.Getwd()
	require.NoError(t, err)
	gotInfo, err := os.Stat(got)
	require.NoError(t, err)
	wantInfo, err := os.Stat(want)
	require.NoError(t, err)
	assert.True(t, os.SameFile(gotInfo, wantInfo), "working directory = %q, want %q", got, want)
}
