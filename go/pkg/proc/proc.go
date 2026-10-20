// Package proc runs native programs, such as package managers. Reconcilers
// take an Exec when they are constructed, so tests can substitute controlled
// processes for the real ones.
package proc

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Exec finds and runs native programs.
type Exec interface {
	// LookPath reports where the named program is, or an error when it is
	// not installed.
	LookPath(name string) (string, error)
	// Output runs a program and returns its standard output, even when it
	// fails. A failure includes the command and its standard error.
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
	// Run runs a program attached to the native standard streams, so it can
	// show progress and ask for authentication.
	Run(ctx context.Context, name string, args ...string) error
}

// DefaultWaitDelay is how long an interrupted program has to exit before it
// is killed.
const DefaultWaitDelay = 5 * time.Second

// OS runs programs on this machine. When the context is cancelled, the
// program is interrupted, then killed if it has not exited after WaitDelay.
type OS struct {
	// Stdin, Stdout and Stderr are the native streams for Run; nil means
	// the process's own.
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	// WaitDelay bounds the wait after an interrupt; zero means
	// DefaultWaitDelay.
	WaitDelay time.Duration
}

// LookPath finds the program on PATH.
func (o OS) LookPath(name string) (string, error) {
	// A PATH lookup, not `command -v`: that is a shell builtin with no
	// executable on Linux, so running it always fails there.
	return exec.LookPath(name)
}

// Output runs the program with its standard output and standard error
// captured.
func (o OS) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd := o.command(ctx, name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stdout.Bytes(), fmt.Errorf("%s failed: %w\nstderr: %s", Format(name, args...), err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// Run runs the program attached to the native streams.
func (o OS) Run(ctx context.Context, name string, args ...string) error {
	cmd := o.command(ctx, name, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if o.Stdin != nil {
		cmd.Stdin = o.Stdin
	}
	if o.Stdout != nil {
		cmd.Stdout = o.Stdout
	}
	if o.Stderr != nil {
		cmd.Stderr = o.Stderr
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s failed: %w", Format(name, args...), err)
	}
	return nil
}

// command prepares a program that is interrupted, not killed, when the
// context is cancelled, and killed once WaitDelay has passed.
func (o OS) command(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
	cmd.WaitDelay = o.WaitDelay
	if cmd.WaitDelay == 0 {
		cmd.WaitDelay = DefaultWaitDelay
	}
	return cmd
}

// Format renders a command for display, quoting arguments a reader could
// misread. The result is only for reading: it is never run.
func Format(name string, args ...string) string {
	words := []string{name}
	for _, a := range args {
		if a == "" || strings.ContainsAny(a, " \t\n'\"\\$`;&|<>()*?[]{}#~") {
			a = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
		}
		words = append(words, a)
	}
	return strings.Join(words, " ")
}
