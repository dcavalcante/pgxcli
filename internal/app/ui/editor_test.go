package ui

import (
	"errors"
	"testing"

	"github.com/balajz/bubbline/editline"
	"github.com/balajz/pgxcli/internal/app/ui/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditorFinishedRestoresInput(t *testing.T) {
	model := &Model{
		input:      components.InputModel{Model: editline.New(80, 5)},
		state:      StateExecuting,
		isSpinning: true,
		styles:     DefaultStyles(),
	}

	updated, _ := model.Update(EditorFinishedMsg{Text: "select 1;\n"})
	result := updated.(*Model)

	assert.Equal(t, StateInput, result.state)
	assert.False(t, result.isSpinning)
	assert.Equal(t, "select 1;\n", result.input.Value())
}

func TestEditorFinishedReportsError(t *testing.T) {
	model := &Model{
		input:      components.InputModel{Model: editline.New(80, 5)},
		state:      StateExecuting,
		isSpinning: true,
		styles:     DefaultStyles(),
	}

	updated, cmd := model.Update(EditorFinishedMsg{Err: errors.New("editor failed")})
	result := updated.(*Model)

	assert.Equal(t, StateInput, result.state)
	assert.False(t, result.isSpinning)
	assert.Empty(t, result.input.Value())
	require.NotNil(t, cmd)
}
