package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aihub/internal/config"
)

func unsetEnvAfterTest(t *testing.T, keys ...string) {
	t.Helper()

	for _, key := range keys {
		t.Cleanup(func() {
			_ = os.Unsetenv(key)
		})
	}
}

func TestLoadDotEnvAppliesValues(t *testing.T) {
	unsetEnvAfterTest(t, "AIHUB_TEST_A", "AIHUB_TEST_B")

	path := writeEnvFile(t, "AIHUB_TEST_A=first\nAIHUB_TEST_B=second\n")
	if err := config.LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv() returned error: %v", err)
	}

	if got, want := os.Getenv("AIHUB_TEST_A"), "first"; got != want {
		t.Fatalf("AIHUB_TEST_A = %q, want %q", got, want)
	}

	if got, want := os.Getenv("AIHUB_TEST_B"), "second"; got != want {
		t.Fatalf("AIHUB_TEST_B = %q, want %q", got, want)
	}
}

func TestLoadDotEnvKeepsProcessEnvironmentPrecedence(t *testing.T) {
	t.Setenv("AIHUB_TEST_PRECEDENCE", "from-process")

	path := writeEnvFile(t, "AIHUB_TEST_PRECEDENCE=from-file\n")
	if err := config.LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv() returned error: %v", err)
	}

	if got, want := os.Getenv("AIHUB_TEST_PRECEDENCE"), "from-process"; got != want {
		t.Fatalf("AIHUB_TEST_PRECEDENCE = %q, want %q", got, want)
	}
}

func TestLoadDotEnvKeepsEmptyProcessVariable(t *testing.T) {
	t.Setenv("AIHUB_TEST_EMPTY", "")

	path := writeEnvFile(t, "AIHUB_TEST_EMPTY=from-file\n")
	if err := config.LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv() returned error: %v", err)
	}

	if got := os.Getenv("AIHUB_TEST_EMPTY"); got != "" {
		t.Fatalf("AIHUB_TEST_EMPTY = %q, want the empty process value preserved", got)
	}
}

func TestLoadDotEnvSupportsCommentsBlankLinesAndExport(t *testing.T) {
	unsetEnvAfterTest(t, "AIHUB_TEST_EXPORT", "AIHUB_TEST_COMMENT")

	path := writeEnvFile(t, `
# full line comment
   # indented comment

export AIHUB_TEST_EXPORT=exported
AIHUB_TEST_COMMENT=value # inline comment
`)
	if err := config.LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv() returned error: %v", err)
	}

	if got, want := os.Getenv("AIHUB_TEST_EXPORT"), "exported"; got != want {
		t.Fatalf("AIHUB_TEST_EXPORT = %q, want %q", got, want)
	}

	if got, want := os.Getenv("AIHUB_TEST_COMMENT"), "value"; got != want {
		t.Fatalf("AIHUB_TEST_COMMENT = %q, want %q", got, want)
	}
}

func TestLoadDotEnvParsesQuotedValues(t *testing.T) {
	unsetEnvAfterTest(t, "AIHUB_TEST_DOUBLE", "AIHUB_TEST_SINGLE", "AIHUB_TEST_HASH")

	path := writeEnvFile(t, `
AIHUB_TEST_DOUBLE="line1\nline2"
AIHUB_TEST_SINGLE='literal $HOME # not a comment'
AIHUB_TEST_HASH=value#kept
`)
	if err := config.LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv() returned error: %v", err)
	}

	if got, want := os.Getenv("AIHUB_TEST_DOUBLE"), "line1\nline2"; got != want {
		t.Fatalf("AIHUB_TEST_DOUBLE = %q, want %q", got, want)
	}

	if got, want := os.Getenv("AIHUB_TEST_SINGLE"), "literal $HOME # not a comment"; got != want {
		t.Fatalf("AIHUB_TEST_SINGLE = %q, want %q", got, want)
	}

	if got, want := os.Getenv("AIHUB_TEST_HASH"), "value#kept"; got != want {
		t.Fatalf("AIHUB_TEST_HASH = %q, want %q", got, want)
	}
}

func TestLoadDotEnvLastDuplicateWins(t *testing.T) {
	unsetEnvAfterTest(t, "AIHUB_TEST_DUP")

	path := writeEnvFile(t, "AIHUB_TEST_DUP=first\nAIHUB_TEST_DUP=second\n")
	if err := config.LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv() returned error: %v", err)
	}

	if got, want := os.Getenv("AIHUB_TEST_DUP"), "second"; got != want {
		t.Fatalf("AIHUB_TEST_DUP = %q, want %q", got, want)
	}
}

func TestLoadDotEnvRejectsInvalidContentWithoutApplyingAnything(t *testing.T) {
	unsetEnvAfterTest(t, "AIHUB_TEST_PARTIAL")

	tests := []struct {
		name    string
		content string
		wantSub string
	}{
		{
			name:    "missing equals sign",
			content: "AIHUB_TEST_PARTIAL=applied\nTHIS_IS_NOT_VALID\n",
			wantSub: "expected KEY=VALUE",
		},
		{
			name:    "invalid variable name",
			content: "AIHUB_TEST_PARTIAL=applied\n1INVALID=value\n",
			wantSub: "invalid variable name",
		},
		{
			name:    "unterminated quote",
			content: "AIHUB_TEST_PARTIAL=applied\nAIHUB_TEST_QUOTE=\"open\n",
			wantSub: "unterminated quoted value",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := writeEnvFile(t, test.content)

			err := config.LoadDotEnv(path)
			if err == nil {
				t.Fatal("LoadDotEnv() must fail on invalid content")
			}

			if !strings.Contains(err.Error(), test.wantSub) {
				t.Fatalf("error = %q, want it to contain %q", err, test.wantSub)
			}

			if _, exists := os.LookupEnv("AIHUB_TEST_PARTIAL"); exists {
				t.Fatal("no variable may be applied when parsing fails")
			}
		})
	}
}

func TestLoadDotEnvReportsMissingFile(t *testing.T) {
	err := config.LoadDotEnv(filepath.Join(t.TempDir(), "missing.env"))
	if err == nil {
		t.Fatal("LoadDotEnv() must fail when the file does not exist")
	}

	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("error = %v, want it to wrap os.ErrNotExist", err)
	}
}

func TestLoadDotEnvIgnoresNoFileWhenEmpty(t *testing.T) {
	path := writeEnvFile(t, "")
	if err := config.LoadDotEnv(path); err != nil {
		t.Fatalf("LoadDotEnv() on an empty file returned error: %v", err)
	}
}
