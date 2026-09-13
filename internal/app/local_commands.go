package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
)

type localCommand struct {
	acceptsArguments bool
	managesPrompt    bool
	handler          func(context.Context, *pgxCLI, string) (tea.Cmd, error)
}

type localCommandExecution struct {
	cmd tea.Cmd
	// managesPrompt is true when the returned command delegates to an execution
	// path that already schedules the next prompt.
	managesPrompt bool
}

var localCommands = map[string]localCommand{
	"\\clear": {
		handler: func(_ context.Context, _ *pgxCLI, _ string) (tea.Cmd, error) {
			return tea.ClearScreen, nil
		},
	},
	"\\cd": {
		acceptsArguments: true,
		handler: func(_ context.Context, _ *pgxCLI, arguments string) (tea.Cmd, error) {
			return nil, changeWorkingDirectory(arguments)
		},
	},
	"\\e": {
		acceptsArguments: true,
		handler: func(_ context.Context, p *pgxCLI, arguments string) (tea.Cmd, error) {
			return editLocalCommand(p, arguments)
		},
	},
	"\\edit": {
		acceptsArguments: true,
		handler: func(_ context.Context, p *pgxCLI, arguments string) (tea.Cmd, error) {
			return editLocalCommand(p, arguments)
		},
	},
	"\\i": {
		acceptsArguments: true,
		managesPrompt:    true,
		handler:          includeLocalCommand,
	},
	"\\include": {
		acceptsArguments: true,
		managesPrompt:    true,
		handler:          includeLocalCommand,
	},
}

func (p *pgxCLI) runLocal(ctx context.Context, query string) (localCommandExecution, bool) {
	commandName, arguments, hasSeparator := splitLocalCommand(query)
	command, ok := localCommands[commandName]
	// A separator means arguments were supplied; \clear remains an exact match.
	if !ok || hasSeparator && !command.acceptsArguments {
		return localCommandExecution{}, false
	}

	cmd, err := command.handler(ctx, p, arguments)
	if err != nil {
		// Errors stay on the normal local-command path so execute adds the next prompt.
		return localCommandExecution{
			cmd: p.printError(fmt.Errorf("%s: %w", commandName, err)),
		}, true
	}

	return localCommandExecution{
		cmd:           cmd,
		managesPrompt: command.managesPrompt,
	}, true
}

func splitLocalCommand(query string) (command, arguments string, hasSeparator bool) {
	separator := strings.IndexFunc(query, unicode.IsSpace)
	if separator == -1 {
		return query, "", false
	}

	return query[:separator], strings.TrimSpace(query[separator:]), true
}

// parsePathArgument accepts one unquoted path or one single- or double-quoted path.
func parsePathArgument(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}

	quote := raw[0]
	if quote == '\'' || quote == '"' {
		closingQuote := strings.IndexByte(raw[1:], quote)
		if closingQuote == -1 {
			return "", errors.New("unterminated quoted path")
		}
		closingQuote++
		if strings.TrimSpace(raw[closingQuote+1:]) != "" {
			return "", errors.New("unexpected text after quoted path")
		}
		return raw[1:closingQuote], nil
	}

	if strings.ContainsFunc(raw, unicode.IsSpace) {
		return "", errors.New("quote paths containing whitespace")
	}

	return raw, nil
}

func expandHomePath(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") && !strings.HasPrefix(path, `~\`) {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	if path == "~" {
		return home, nil
	}

	return filepath.Join(home, path[2:]), nil
}

func changeWorkingDirectory(raw string) error {
	path := "~"
	if strings.TrimSpace(raw) != "" {
		var err error
		path, err = parsePathArgument(raw)
		if err != nil {
			return err
		}
	}

	expandedPath, err := expandHomePath(path)
	if err != nil {
		return err
	}
	// \cd changes the process working directory for later local path operations.
	if err := os.Chdir(expandedPath); err != nil {
		return fmt.Errorf("change directory to %q: %w", path, err)
	}

	return nil
}
