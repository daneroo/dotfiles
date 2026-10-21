// Package completions keeps Bash completion scripts current: scripts cached
// in the repository, and Bun's corrected global script.
package completions

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/daneroo/dotfiles/go/pkg/proc"
	"github.com/daneroo/dotfiles/go/pkg/run"
)

// Generator is a tool whose Bash completion script is cached in the
// repository, so shells source a file instead of running the tool at startup.
type Generator struct {
	// Name is the tool's display name, such as "docker".
	Name string
	// Command is the executable, such as "docker".
	Command string
	// Args make the tool print its Bash completion script, such as
	// ["completion", "bash"].
	Args []string
	// Target is the cached script's path, such as
	// "<repo>/core/.config/bash_includes/docker_completion.bash".
	Target string
}

// Reconciler keeps completion scripts current under the run.Runner: the
// scripts cached in the repository, and Bun's corrected global script.
type Reconciler struct {
	cached    []Generator
	bunTarget string
	x         proc.Exec
}

// New returns the completions reconciler for the cached generators and for
// Bun's corrected script at bunTarget, the file "bun" in the global
// bash-completion directory; an empty bunTarget leaves Bun out. Tools run
// through x.
//
// Bun gets its own target because of 2 upstream bugs:
//   - bug 1: Bun names its script bun.completion.bash instead of "bun",
//     which bash-completion v2's on-demand loader needs.
//     https://github.com/oven-sh/bun/issues/671
//   - bug 2: a broken regex prints "invalid regular expression" on every
//     bun run <Tab>. https://github.com/oven-sh/bun/issues/24847
//
// Bun's own file is never written, so it stays as proof both bugs are still
// unfixed.
func New(cached []Generator, bunTarget string, x proc.Exec) *Reconciler {
	return &Reconciler{cached: cached, bunTarget: bunTarget, x: x}
}

// Name is the reconciler's stable name.
func (r *Reconciler) Name() string { return "CLI Completions" }

