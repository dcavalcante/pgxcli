package app

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/balajz/pgxcli/internal/app/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadIncludeFile(t *testing.T) {
	originalDirectory, err := os.Getwd()
	require.NoError(t, err)
	root := t.TempDir()
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalDirectory))
	})

	home := filepath.Join(root, "home")
	require.NoError(t, os.Mkdir(home, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "query.sql"), []byte("select 1;\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "query notes.sql"), []byte("select 2;\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(home, "home.sql"), []byte("select 3;\n"), 0o600))
	require.NoError(t, os.Chdir(root))
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	testCases := []struct {
		name      string
		argument  string
		wantQuery string
	}{
		{name: "relative path", argument: "query.sql", wantQuery: "select 1;\n"},
		{name: "quoted path", argument: `'query notes.sql'`, wantQuery: "select 2;\n"},
		{name: "home path", argument: "~/home.sql", wantQuery: "select 3;\n"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			query, err := readIncludeFile(testCase.argument)
			require.NoError(t, err)
			assert.Equal(t, testCase.wantQuery, query)
		})
	}
}

func TestReadIncludeFileErrors(t *testing.T) {
	t.Parallel()
	missingPath := filepath.Join(t.TempDir(), "missing.sql")

	testCases := []struct {
		name     string
		argument string
		wantErr  string
	}{
		{name: "missing filename", wantErr: "filename is required"},
		{name: "unquoted whitespace", argument: "query notes.sql", wantErr: "quote paths containing whitespace"},
		{name: "missing file", argument: missingPath, wantErr: `read SQL file "` + missingPath + `"`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := readIncludeFile(testCase.argument)
			require.ErrorContains(t, err, testCase.wantErr)
		})
	}
}

func TestIncludeAliasesDelegateExecution(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "query.sql")
	require.NoError(t, os.WriteFile(path, nil, 0o600))
	cli := &pgxCLI{}

	for _, query := range []string{`\i "` + path + `"`, `\include "` + path + `"`} {
		execution, matched := cli.runLocal(context.Background(), query)
		assert.True(t, matched, query)
		assert.True(t, execution.delegatesExecution, query)
		assert.NotNil(t, execution.cmd, query)
	}

	execution, matched := cli.runLocal(context.Background(), `\includes "`+path+`"`)
	assert.False(t, matched)
	assert.False(t, execution.delegatesExecution)
	assert.Nil(t, execution.cmd)
}

func TestIncludeErrorUsesLocalPromptLifecycle(t *testing.T) {
	t.Parallel()

	execution, matched := (&pgxCLI{}).runLocal(context.Background(), `\i`)
	require.True(t, matched)
	assert.False(t, execution.delegatesExecution)
	assert.NotNil(t, execution.cmd)
}

func TestExecuteIncludeDoesNotAddOuterPrompt(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "query.sql")
	require.NoError(t, os.WriteFile(path, nil, 0o600))
	cli := &pgxCLI{
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		lastQuery: "select previous;",
	}

	cmd := cli.execute(context.Background(), `\i "`+path+`"`)
	require.NotNil(t, cmd)
	_, ok := cmd().(ui.ExecCmdMsg)
	assert.True(t, ok, "successful include should return the query runner message directly")
	assert.Equal(t, "select previous;", cli.lastQuery)
}
