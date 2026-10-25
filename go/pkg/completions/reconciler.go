// Package completions keeps Bash completion scripts current in
// bash-completion's per-user directory, where bash loads each one on demand,
// the first time its command is completed.
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

// Marker is the first line of every script reconfig writes, which tells its
// scripts from those other installers put in the same directory.
const Marker = "# Written by reconfig (daneroo/dotfiles); changes are overwritten on its next run.\n"

// Tool is a command whose completion script reconfig writes, in a file named
// after the command.
type Tool struct {
	// Name is the command completed, and the script's file name, such as
	// "docker".
	Name string
	// Command is the executable the script needs, such as "docker"; a
	// missing one skips the tool.
	Command string
	// Args make Command print its Bash completion script, such as
	// ["completion", "bash"].
	Args []string
	// Script, when set, is the whole script, written instead of running
	// Command, such as aws's, which asks aws_completer on each Tab.
	Script string
	// bun marks Bun's own generation, isolated and with bug 2 fixed.
	bun bool
}

// Bun is Bun's tool. Its script is generated in isolation and corrected,
// because of 2 upstream bugs:
//   - bug 1: bun completions writes its script itself, named
//     bun.completion.bash instead of "bun", which bash-completion's on-demand
//     loader needs. https://github.com/oven-sh/bun/issues/671
//   - bug 2: a broken regex prints "invalid regular expression" on every
//     bun run <Tab>. https://github.com/oven-sh/bun/issues/24847
var Bun = Tool{Name: "bun", Command: "bun", bun: true}

// Dir is bash-completion's per-user directory, which it searches first when
// it loads a script on demand: under XDG_DATA_HOME, or under home's
// .local/share when that is unset or empty. BASH_COMPLETION_USER_DIR, a list
// that replaces it, is not followed.
func Dir(home string) string {
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" {
		data = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(data, "bash-completion", "completions")
}

// Reconciler keeps the tools' completion scripts current in one directory
// under the run.Runner.
type Reconciler struct {
	dir   string
	tools []Tool
	x     proc.Exec
}

// New returns the completions reconciler for the tools' scripts in dir.
// Tools run through x.
func New(dir string, tools []Tool, x proc.Exec) *Reconciler {
	return &Reconciler{dir: dir, tools: tools, x: x}
}

// Name is the reconciler's stable name.
func (r *Reconciler) Name() string { return "CLI Completions" }

// Plan generates each script in memory and compares it with its target, then
// plans to remove the marked scripts of tools no longer listed. It writes no
// target and creates no directory. A missing tool is skipped, its script
// kept; a failed generation is an error.
func (r *Reconciler) Plan(ctx context.Context) (run.Plan, error) {
	p := &completionsPlan{}
	listed := map[string]bool{}
	for _, t := range r.tools {
		listed[t.Name] = true
		script, ok, err := r.script(ctx, p, t)
		if err != nil {
			return nil, err
		}
		if !ok {
			p.skip(t.Name)
			continue
		}
		if err := p.compare(t.Name, filepath.Join(r.dir, t.Name), script); err != nil {
			return nil, err
		}
	}
	unlisted, err := unlistedScripts(r.dir, listed)
	if err != nil {
		return nil, err
	}
	for _, name := range unlisted {
		path := filepath.Join(r.dir, name)
		p.notes = append(p.notes, run.Note{Mark: run.Problem, Text: fmt.Sprintf("%s completion script is no longer listed (%s)", name, path)})
		p.removes = append(p.removes, path)
	}
	return p, nil
}

