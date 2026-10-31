//go:build linux || freebsd || openbsd || netbsd

package clipboard

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// The fake helper is this test binary re-run with these set: mode picks how
// the fake clipboard behaves, received is where it writes the text it got.
const (
	fakeHelperModeEnv     = "CLIP_TEST_HELPER"
	fakeHelperReceivedEnv = "CLIP_TEST_HELPER_RECEIVED"
)

// TestMain turns the test binary into the helper when asked, so the
// startHelper/Serve handshake crosses a real process boundary without a
// display.
func TestMain(m *testing.M) {
	mode := os.Getenv(fakeHelperModeEnv)
	if mode == "" {
		os.Exit(m.Run())
	}
	os.Exit(runFakeHelper(mode))
}

// runFakeHelper stands in for `clip serve-clipboard <size>` with the
// clipboard faked per mode, so what gets tested is the handshake.
func runFakeHelper(mode string) int {
	if len(os.Args) < 2 {
		return 2
	}
	size, err := strconv.Atoi(os.Args[1])
	if err != nil {
		return 2
	}

	switch mode {
	case "crash":
		return 3
	case "ignore-stdin":
		// Neither drains stdin nor reports, so the parent has to give up.
		time.Sleep(time.Minute)
		return 0
	}

	var stored []byte
	writeClipboard = func(text []byte) (<-chan struct{}, error) {
		switch mode {
		case "fail":
			return nil, errors.New("fake clipboard refused the text")
		case "hang":
			time.Sleep(time.Minute)
		}
		if received := os.Getenv(fakeHelperReceivedEnv); received != "" {
			if err := os.WriteFile(received, text, 0o600); err != nil {
				return nil, err
			}
		}
		stored = text
		// Nothing replaces the fake clipboard, so Serve blocks like the real
		// helper until the test kills it. The timer bounds an orphan left by
		// an interrupted test run.
		replaced := make(chan struct{})
		time.AfterFunc(time.Minute, func() { close(replaced) })
		return replaced, nil
	}
	readClipboard = func(context.Context) ([]byte, error) {
		if mode == "corrupt" {
			return append(bytes.Clone(stored), '!'), nil
		}
		return stored, nil
	}

	if err := Serve(os.Stdin, size, os.NewFile(3, "ready")); err != nil {
		return 1
	}
	return 0
}

// fakeHelper builds the command startHelper runs under test: this binary in
// helper mode, with the byte count Copy would pass. Whatever is still running
// is killed when the test ends.
func fakeHelper(t *testing.T, mode, receivedPath string, text []byte) *exec.Cmd {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}
	cmd := exec.Command(exe, strconv.Itoa(len(text)))
	cmd.Env = append(os.Environ(), fakeHelperModeEnv+"="+mode, fakeHelperReceivedEnv+"="+receivedPath)
	t.Cleanup(func() {
		if cmd.Process != nil && cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})
	return cmd
}

// shortenHelperReadyTimeout makes the give-up paths testable in milliseconds.
func shortenHelperReadyTimeout(t *testing.T) {
	t.Helper()
	old := helperReadyTimeout
	helperReadyTimeout = 200 * time.Millisecond
	t.Cleanup(func() { helperReadyTimeout = old })
}

// assertReaped checks that a failed helper was waited on, so it is neither a
// zombie nor an orphan.
func assertReaped(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if cmd.ProcessState == nil {
		t.Error("helper was not reaped after the failure")
	}
}

func TestStartHelperLeavesItServing(t *testing.T) {
	received := filepath.Join(t.TempDir(), "received")
	text := []byte("hello clip")
	cmd := fakeHelper(t, "ok", received, text)
	if err := startHelper(cmd, text); err != nil {
		t.Fatalf("startHelper() error = %v, want nil", err)
	}
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		t.Errorf("helper is not running after startHelper() returned: %v", err)
	}
	if got, err := os.ReadFile(received); err != nil || !bytes.Equal(got, text) {
		t.Errorf("helper received %q (err %v), want %q", got, err, text)
	}
}

func TestStartHelperReportsTheHelpersError(t *testing.T) {
	text := []byte("x")
	cmd := fakeHelper(t, "fail", "", text)
	err := startHelper(cmd, text)
	if err == nil || err.Error() != "fake clipboard refused the text" {
		t.Errorf("startHelper() error = %v, want the helper's own message", err)
	}
	assertReaped(t, cmd)
}

