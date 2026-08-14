package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/balajz/pgxcli/internal/app/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditorCommand(t *testing.T) {
	executable, err := os.Executable()
	require.NoError(t, err)
	t.Setenv("PSQL_EDITOR", "")
	t.Setenv("EDITOR", fmt.Sprintf("%q --wait", executable))
	t.Setenv("VISUAL", "")

	cmd, err := editorCommand("query.sql")
	require.NoError(t, err)
	assert.Equal(t, []string{executable, "--wait", "query.sql"}, cmd.Args)
}

func TestEditorNamePrecedence(t *testing.T) {
	t.Setenv("PSQL_EDITOR", "psql-editor --wait")
	t.Setenv("EDITOR", "editor --wait")
	t.Setenv("VISUAL", "visual --wait")
	assert.Equal(t, "psql-editor --wait", editorName())

	t.Setenv("PSQL_EDITOR", "")
	assert.Equal(t, "editor --wait", editorName())

	t.Setenv("EDITOR", "")
	assert.Equal(t, "visual --wait", editorName())
}

func TestEditorNameUsesPlatformDefault(t *testing.T) {
	t.Setenv("PSQL_EDITOR", "")
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")

	if runtime.GOOS == "windows" {
		assert.Equal(t, "notepad.exe", editorName())
	} else {
		assert.Equal(t, "vi", editorName())
	}
}

func TestTemporaryEditorSession(t *testing.T) {
	session, err := newEditorSession("", "select 1;")
	require.NoError(t, err)
	t.Cleanup(session.cleanup)

	data, err := os.ReadFile(session.path)
	require.NoError(t, err)
	assert.Equal(t, "select 1;", string(data))

	require.NoError(t, os.WriteFile(session.path, []byte("select 2;\n"), 0o600))
	msg := session.finish(nil)
	result := requireEditorFinishedMsg(t, msg)
	require.NoError(t, result.Err)
	assert.Equal(t, "select 2;\n", result.Text)
	assert.NoFileExists(t, session.path)
}

func TestNamedEditorSession(t *testing.T) {
	path := filepath.Join(t.TempDir(), "query.sql")
	require.NoError(t, os.WriteFile(path, []byte("select 1;"), 0o600))

	session, err := newEditorSession(path, "ignored query")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte("select 2;"), 0o600))

	result := requireEditorFinishedMsg(t, session.finish(nil))
	require.NoError(t, result.Err)
	assert.Equal(t, "select 2;", result.Text)
	assert.FileExists(t, path)
}

func TestEditorSessionReportsProcessErrorAndCleansUp(t *testing.T) {
	session, err := newEditorSession("", "select 1;")
	require.NoError(t, err)
	t.Cleanup(session.cleanup)

	result := requireEditorFinishedMsg(t, session.finish(errors.New("editor failed")))
	require.ErrorContains(t, result.Err, "run editor: editor failed")
	assert.Empty(t, result.Text)
	assert.NoFileExists(t, session.path)
}

func TestEditAliasesAreHandledLocally(t *testing.T) {
	t.Setenv("PSQL_EDITOR", "")
	t.Setenv("EDITOR", "")
	t.Setenv("VISUAL", "")
	cli := &pgxCLI{}

	for _, query := range []string{`\e`, `\edit`, `\e query.sql`, `\edit query.sql`} {
		cmd, matched := cli.runLocal(query)
		assert.True(t, matched, query)
		assert.NotNil(t, cmd, query)
	}

	cmd, matched := cli.runLocal(`\editor query.sql`)
	assert.False(t, matched)
	assert.Nil(t, cmd)
}

func requireEditorFinishedMsg(t *testing.T, msg any) ui.EditorFinishedMsg {
	t.Helper()

	result, ok := msg.(ui.EditorFinishedMsg)
	require.True(t, ok)
	return result
}
