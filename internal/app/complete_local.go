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
	return completeLocalPath(input, line, column, limit, []string{`\cd`}, false)
}

func completeEditFile(input [][]rune, line, column, limit int) (bubbline.Completions, bool) {
	return completeLocalPath(input, line, column, limit, []string{`\edit`, `\e`}, true)
}

func completeLocalPath(
	input [][]rune,
	line, column, limit int,
	commands []string,
	includeFiles bool,
) (bubbline.Completions, bool) {
	currentLine, argumentStart, handled := localCommandArgumentBounds(input, line, column, commands)
	if !handled {
		return nil, false
	}
	if column < argumentStart {
		return nil, true
	}

	prefix, quote := directoryCompletionPrefix(string(currentLine[argumentStart:column]))
	directoryPrefix, namePrefix := filepath.Split(prefix)
	searchDirectory := directoryPrefix
	if searchDirectory == "" {
		searchDirectory = "."
	}

	expandedDirectory, err := expandHomePath(searchDirectory)
	if err != nil {
		return nil, true
	}
	entries, err := os.ReadDir(expandedDirectory)
	if err != nil {
		return nil, true
	}

	words, descriptions := matchingPaths(
		entries,
		expandedDirectory,
		directoryPrefix,
		namePrefix,
		quote,
		limit,
		includeFiles,
	)
	category := "directories"
	if includeFiles {
		category = "files and directories"
	}

	return editline.SimpleWordsCompletionWithDescriptions(
		words,
		descriptions,
		category,
		column,
		argumentStart,
		len(currentLine),
	), true
}

func localCommandArgumentBounds(
	input [][]rune,
	line, column int,
	commands []string,
) ([]rune, int, bool) {
	if len(input) != 1 || line != 0 || column < 0 || column > len(input[0]) {
		return nil, 0, false
	}

	currentLine := input[0]
	for _, commandText := range commands {
		command := []rune(commandText)
		if len(currentLine) <= len(command) || string(currentLine[:len(command)]) != commandText {
			continue
		}
		if !unicode.IsSpace(currentLine[len(command)]) || column <= len(command) {
			continue
		}

		argumentStart := len(command)
		for argumentStart < len(currentLine) && unicode.IsSpace(currentLine[argumentStart]) {
			argumentStart++
		}
		return currentLine, argumentStart, true
	}

	return nil, 0, false
}

func matchingPaths(
	entries []os.DirEntry,
	searchDirectory, directoryPrefix, namePrefix string,
	quote byte,
	limit int,
	includeFiles bool,
) (words, descriptions []string) {
	if limit <= 0 {
		return nil, nil
	}

	includeHidden := strings.HasPrefix(namePrefix, ".")
	for _, entry := range entries {
		isDirectory := entryIsDirectory(searchDirectory, entry)
		if !isDirectory && !includeFiles || !strings.HasPrefix(entry.Name(), namePrefix) {
			continue
		}
		if !includeHidden && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		candidate := directoryPrefix + entry.Name()
		description := "file"
		if isDirectory {
			candidate += completionPathSeparator(directoryPrefix)
			description = "directory"
		}
		candidate, ok := quoteCompletionPath(candidate, quote)
		if !ok {
			continue
		}
		words = append(words, candidate)
		descriptions = append(descriptions, description)
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

	info, err := os.Stat(filepath.Join(parent, entry.Name()))
	return err == nil && info.IsDir()
}
