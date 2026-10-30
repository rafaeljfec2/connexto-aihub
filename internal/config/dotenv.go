package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// maxDotEnvSize bounds the accepted dotenv file size.
const maxDotEnvSize = 1 << 20

type dotEnvEntry struct {
	key   string
	value string
}

// LoadDotEnv reads KEY=VALUE pairs from path and exports them to the process
// environment. Existing variables are never overwritten, so the real
// environment takes precedence over the file.
//
// The file is parsed completely before anything is applied: when parsing fails,
// no variable is set. A missing file is reported as fs.ErrNotExist so the
// caller can decide whether the file is optional (it wraps os.ErrNotExist).
func LoadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open env file: %w", err)
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(io.LimitReader(file, maxDotEnvSize+1))
	if err != nil {
		return fmt.Errorf("read env file %q: %w", path, err)
	}

	if len(content) > maxDotEnvSize {
		return fmt.Errorf("env file %q exceeds %d bytes", path, maxDotEnvSize)
	}

	entries, err := parseDotEnv(string(content))
	if err != nil {
		return fmt.Errorf("parse env file %q: %w", path, err)
	}

	// Duplicates inside the same file resolve to the last occurrence, while the
	// real environment still wins over every file entry.
	values := make(map[string]string, len(entries))
	order := make([]string, 0, len(entries))

	for _, entry := range entries {
		if _, seen := values[entry.key]; !seen {
			order = append(order, entry.key)
		}

		values[entry.key] = entry.value
	}

	for _, key := range order {
		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		if err := os.Setenv(key, values[key]); err != nil {
			return fmt.Errorf("set %s from env file %q: %w", key, path, err)
		}
	}

	return nil
}

func parseDotEnv(content string) ([]dotEnvEntry, error) {
	lines := strings.Split(content, "\n")
	entries := make([]dotEnvEntry, 0, len(lines))

	for index, raw := range lines {
		lineNumber := index + 1
		line := strings.TrimSpace(raw)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))

		key, value, found := strings.Cut(line, "=")
		if !found {
			return nil, fmt.Errorf("line %d: expected KEY=VALUE", lineNumber)
		}

		key = strings.TrimSpace(key)
		if !isValidEnvKey(key) {
			return nil, fmt.Errorf("line %d: invalid variable name %q", lineNumber, key)
		}

		parsed, err := parseDotEnvValue(strings.TrimSpace(value))
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNumber, err)
		}

		entries = append(entries, dotEnvEntry{key: key, value: parsed})
	}

	return entries, nil
}

func parseDotEnvValue(value string) (string, error) {
	if value == "" {
		return "", nil
	}

	switch value[0] {
	case '"':
		parsed, rest, err := parseQuotedValue(value, '"')
		if err != nil {
			return "", err
		}
		if err := ensureNoContent(rest); err != nil {
			return "", err
		}
		return parsed, nil
	case '\'':
		parsed, rest, err := parseQuotedValue(value, '\'')
		if err != nil {
			return "", err
		}
		if err := ensureNoContent(rest); err != nil {
			return "", err
		}
		return parsed, nil
	default:
		return stripInlineComment(value), nil
	}
}

func parseQuotedValue(value string, quote byte) (string, string, error) {
	var builder strings.Builder

	for index := 1; index < len(value); index++ {
		char := value[index]

		if char == '\\' && quote == '"' && index+1 < len(value) {
			index++
			switch value[index] {
			case 'n':
				builder.WriteByte('\n')
			case 'r':
				builder.WriteByte('\r')
			case 't':
				builder.WriteByte('\t')
			case '"':
				builder.WriteByte('"')
			case '\\':
				builder.WriteByte('\\')
			default:
				builder.WriteByte('\\')
				builder.WriteByte(value[index])
			}
			continue
		}

		if char == quote {
			return builder.String(), value[index+1:], nil
		}

		builder.WriteByte(char)
	}

	return "", "", errors.New("unterminated quoted value")
}

func ensureNoContent(rest string) error {
	rest = strings.TrimSpace(rest)
	if rest == "" || strings.HasPrefix(rest, "#") {
		return nil
	}

	return fmt.Errorf("unexpected content after quoted value: %q", rest)
}

func stripInlineComment(value string) string {
	for index := 1; index < len(value); index++ {
		if value[index] != '#' {
			continue
		}

		if value[index-1] == ' ' || value[index-1] == '\t' {
			return strings.TrimRight(value[:index], " \t")
		}
	}

	return strings.TrimRight(value, " \t")
}

func isValidEnvKey(key string) bool {
	if key == "" {
		return false
	}

	for index, char := range key {
		isLetter := char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z'
		isDigit := char >= '0' && char <= '9'
		isUnderscore := char == '_'

		switch {
		case index == 0 && (isLetter || isUnderscore):
			continue
		case index > 0 && (isLetter || isDigit || isUnderscore):
			continue
		default:
			return false
		}
	}

	return true
}
