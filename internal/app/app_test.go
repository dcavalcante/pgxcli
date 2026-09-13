package app

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecuteRemembersOnlySQLInput(t *testing.T) {
	t.Parallel()

	cli := &pgxCLI{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	cli.execute(context.Background(), "select 1;")
	assert.Equal(t, "select 1;", cli.lastQuery)

	cli.execute(context.Background(), `\c postgres`)
	assert.Equal(t, "select 1;", cli.lastQuery)
}
