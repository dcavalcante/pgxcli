package app

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/balajz/bubbline"
	"github.com/balajz/bubbline/editline"
)

func completeChangeDirectory(input [][]rune, line, column, limit int) (bubbline.Completions, bool) {
	currentLine, argumentStart, handled := changeDirectoryArgumentBounds(input, line, column)
	if !handled {
		return nil, false
	}
	if column < argumentStart {
		return nil, true
	}

	prefix, quote := directoryCompletionPrefix(string(currentLine[argumentStart:column]))
	// Keep ~ in the input and append a separator so later completion reads its home directory.
	if prefix == "~" {
		candidate, ok := quoteCompletionPath("~"+string(os.PathSeparator), quote)
		if !ok {
			return nil, true
		}

		return editline.SimpleWordsCompletionWithDescriptions(
			[]string{candidate},
			[]string{"home directory"},
			"directories",
			column,
			argumentStart,
			len(currentLine),
		), true
	}

	directoryPrefix, namePrefix := filepath.Split(prefix)
	searchDirectory := directoryPrefix
	if searchDirectory == "" {
		searchDirectory = "."
	}

	// Read from the expanded path but preserve ~ in the replacement.
	expandedDirectory, err := expandHomeDirectory(searchDirectory)
	if err != nil {
		return nil, true
	}
	entries, err := os.ReadDir(expandedDirectory)
	if err != nil {
		return nil, true
	}

	words, descriptions := matchingDirectories(
		entries,
		expandedDirectory,
		directoryPrefix,
		namePrefix,
		quote,
		limit,
	)

	return editline.SimpleWordsCompletionWithDescriptions(
		words,
		descriptions,
		"directories",
		column,
		argumentStart,
		len(currentLine),
	), true
}

// Complete only a single-line \cd with an argument; bare \cd stays with meta-command completion.
func changeDirectoryArgumentBounds(input [][]rune, line, column int) ([]rune, int, bool) {
	if len(input) != 1 || line != 0 || column < 0 || column > len(input[0]) {
		return nil, 0, false
	}

	currentLine := input[0]
	command := []rune(`\cd`)
	if len(currentLine) <= len(command) || string(currentLine[:len(command)]) != string(command) {
		return nil, 0, false
	}
	if !unicode.IsSpace(currentLine[len(command)]) || column <= len(command) {
		return nil, 0, false
	}

	argumentStart := len(command)
	for argumentStart < len(currentLine) && unicode.IsSpace(currentLine[argumentStart]) {
		argumentStart++
	}
	return currentLine, argumentStart, true
}

func matchingDirectories(
	entries []os.DirEntry,
	searchDirectory, directoryPrefix, namePrefix string,
	quote byte,
	limit int,
) (words, descriptions []string) {
	if limit <= 0 {
		return nil, nil
	}

	includeHidden := strings.HasPrefix(namePrefix, ".")
	for _, entry := range entries {
		if !entryIsDirectory(searchDirectory, entry) || !strings.HasPrefix(entry.Name(), namePrefix) {
			continue
		}
		if !includeHidden && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		candidate := directoryPrefix + entry.Name() + completionPathSeparator(directoryPrefix)
		candidate, ok := quoteCompletionPath(candidate, quote)
		if !ok {
			continue
		}
		words = append(words, candidate)
		descriptions = append(descriptions, "directory")
		if len(words) >= limit {
			break
		}
	}
	return words, descriptions
}

func directoryCompletionPrefix(raw string) (path string, quote byte) {
	if raw == "" || raw[0] != '\'' && raw[0] != '"' {
		return raw, 0
	}

	quote = raw[0]
	path = raw[1:]
	if len(path) > 0 && path[len(path)-1] == quote {
		path = path[:len(path)-1]
	}
	return path, quote
}

func completionPathSeparator(directoryPrefix string) string {
	if strings.HasSuffix(directoryPrefix, "/") {
		return "/"
	}
	if strings.HasSuffix(directoryPrefix, `\`) {
		return `\`
	}
	return string(os.PathSeparator)
}

func quoteCompletionPath(path string, quote byte) (string, bool) {
	if quote != 0 {
		if strings.ContainsRune(path, rune(quote)) {
			return "", false
		}
		return string(quote) + path + string(quote), true
	}
	if !strings.ContainsFunc(path, unicode.IsSpace) {
		return path, true
	}
	if !strings.ContainsRune(path, '\'') {
		return "'" + path + "'", true
	}
	if !strings.ContainsRune(path, '"') {
		return `"` + path + `"`, true
	}
	return "", false
}

func entryIsDirectory(parent string, entry os.DirEntry) bool {
	if entry.IsDir() {
		return true
	}
	if entry.Type()&os.ModeSymlink == 0 {
		return false
	}

	// Resolve symlinks because DirEntry describes the link, not its target.
	info, err := os.Stat(filepath.Join(parent, entry.Name()))
	return err == nil && info.IsDir()
}
