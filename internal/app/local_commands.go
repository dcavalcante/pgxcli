package app

import tea "charm.land/bubbletea/v2"

type localCommand func(*pgxCLI) tea.Cmd

var localCommands = map[string]localCommand{
	"\\clear": func(_ *pgxCLI) tea.Cmd { return tea.ClearScreen },
}

func (p *pgxCLI) runLocal(query string) (tea.Cmd, bool) {
	command, ok := localCommands[query]
	if !ok {
		return nil, false
	}

	return command(p), true
}
