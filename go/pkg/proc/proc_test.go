package proc_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/daneroo/dotfiles/go/pkg/proc"
)

func TestOutput(t *testing.T) {
	x := proc.OS{}
	out, err := x.Output(context.Background(), "sh", "-c", "echo out; echo diag >&2")
	if err != nil {
		t.Fatalf("Output failed: %v", err)
	}
	if string(out) != "out\n" {
		t.Errorf("stdout = %q, want only the standard output", out)
	}

	// A failure still returns standard output, as npm outdated needs
	out, err = x.Output(context.Background(), "sh", "-c", "echo partial; echo why >&2; exit 3")
	if err == nil {
		t.Fatal("Output succeeded for a failing program")
	}
	if string(out) != "partial\n" {
		t.Errorf("stdout = %q, want partial", out)
	}
	for _, want := range []string{"sh -c 'echo partial; echo why >&2; exit 3' failed", "exit status 3", "stderr: why"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error lacks %q: %v", want, err)
		}
	}
}

func TestRunUsesNativeStreams(t *testing.T) {
	var stdout, stderr strings.Builder
	x := proc.OS{Stdin: strings.NewReader("answer\n"), Stdout: &stdout, Stderr: &stderr}
	if err := x.Run(context.Background(), "sh", "-c", "read a; echo got $a; echo diag >&2"); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if stdout.String() != "got answer\n" || stderr.String() != "diag\n" {
		t.Errorf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
}

// Cancellation interrupts the program, which may clean up before exiting.
func TestCancelInterruptsGracefully(t *testing.T) {
	cleaned := filepath.Join(t.TempDir(), "cleaned")
	ready := filepath.Join(t.TempDir(), "ready")
	script := `trap 'touch "$1"; exit 0' INT; touch "$2"; while :; do sleep 0.05; done`
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		waitFor(t, ready)
		cancel()
	}()
	start := time.Now()
	err := proc.OS{WaitDelay: 5 * time.Second}.Run(ctx, "sh", "-c", script, "sh", cleaned, ready)
	if err == nil {
		t.Error("Run of a cancelled program succeeded")
	}
	if took := time.Since(start); took > 3*time.Second {
		t.Errorf("an interrupted program took %v to stop", took)
	}
	if _, err := os.Stat(cleaned); err != nil {
		t.Errorf("the program was not interrupted gracefully: %v", err)
	}
}

// A program that ignores the interrupt is killed once WaitDelay has passed.
func TestCancelKillsAfterWaitDelay(t *testing.T) {
	ready := filepath.Join(t.TempDir(), "ready")
	script := `trap '' INT; touch "$1"; while :; do sleep 0.05; done`
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		waitFor(t, ready)
		cancel()
	}()
	start := time.Now()
	_, err := proc.OS{WaitDelay: 200 * time.Millisecond}.Output(ctx, "sh", "-c", script, "sh", ready)
	if err == nil {
		t.Error("Output of a cancelled program succeeded")
	}
	if took := time.Since(start); took > 3*time.Second {
		t.Errorf("a program ignoring the interrupt took %v to be killed", took)
	}
}

func TestFormat(t *testing.T) {
	got := proc.Format("mise", "use", "-g", "node@>=20, <22", "it's")
	want := `mise use -g 'node@>=20, <22' 'it'\''s'`
	if got != want {
		t.Errorf("Format = %s, want %s", got, want)
	}
}

// waitFor waits until the file exists, so a signal is not sent before the
// program has set its trap.
func waitFor(t *testing.T, path string) {
	for range 200 {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("%s never appeared", path)
}