func TestStartHelperReportsAMismatchedReadBack(t *testing.T) {
	text := []byte("x")
	cmd := fakeHelper(t, "corrupt", "", text)
	err := startHelper(cmd, text)
	if err == nil || !strings.Contains(err.Error(), "clipboard holds 2 bytes after copying 1") {
		t.Errorf("startHelper() error = %v, want a read-back mismatch", err)
	}
	assertReaped(t, cmd)
}

func TestStartHelperReportsASilentExit(t *testing.T) {
	text := []byte("x")
	cmd := fakeHelper(t, "crash", "", text)
	err := startHelper(cmd, text)
	if err == nil || !strings.Contains(err.Error(), "exited without reporting (exit status 3)") {
		t.Errorf("startHelper() error = %v, want an exited-without-reporting error with the status", err)
	}
	assertReaped(t, cmd)
}

func TestStartHelperGivesUpOnAHungHelper(t *testing.T) {
	shortenHelperReadyTimeout(t)
	text := []byte("x")
	cmd := fakeHelper(t, "hang", "", text)
	err := startHelper(cmd, text)
	if err == nil || !strings.Contains(err.Error(), "did not report") {
		t.Errorf("startHelper() error = %v, want a timeout error", err)
	}
	assertReaped(t, cmd)
}

func TestStartHelperGivesUpWhenStdinIsIgnored(t *testing.T) {
	shortenHelperReadyTimeout(t)
	// Larger than any pipe buffer, so the write cannot finish on its own.
	text := bytes.Repeat([]byte("x"), 1<<20)
	cmd := fakeHelper(t, "ignore-stdin", "", text)
	err := startHelper(cmd, text)
	if err == nil || !strings.Contains(err.Error(), "did not report") {
		t.Errorf("startHelper() error = %v, want a timeout error", err)
	}
	assertReaped(t, cmd)
}

// nopCloser lets a bytes.Buffer stand in for the readiness pipe.
type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }

// fakeClipboard swaps in test doubles for the library calls for one test.
func fakeClipboard(t *testing.T, write func([]byte) (<-chan struct{}, error), read func(context.Context) ([]byte, error)) {
	t.Helper()
	oldWrite, oldRead := writeClipboard, readClipboard
	writeClipboard, readClipboard = write, read
	t.Cleanup(func() { writeClipboard, readClipboard = oldWrite, oldRead })
}

func TestServeRefusesAShortRead(t *testing.T) {
	fakeClipboard(t,
		func([]byte) (<-chan struct{}, error) { t.Fatal("wrote a short read to the clipboard"); return nil, nil },
		func(context.Context) ([]byte, error) { return nil, nil },
	)

	var report bytes.Buffer
	err := Serve(strings.NewReader("abc"), 5, nopCloser{&report})
	if err == nil || !strings.Contains(err.Error(), "received 3 of 5 bytes") {
		t.Errorf("Serve() error = %v, want a short-read error", err)
	}
	if !strings.HasPrefix(report.String(), "received 3 of 5 bytes") {
		t.Errorf("Serve() reported %q, want the short-read error", report.String())
	}
}

func TestServeReportsALostClipboard(t *testing.T) {
	replaced := make(chan struct{})
	close(replaced)
	fakeClipboard(t,
		func([]byte) (<-chan struct{}, error) { return replaced, nil },
		func(ctx context.Context) ([]byte, error) { <-ctx.Done(); return nil, ctx.Err() },
	)

	var report bytes.Buffer
	err := Serve(strings.NewReader("x"), 1, nopCloser{&report})
	if err == nil || !strings.Contains(err.Error(), "lost the clipboard") {
		t.Errorf("Serve() error = %v, want a lost-clipboard error", err)
	}
	if !strings.HasPrefix(report.String(), "lost the clipboard") {
		t.Errorf("Serve() reported %q, want the lost-clipboard error", report.String())
	}
}

func TestServeNamesTheX11LimitForABigPayload(t *testing.T) {
	fakeClipboard(t,
		func([]byte) (<-chan struct{}, error) { return make(chan struct{}), nil },
		func(context.Context) ([]byte, error) { return nil, nil },
	)

	text := bytes.Repeat([]byte("x"), x11MaxPayload+1)
	err := Serve(bytes.NewReader(text), len(text), nil)
	if err == nil || !strings.Contains(err.Error(), "X11 carries at most 262116 bytes per transfer") {
		t.Errorf("Serve() error = %v, want the X11 limit named", err)
	}
}

func TestPasteWithoutADisplayIsUnavailable(t *testing.T) {
	if os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != "" {
		t.Skip("a display is available")
	}
	if _, err := Paste(); !errors.Is(err, errUnavailable) {
		t.Errorf("Paste() error = %v, want errUnavailable", err)
	}
}
