package process_test

import (
	"testing"

	"github.com/lokeshreddygoli/hotreload/internal/process"
)

func TestShellArgs(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{
			input:    "go build -o ./bin/server ./cmd/server",
			expected: []string{"go", "build", "-o", "./bin/server", "./cmd/server"},
		},
		{
			input:    `go build -ldflags "-s -w" -o ./bin/server`,
			expected: []string{"go", "build", "-ldflags", "-s -w", "-o", "./bin/server"},
		},
		{
			input:    `echo 'hello world'`,
			expected: []string{"echo", "hello world"},
		},
		{
			input:    "./bin/server",
			expected: []string{"./bin/server"},
		},
		{
			input:    "  go  build  ",
			expected: []string{"go", "build"},
		},
		{
			input:    `go build -o "C:\My Project\bin\server.exe" .`,
			expected: []string{"go", "build", "-o", `C:\My Project\bin\server.exe`, "."},
		},
	}

	for _, tt := range tests {
		got := process.ShellArgs(tt.input)
		if len(got) != len(tt.expected) {
			t.Errorf("ShellArgs(%q):\n  got  %v\n  want %v", tt.input, got, tt.expected)
			continue
		}
		for i := range got {
			if got[i] != tt.expected[i] {
				t.Errorf("ShellArgs(%q)[%d]: got %q, want %q", tt.input, i, got[i], tt.expected[i])
			}
		}
	}
}