// unlistedScripts names the marked scripts in dir whose tools are not
// listed. Unmarked files, links and directories are never reconfig's.
func unlistedScripts(dir string, listed map[string]bool) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list completion scripts in %s: %w", dir, err)
	}
	var names []string
	for _, e := range entries {
		if listed[e.Name()] || !e.Type().IsRegular() {
			continue
		}
		marked, err := hasMarker(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		if marked {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// hasMarker reports whether the file starts with the Marker.
func hasMarker(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("failed to read %s: %w", path, err)
	}
	defer f.Close()
	head := make([]byte, len(Marker))
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("failed to read %s: %w", path, err)
	}
	return string(head[:n]) == Marker, nil
}

// script produces a tool's marked script, one of three ways: Bun's own
// generation, a fixed script, or the tool's command. ok is false when the
// tool is missing.
func (r *Reconciler) script(ctx context.Context, p *completionsPlan, t Tool) (script []byte, ok bool, err error) {
	switch {
	case t.bun:
		script, ok, err = r.bunScript(ctx, p)
	case t.Script != "":
		script, ok = []byte(t.Script), r.installed(ctx, t.Command)
	default:
		script, ok, err = r.generated(ctx, t)
	}
	if !ok || err != nil {
		return nil, ok, err
	}
	return append([]byte(Marker), script...), true, nil
}

// generated runs the tool's command to print its script.
func (r *Reconciler) generated(ctx context.Context, t Tool) ([]byte, bool, error) {
	cmd, ok := r.resolve(ctx, t.Command)
	if !ok {
		return nil, false, nil
	}
	cmd.Args = t.Args
	script, err := generate(ctx, r.x, t.Name, cmd)
	return script, true, err
}

// bunScript generates Bun's script with bug 2 fixed.
func (r *Reconciler) bunScript(ctx context.Context, p *completionsPlan) ([]byte, bool, error) {
	exe, ok := r.bunExecutable(ctx)
	if !ok {
		return nil, false, nil
	}
	script, err := generateBun(ctx, r.x, exe)
	if err != nil {
		return nil, true, err
	}
	script, fixed := fixBun(script)
	if !fixed {
		p.notes = append(p.notes, run.Note{Mark: run.Info, Text: "bun: its completion script no longer has the known regex bug, nothing to fix"})
	}
	return script, true, nil
}

// installed reports whether a tool is on PATH or installed through mise.
func (r *Reconciler) installed(ctx context.Context, name string) bool {
	if _, err := r.x.LookPath(name); err == nil {
		return true
	}
	_, ok := r.miseWhich(ctx, name)
	return ok
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
		return nil, fmt.Errorf("failed to get %s completion script: %w", name, err)
	}
	if len(bytes.TrimSpace(out)) == 0 {
		return nil, fmt.Errorf("failed to get %s completion script: %s printed nothing", name, c)
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

// completionsPlan is the completions plan: its notes, and the approved
// scripts to write and to remove.
type completionsPlan struct {
	notes   []run.Note
	writes  []write
	removes []string
}

// write is a script to write to its target.
type write struct {
	target string
	script []byte
}

// skip notes a missing tool.
func (p *completionsPlan) skip(name string) {
	p.notes = append(p.notes, run.Note{Mark: run.Info, Text: name + " is not installed, skipping its completion script"})
}

// compare notes whether the target already holds the script, and plans to
// write it when it does not. A link at the target is replaced by a file
// rather than written through, so it never changes the file it points to.
func (p *completionsPlan) compare(name, target string, script []byte) error {
	info, err := os.Lstat(target)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		p.notes = append(p.notes, run.Note{Mark: run.Problem, Text: fmt.Sprintf("%s completion script is missing (%s)", name, target)})
	case err != nil:
		return fmt.Errorf("failed to check %s completion script: %w", name, err)
	case info.Mode()&fs.ModeSymlink != 0:
		p.notes = append(p.notes, run.Note{Mark: run.Problem, Text: fmt.Sprintf("%s completion script: %s is a symlink, to be replaced by a file", name, target)})
	case !info.Mode().IsRegular():
		return fmt.Errorf("failed to check %s completion script: %s is not a regular file", name, target)
	default:
		current, err := os.ReadFile(target)
		if err != nil {
			return fmt.Errorf("failed to read %s completion script: %w", name, err)
		}
		if bytes.Equal(current, script) {
			p.notes = append(p.notes, run.Note{Mark: run.OK, Text: fmt.Sprintf("%s completion script is up to date (%s)", name, target)})
			return nil
		}
		p.notes = append(p.notes, run.Note{Mark: run.Problem, Text: fmt.Sprintf("%s completion script is outdated (%s)", name, target)})
	}
	p.writes = append(p.writes, write{target: target, script: script})
	return nil
}

// Report describes the plan.
func (p *completionsPlan) Report() run.Report {
	writes := run.Group{Title: "Completion scripts to write"}
	for _, w := range p.writes {
		writes.Actions = append(writes.Actions, "write "+w.target)
	}
	removes := run.Group{Title: "Completion scripts to remove"}
	for _, path := range p.removes {
		removes.Actions = append(removes.Actions, "remove "+path)
	}
	r := run.Report{Groups: []run.Group{writes, removes}}
	if len(p.notes) > 0 {
		r.Paragraphs = [][]run.Note{p.notes}
	}
	return r
}

// Apply writes the approved scripts to the changed targets, then removes the
// unlisted ones, in order, stopping at the first failure.
func (p *completionsPlan) Apply(ctx context.Context) error {
	for _, w := range p.writes {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := replaceFile(w.target, w.script); err != nil {
			return err
		}
	}
	for _, path := range p.removes {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("failed to remove %s: %w", path, err)
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
