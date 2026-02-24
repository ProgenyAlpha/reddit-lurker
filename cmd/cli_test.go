package cmd

import "testing"

// TestExtractPositionalAndFlags covers the parser that lets flags appear
// before or after the positional argument.
func TestExtractPositionalAndFlags(t *testing.T) {
	tests := []struct {
		name           string
		input          []string
		wantPositional string
		wantFlags      []string
	}{
		{
			name:           "positional then flags",
			input:          []string{"my query", "--sort", "new"},
			wantPositional: "my query",
			wantFlags:      []string{"--sort", "new"},
		},
		{
			name:           "flags then positional",
			input:          []string{"--sort", "new", "my query"},
			wantPositional: "my query",
			wantFlags:      []string{"--sort", "new"},
		},
		{
			name:           "bool flag then positional",
			input:          []string{"--compact", "my query"},
			wantPositional: "my query",
			wantFlags:      []string{"--compact"},
		},
		{
			name:           "positional only",
			input:          []string{"my query"},
			wantPositional: "my query",
			wantFlags:      []string{},
		},
		{
			name:           "empty args",
			input:          []string{},
			wantPositional: "",
			wantFlags:      []string{},
		},
		{
			name:           "json bool flag then positional",
			input:          []string{"--json", "my query"},
			wantPositional: "my query",
			wantFlags:      []string{"--json"},
		},
		{
			name:           "no-cache bool flag then positional",
			input:          []string{"--no-cache", "my query"},
			wantPositional: "my query",
			wantFlags:      []string{"--no-cache"},
		},
		{
			name:           "multiple flags around positional",
			input:          []string{"--sort", "top", "r/golang", "--limit", "10"},
			wantPositional: "r/golang",
			wantFlags:      []string{"--sort", "top", "--limit", "10"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPositional, gotFlags := extractPositionalAndFlags(tt.input)

			if gotPositional != tt.wantPositional {
				t.Errorf("positional: got %q, want %q", gotPositional, tt.wantPositional)
			}

			// Normalise nil to empty slice for comparison.
			if gotFlags == nil {
				gotFlags = []string{}
			}

			if len(gotFlags) != len(tt.wantFlags) {
				t.Errorf("flags length: got %d (%v), want %d (%v)",
					len(gotFlags), gotFlags, len(tt.wantFlags), tt.wantFlags)
				return
			}
			for i := range tt.wantFlags {
				if gotFlags[i] != tt.wantFlags[i] {
					t.Errorf("flags[%d]: got %q, want %q", i, gotFlags[i], tt.wantFlags[i])
				}
			}
		})
	}
}
