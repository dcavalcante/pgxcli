package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func includeLocalCommand(ctx context.Context, p *pgxCLI, arguments string) (tea.Cmd, error) {
	query, err := readIncludeFile(arguments)
	if err != nil {
		return nil, err
	}

	// runQuery already returns an ExecCmdMsg whose command schedules the next prompt.
	return func() tea.Msg {
		return p.runQuery(ctx, query)
	}, nil
}

func readIncludeFile(arguments string) (string, error) {
	if strings.TrimSpace(arguments) == "" {
		return "", errors.New("filename is required")
	}

	path, err := parsePathArgument(arguments)
	if err != nil {
		return "", err
	}
	expandedPath, err := expandHomePath(path)
	if err != nil {
		return "", err
	}

	contents, err := os.ReadFile(expandedPath)
	if err != nil {
		return "", fmt.Errorf("read SQL file %q: %w", path, err)
	}
	return string(contents), nil
}
