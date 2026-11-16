package screenshots

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// Error messages contain only public, locally controlled metadata.
type Error string

func (e Error) Error() string { return string(e) }

func require(condition bool, message string) error {
	if condition {
		return nil
	}
	return Error(message)
}

// run executes a command with a deadline and returns its standard output.
func run(timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := exec.CommandContext(ctx, args[0], args[1:]...)
	var stdout bytes.Buffer
	command.Stdout, command.Stderr = &stdout, io.Discard
	err := command.Run()
	if ctx.Err() != nil {
		return "", errors.New("command timed out")
	}
	var exit *exec.ExitError
	if err != nil && !errors.As(err, &exit) {
		return "", errors.New("command could not start")
	}
	code := command.ProcessState.ExitCode()
	if err := require(code == 0, fmt.Sprintf("%s operation failed (exit=%d)", filepath.Base(args[0]), code)); err != nil {
		return "", err
	}
	return stdout.String(), nil
}

func run120(args ...string) (string, error) { return run(120*time.Second, args...) }

// cleanupStep is one bounded cleanup action with a fixed public label.
type cleanupStep struct {
	label  string
	action func() error
}

// cleanup attempts every step and reports only the labels of failed steps. When
// a primary failure is already being reported, cleanup failures are printed instead.
func cleanup(steps []cleanupStep, primary error, stderr io.Writer) error {
	var failures []string
	for _, step := range steps {
		if err := step.action(); err != nil {
			failures = append(failures, step.label)
		}
	}
	if len(failures) == 0 {
		return primary
	}
	message := "Cleanup failed: " + strings.Join(failures, ", ") + "; details kept private"
	if primary != nil {
		fmt.Fprintln(stderr, message)
		return primary
	}
	return Error(message)
}

// privateProcess consumes potentially authenticated debug output without public logging.
type privateProcess struct {
	command *exec.Cmd
	stdin   io.WriteCloser
	stdout  *os.File
	exited  atomic.Bool
	done    chan struct{}
	read    chan struct{}
	// The reader keeps every line in an unbounded queue, so the child never
	// blocks on a full pipe while nobody is waiting for its output.
	mutex  sync.Mutex
	queue  []string
	eof    bool
	notify chan struct{}
}

func (p *privateProcess) push(line string, eof bool) {
	p.mutex.Lock()
	if eof {
		p.eof = true
	} else {
		p.queue = append(p.queue, line)
	}
	p.mutex.Unlock()
	select {
	case p.notify <- struct{}{}:
	default:
	}
}

// next returns the next queued line; ok is false when none is queued, and end
// is true once the output is closed and fully consumed.
func (p *privateProcess) next() (line string, ok, end bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if len(p.queue) > 0 {
		line, p.queue = p.queue[0], p.queue[1:]
		return line, true, false
	}
	return "", false, p.eof
}

func startPrivate(args ...string) (*privateProcess, error) {
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	command := exec.Command(args[0], args[1:]...)
	command.Stdout, command.Stderr = writer, writer
	stdin, err := command.StdinPipe()
	if err != nil {
		reader.Close()
		writer.Close()
		return nil, err
	}
	if err := command.Start(); err != nil {
		reader.Close()
		writer.Close()
		return nil, errors.New("process could not start")
	}
	writer.Close()
	process := &privateProcess{command: command, stdin: stdin, stdout: reader, notify: make(chan struct{}, 1),
		done: make(chan struct{}), read: make(chan struct{})}
	go func() {
		defer close(process.read)
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
		for scanner.Scan() {
			process.push(scanner.Text()+"\n", false)
		}
		process.push("", true)
	}()
	go func() {
		_ = command.Wait()
		process.exited.Store(true)
		close(process.done)
	}()
	return process, nil
}

// waitFor reads lines until parse returns a value, the process exits, or the timeout passes.
func (p *privateProcess) waitFor(parse func(string) (string, bool), description string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	lines := 0
	vmMarker, errorMarker := false, false
	failure := func(reason string) error {
		return Error(fmt.Sprintf("%s: %s (lines=%d, vm_marker=%s, error_marker=%s, process_running=%s); debug output was kept private",
			description, reason, lines, capitalBool(vmMarker), capitalBool(errorMarker), capitalBool(!p.exited.Load())))
	}
	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()
	for {
		if !time.Now().Before(deadline) {
			return "", failure("timed out")
		}
		line, ok, end := p.next()
		if end {
			return "", failure("process exited before it was ready")
		}
		if !ok {
			select {
			case <-p.notify:
				continue
			case <-timer.C:
				return "", failure("timed out")
			}
		}
		lines++
		vmMarker = vmMarker || strings.Contains(line, "The Dart VM service is listening on ")
		for _, marker := range []string{"[ERROR:", "Unhandled Exception", "FATAL"} {
			errorMarker = errorMarker || strings.Contains(line, marker)
		}
		if value, ok := parse(line); ok {
			return value, nil
		}
	}
}

// capitalBool keeps the established True/False spelling of the diagnostic flags.
func capitalBool(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

func (p *privateProcess) close() error {
	if !p.exited.Load() {
		_ = p.command.Process.Signal(syscall.SIGTERM)
		select {
		case <-p.done:
		case <-time.After(10 * time.Second):
			_ = p.command.Process.Kill()
			select {
			case <-p.done:
			case <-time.After(10 * time.Second):
				return errors.New("process did not stop")
			}
		}
	}
	select {
	case <-p.read:
	case <-time.After(2 * time.Second):
	}
	// A grandchild can still hold the pipe open; closing it ends the reader.
	_ = p.stdout.Close()
	if err := p.stdin.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		return err
	}
	return nil
}
