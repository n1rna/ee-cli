package origin

import (
	"errors"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

// withLookPath temporarily replaces the package lookPath with one that reports
// the given set of executables as present, restoring the original afterwards.
func withLookPath(t *testing.T, present ...string) {
	t.Helper()
	set := make(map[string]bool, len(present))
	for _, p := range present {
		set[p] = true
	}
	orig := lookPath
	lookPath = func(name string) (string, error) {
		if set[name] {
			return "/usr/bin/" + name, nil
		}
		return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
	}
	t.Cleanup(func() { lookPath = orig })
}

func TestResolveTool(t *testing.T) {
	tests := []struct {
		name    string
		tool    string
		present []string
		want    []string
		wantErr bool
	}{
		{
			name:    "wrangler on PATH is used directly",
			tool:    "wrangler",
			present: []string{"wrangler", "bunx", "npx"},
			want:    []string{"wrangler"},
		},
		{
			name:    "falls back to bunx when wrangler missing",
			tool:    "wrangler",
			present: []string{"bunx", "npx"},
			want:    []string{"bunx", "wrangler"},
		},
		{
			name:    "falls back to npx when only npx available",
			tool:    "wrangler",
			present: []string{"npx"},
			want:    []string{"npx", "-y", "wrangler"},
		},
		{
			name:    "prefers bunx over npx",
			tool:    "wrangler",
			present: []string{"npx", "bunx"},
			want:    []string{"bunx", "wrangler"},
		},
		{
			name:    "errors when wrangler and runners all missing",
			tool:    "wrangler",
			present: []string{},
			wantErr: true,
		},
		{
			name:    "gh on PATH is used directly",
			tool:    "gh",
			present: []string{"gh"},
			want:    []string{"gh"},
		},
		{
			name:    "gh has no runner fallback",
			tool:    "gh",
			present: []string{"bunx", "npx"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withLookPath(t, tt.present...)
			got, err := ResolveTool(tt.tool)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got argv %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ResolveTool(%q) = %v, want %v", tt.tool, got, tt.want)
			}
		})
	}
}

func TestCheckToolWithRunnerFallback(t *testing.T) {
	withLookPath(t, "bunx")
	if err := CheckTool("wrangler"); err != nil {
		t.Errorf("expected wrangler to resolve via bunx, got error: %v", err)
	}
}

func TestCheckToolMissingMentionsRunners(t *testing.T) {
	withLookPath(t) // nothing available
	err := CheckTool("wrangler")
	if err == nil {
		t.Fatal("expected error when wrangler and runners are unavailable")
	}
	msg := err.Error()
	for _, want := range []string{"bunx", "npx"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error message %q should mention %q", msg, want)
		}
	}
}

func TestToolCommandBuildsArgv(t *testing.T) {
	withLookPath(t, "npx")
	cmd, err := ToolCommand("wrangler", "whoami")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// cmd.Args includes the resolved runner prefix plus the tool args.
	want := []string{"npx", "-y", "wrangler", "whoami"}
	if !reflect.DeepEqual(cmd.Args, want) {
		t.Errorf("cmd.Args = %v, want %v", cmd.Args, want)
	}
}

func TestRunCommandErrorsWhenUnresolvable(t *testing.T) {
	withLookPath(t) // nothing available
	_, err := RunCommand("wrangler", "whoami")
	if err == nil {
		t.Fatal("expected error when wrangler cannot be resolved")
	}
	// Should be the resolver error, not an exec start error.
	var execErr *exec.Error
	if errors.As(err, &execErr) {
		t.Errorf("expected resolver error, got exec error: %v", err)
	}
}
