package app

import (
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
		{name: "empty", query: "", match: false},
	}

	cli := &pgxCLI{}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cmd, matched := cli.runLocal(testCase.query)
			assert.Equal(t, testCase.match, matched)
			if testCase.match {
				require.NotNil(t, cmd)
				assert.Equal(t, tea.ClearScreen(), cmd())
				return
			}
			assert.Nil(t, cmd)
		})
	}
}