// Plan generates each script in memory and compares it with its target. It
// writes no target and creates no target directory. A missing tool is
// skipped; a failed generation is an error.
func (r *Reconciler) Plan(ctx context.Context) (run.Plan, error) {
	p := &completionsPlan{}
	for _, g := range r.cached {
		cmd, ok := r.resolve(ctx, g.Command)
		if !ok {
			p.skip(g.Name)
			continue
		}
		cmd.Args = g.Args
		script, err := generate(ctx, r.x, g.Name, cmd)
		if err != nil {
			return nil, err
		}
		if err := p.compare(g.Name, g.Target, script); err != nil {
			return nil, err
		}
	}
	if r.bunTarget != "" {
		if err := r.planBun(ctx, p); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// planBun compares Bun's fresh script, with bug 2 fixed, with its target.
func (r *Reconciler) planBun(ctx context.Context, p *completionsPlan) error {
	exe, ok := r.bunExecutable(ctx)
	if !ok {
		p.skip("bun")
		return nil
	}
	script, err := generateBun(ctx, r.x, exe)
	if err != nil {
		return err
	}
	script, fixed := fixBun(script)
	if !fixed {
		p.notes = append(p.notes, run.Note{Mark: run.Info, Text: "bun: its completion script no longer has the known regex bug, nothing to fix"})
	}
	return p.compare("bun", r.bunTarget, script)
}

// resolve finds a tool on PATH or, when the tool was installed earlier in
// this run and the parent shell has not loaded its location, through mise:
// mise which names the executable, and its directory goes first on PATH, so
// programs it runs, such as node for npm, come from the same installation.
func (r *Reconciler) resolve(ctx context.Context, name string) (proc.Cmd, bool) {
	if _, err := r.x.LookPath(name); err == nil {
		return proc.Command(name), true
	}
	path, ok := r.miseWhich(ctx, name)
	if !ok {
		return proc.Cmd{}, false
	}
	return proc.Command(path).WithEnv(prependPath(filepath.Dir(path))), true
}

// bunExecutable finds Bun's own executable, which generation copies. On
// PATH, bun is often a mise shim, a link to mise itself, so mise answers
// first; otherwise the links of bun on PATH are followed.
func (r *Reconciler) bunExecutable(ctx context.Context) (string, bool) {
	if path, ok := r.miseWhich(ctx, "bun"); ok {
		return path, true
	}
	path, err := r.x.LookPath("bun")
	if err != nil {
		return "", false
	}
	exe, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", false
	}
	return exe, true
}

// miseWhich asks mise where a tool it manages is, by running mise which.
func (r *Reconciler) miseWhich(ctx context.Context, name string) (string, bool) {
	if _, err := r.x.LookPath("mise"); err != nil {
		return "", false
	}
	out, err := r.x.Output(ctx, proc.Command("mise", "which", name))
	path := strings.TrimSpace(string(out))
	if err != nil || path == "" {
		return "", false
	}
	return path, true
}

// generate runs a tool to print its completion script, keeping only its
// standard output: diagnostics on standard error never reach the script.
func generate(ctx context.Context, x proc.Exec, name string, c proc.Cmd) ([]byte, error) {
	out, err := x.Output(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("failed to get %s completion: %w", name, err)
	}
	if len(bytes.TrimSpace(out)) == 0 {
		return nil, fmt.Errorf("failed to get %s completion: %s printed nothing", name, c)
	}
	return out, nil
}

// generateBun runs a copy of Bun's executable to print its Bash completion
// script, isolated in scratch. Before printing, bun completions links bunx
// beside its executable when there is no bunx on PATH: a harmless bunx goes
// first on PATH, and the copy confines any link to scratch anyway. Bun's
// install, XDG and temporary directories, the Homebrew prefix and the working
// directory are scratch too. Scratch is removed on every path.
func generateBun(ctx context.Context, x proc.Exec, exe string) ([]byte, error) {
	scratch, err := os.MkdirTemp("", "reconfig-bun-")
	if err != nil {
		return nil, fmt.Errorf("failed to create scratch for bun completion: %w", err)
	}
	defer os.RemoveAll(scratch)

	dir := func(name string) string { return filepath.Join(scratch, name) }
	for _, name := range []string{"bin", "path", "install", "config", "data", "state", "cache", "tmp", "homebrew", "work"} {
		if err := os.Mkdir(dir(name), 0o755); err != nil {
			return nil, fmt.Errorf("failed to create scratch for bun completion: %w", err)
		}
	}
	bun := filepath.Join(dir("bin"), "bun")
	if err := copyExecutable(exe, bun); err != nil {
		return nil, fmt.Errorf("failed to copy %s to scratch: %w", exe, err)
	}
	if err := os.WriteFile(filepath.Join(dir("path"), "bunx"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create scratch for bun completion: %w", err)
	}

	c := proc.Command(bun, "completions").WithEnv(
		prependPath(dir("path")),
		"SHELL=/bin/bash", // bun completions picks the script for SHELL
		"BUN_INSTALL="+dir("install"),
		"XDG_CONFIG_HOME="+dir("config"),
		"XDG_DATA_HOME="+dir("data"),
		"XDG_STATE_HOME="+dir("state"),
		"XDG_CACHE_HOME="+dir("cache"),
		"TMPDIR="+dir("tmp"),
		"HOMEBREW_PREFIX="+dir("homebrew"),
	).WithDir(dir("work"))
	return generate(ctx, x, "bun", c)
}

// copyExecutable copies the executable at src to dst.
func copyExecutable(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// fixBun fixes bug 2 by returning before the broken regex is used, and
// reports whether the script still had the bug.
func fixBun(script []byte) ([]byte, bool) {
	const buggyLine = "local re_prev_script="
	const patchedLine = "return; local re_prev_script="
	if !bytes.Contains(script, []byte(buggyLine)) {
		return script, false
	}
	return bytes.Replace(script, []byte(buggyLine), []byte(patchedLine), 1), true
}

// prependPath returns the PATH setting with dir first.
func prependPath(dir string) string {
	return "PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH")
}

// completionsPlan is the completions plan: its notes and the approved
// scripts to write.
type completionsPlan struct {
	notes  []run.Note
	writes []write
}

// write is a script to write to its target.
type write struct {
	target string
	script []byte
}

// skip notes a missing tool.
func (p *completionsPlan) skip(name string) {
	p.notes = append(p.notes, run.Note{Mark: run.Info, Text: name + " is not installed, skipping its completions"})
}

// compare notes whether the target already holds the script, and plans to
// write it when it does not. A link at the target is replaced by a file
// rather than written through, so it never changes the file it points to.
func (p *completionsPlan) compare(name, target string, script []byte) error {
	info, err := os.Lstat(target)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		p.notes = append(p.notes, run.Note{Mark: run.Problem, Text: fmt.Sprintf("%s completions are missing (%s)", name, target)})
	case err != nil:
		return fmt.Errorf("failed to check %s completions: %w", name, err)
	case info.Mode()&fs.ModeSymlink != 0:
		p.notes = append(p.notes, run.Note{Mark: run.Problem, Text: fmt.Sprintf("%s completions: %s is a symlink, to be replaced by a file", name, target)})
	case !info.Mode().IsRegular():
		return fmt.Errorf("failed to check %s completions: %s is not a regular file", name, target)
	default:
		current, err := os.ReadFile(target)
		if err != nil {
			return fmt.Errorf("failed to read %s completions: %w", name, err)
		}
		if bytes.Equal(current, script) {
			p.notes = append(p.notes, run.Note{Mark: run.OK, Text: fmt.Sprintf("%s completions are up to date (%s)", name, target)})
			return nil
		}
		p.notes = append(p.notes, run.Note{Mark: run.Problem, Text: fmt.Sprintf("%s completions are outdated (%s)", name, target)})
	}
	p.writes = append(p.writes, write{target: target, script: script})
	return nil
}

// Report describes the plan.
func (p *completionsPlan) Report() run.Report {
	g := run.Group{Title: "Completion files to write"}
	for _, w := range p.writes {
		g.Actions = append(g.Actions, "write "+w.target)
	}
	r := run.Report{Groups: []run.Group{g}}
	if len(p.notes) > 0 {
		r.Paragraphs = [][]run.Note{p.notes}
	}
	return r
}

// Apply writes the approved scripts to the changed targets, in order,
// stopping at the first failure.
func (p *completionsPlan) Apply(ctx context.Context) error {
	for _, w := range p.writes {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := replaceFile(w.target, w.script); err != nil {
			return err
		}
	}
	return nil
}

// replaceFile writes the content beside the target, then renames it over the
// target, so a shell never sources a partial script.
func replaceFile(target string, content []byte) (err error) {
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	f, err := os.CreateTemp(dir, "."+filepath.Base(target)+".reconfig-*")
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", target, err)
	}
	defer func() {
		if err != nil {
			os.Remove(f.Name())
		}
	}()
	if _, err = f.Write(content); err != nil {
		f.Close()
		return fmt.Errorf("failed to write %s: %w", target, err)
	}
	if err = f.Chmod(0o644); err != nil {
		f.Close()
		return fmt.Errorf("failed to write %s: %w", target, err)
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("failed to write %s: %w", target, err)
	}
	if err = os.Rename(f.Name(), target); err != nil {
		return fmt.Errorf("failed to write %s: %w", target, err)
	}
	return nil
}
