//go:build linux || freebsd || openbsd || netbsd

package clipboard

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

// Copy hands text to a detached helper (clip re-executed with
// [ServeCommandName] and the byte count) that keeps serving pastes after clip
// exits, like wl-copy does. It returns once the helper reports the text is on
// the clipboard.
func Copy(text []byte) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not locate the clip binary for the clipboard helper: %w", err)
	}
	return startHelper(exec.Command(exe, ServeCommandName, strconv.Itoa(len(text))), text)
}

// startHelper runs cmd detached, feeds it text on stdin, and waits only for
// its report on fd 3. Success leaves it running; any failure kills and reaps
// it.
func startHelper(cmd *exec.Cmd, text []byte) error {
	readyReader, readyWriter, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("could not create the clipboard helper's readiness pipe: %w", err)
	}
	defer readyReader.Close()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		readyWriter.Close()
		return fmt.Errorf("could not create the clipboard helper's stdin pipe: %w", err)
	}
	cmd.ExtraFiles = []*os.File{readyWriter}
	// Root, so the helper pins no mount clip happened to be run from.
	cmd.Dir = "/"
	detachFromTerminal(cmd)

	if err := cmd.Start(); err != nil {
		readyWriter.Close()
		return fmt.Errorf("could not start the clipboard helper: %w", err)
	}
	// Only the helper may hold the write end, or the read below never sees EOF.
	readyWriter.Close()

	// The write shares the timeout with the read: a helper that never drains
	// stdin would otherwise block the write forever once text outgrows the
	// pipe buffer.
	type readiness struct {
		report []byte
		err    error
	}
	reported := make(chan readiness, 1)
	go func() {
		_, writeErr := stdin.Write(text)
		closeErr := stdin.Close()
		report, readErr := io.ReadAll(readyReader)
		reported <- readiness{report, errors.Join(writeErr, closeErr, readErr)}
	}()

	var ready readiness
	select {
	case ready = <-reported:
	case <-time.After(helperReadyTimeout):
		killHelper(cmd)
		return fmt.Errorf("clipboard helper did not report within %s", helperReadyTimeout)
	}

	switch report := bytes.TrimSpace(ready.report); {
	case bytes.Equal(report, []byte("ok")):
		return nil
	case len(report) > 0:
		killHelper(cmd)
		return errors.New(string(report))
	default:
		err := errors.New("clipboard helper exited without reporting")
		if state := killHelper(cmd); state != nil && state.Exited() {
			err = fmt.Errorf("%w (%v)", err, state)
		}
		if ready.err != nil {
			err = fmt.Errorf("%w: %w", err, ready.err)
		}
		return err
	}
}

// detachFromTerminal makes cmd a session leader so job control and SIGHUP
// don't reach it after clip exits.
func detachFromTerminal(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

// killHelper kills a helper that didn't come up, reaps it, and returns how it
// ended.
func killHelper(cmd *exec.Cmd) *os.ProcessState {
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	return cmd.ProcessState
}
