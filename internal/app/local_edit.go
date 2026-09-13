package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/balajz/pgxcli/internal/app/ui"
	"github.com/google/shlex"
)

type editorSession struct {
	path      string
	temporary bool
}

func editLocalCommand(p *pgxCLI, arguments string) (tea.Cmd, error) {
	path, err := parsePathArgument(arguments)
	if err != nil {
		return nil, err
	}
	if path != "" {
		path, err = expandHomePath(path)
		if err != nil {
			return nil, err
		}
	}

	// The input buffer has already been submitted and reset when \e runs, so a
	// bare command seeds its temporary file from the most recently submitted SQL.
	session, err := newEditorSession(path, p.lastQuery)
	if err != nil {
		return nil, err
	}
	command, err := editorCommand(session.path)
	if err != nil {
		session.cleanup()
		return nil, err
	}

	return tea.ExecProcess(command, session.finish), nil
}

func editorCommand(path string) (*exec.Cmd, error) {
	arguments, err := shlex.Split(editorCommandLine())
	if err != nil {
		return nil, fmt.Errorf("parse editor command: %w", err)
	}
	if len(arguments) == 0 || arguments[0] == "" {
		return nil, errors.New("editor command is empty")
	}

	return exec.Command(arguments[0], append(arguments[1:], path)...), nil
}

func editorCommandLine() string {
	// Match psql's editor precedence while still providing a platform default.
	for _, variable := range []string{"PSQL_EDITOR", "EDITOR", "VISUAL"} {
		if editor := strings.TrimSpace(os.Getenv(variable)); editor != "" {
			return editor
		}
	}

	if runtime.GOOS == "windows" {
		return "notepad.exe"
	}
	return "vi"
}

func newEditorSession(path, initialQuery string) (*editorSession, error) {
	if path != "" {
		return &editorSession{path: path}, nil
	}

	file, err := os.CreateTemp("", "pgxcli-*.sql")
	if err != nil {
		return nil, fmt.Errorf("create temporary editor file: %w", err)
	}
	session := &editorSession{path: file.Name(), temporary: true}
	if _, err := file.WriteString(initialQuery); err != nil {
		_ = file.Close()
		session.cleanup()
		return nil, fmt.Errorf("write temporary editor file: %w", err)
	}
	if err := file.Close(); err != nil {
		session.cleanup()
		return nil, fmt.Errorf("close temporary editor file: %w", err)
	}

	return session, nil
}

func (s *editorSession) finish(editorErr error) tea.Msg {
	defer s.cleanup()

	if editorErr != nil {
		return ui.EditorFinishedMsg{Err: fmt.Errorf("run editor: %w", editorErr)}
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return ui.EditorFinishedMsg{Err: fmt.Errorf("read edited file %q: %w", s.path, err)}
	}

	return ui.EditorFinishedMsg{Text: string(data)}
}

func (s *editorSession) cleanup() {
	if s.temporary {
		_ = os.Remove(s.path)
	}
}
