// Package clipboard puts text on the system clipboard and reads it back via
// golang.design/x/clipboard: pure Go, Wayland data-control when the
// compositor offers it, X11 otherwise. A Wayland compositor without
// data-control (GNOME's Mutter) is reached through XWayland, which it bridges
// to its own clipboard.
//
// Copy has two implementations. On X11 and Wayland the process that wrote the
// clipboard answers every paste, and the data goes with it when it exits, so
// Copy hands the text to a detached helper that outlives clip
// (copy_detached.go). macOS and Windows hold the data themselves, so Copy
// writes it directly (copy_direct.go).
package clipboard

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"runtime/debug"
	"time"

	sys "golang.design/x/clipboard"
)

// ServeCommandName is the argument the detached Copy re-executes clip with,
// followed by the byte count it is about to send. main runs Serve for it
// ahead of kingpin, so it stays out of help and completion and never touches
// config.
const ServeCommandName = "serve-clipboard"

// helperReadyTimeout is how long Copy waits for the helper's report before
// killing it. A var so tests can shorten it.
var helperReadyTimeout = 10 * time.Second

// x11MaxPayload is the most one core X11 request carries: a 16-bit length in
// 4-byte units, less the ChangeProperty header. The library sends the whole
// selection in one such request, with neither BIG-REQUESTS nor INCR.
const x11MaxPayload = 0xffff*4 - 24

// errUnavailable replaces the library's init error, which is a multi-line
// tutorial on installing Xvfb.
var errUnavailable = errors.New("clipboard unavailable: no X11 display or Wayland data-control compositor reachable (check DISPLAY and WAYLAND_DISPLAY)")

// writeClipboard puts text on the clipboard and returns the channel that fires
// when something replaces it; readClipboard returns the clipboard's text, nil
// when it holds none. Vars so the test helper can run Serve without a display.
var (
	writeClipboard = func(text []byte) (<-chan struct{}, error) {
		if err := sys.Init(); err != nil {
			return nil, errUnavailable
		}
		return sys.Write(context.Background(), sys.FmtText, text)
	}

	readClipboard = func(ctx context.Context) ([]byte, error) {
		if err := sys.Init(); err != nil {
			return nil, errUnavailable
		}
		text, err := sys.Read(ctx, sys.FmtText)
		if errors.Is(err, sys.ErrNoData) {
			return nil, nil
		}
		return text, err
	}
)

// Paste returns the clipboard's text. A clipboard holding no text yields nil.
func Paste() ([]byte, error) {
	return readClipboard(context.Background())
}

// Serve is the helper side of the detached Copy: read exactly size bytes of
// text, put them on the clipboard, read them back, report "ok" or the error
// on ready, then serve pastes until something replaces the clipboard. The
// exact size keeps the buffer at the payload's size and makes a short read
// (the parent died mid-write) an error rather than a truncated copy.
func Serve(text io.Reader, size int, ready io.WriteCloser) error {
	data := make([]byte, size)
	n, err := io.ReadFull(text, data)
	if err != nil {
		err = fmt.Errorf("received %d of %d bytes to copy: %w", n, size, err)
		reportReady(ready, err)
		return err
	}

	replaced, err := writeClipboard(data)
	if err == nil {
		err = verifyCopy(data, replaced)
	}
	if err != nil {
		reportReady(ready, err)
		return err
	}
	reportReady(ready, nil)
	// The helper lives until something replaces the clipboard; the read-back's
	// buffer is garbage now and should not sit in its heap that whole time.
	debug.FreeOSMemory()

	if replaced != nil {
		<-replaced
	}
	return nil
}

// verifyCopy reads the clipboard back and checks it holds want. Taking the
// selection succeeds before any data moves, so a transfer the backend can't
// make (X11 past x11MaxPayload) would otherwise surface as an empty paste
// after `clip copy` reported success. The read is bounded to half the parent's
// wait so the helper's own error reaches it first.
func verifyCopy(want []byte, replaced <-chan struct{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), helperReadyTimeout/2)
	defer cancel()

	type readback struct {
		got []byte
		err error
	}
	// Captured here rather than in the goroutine so a test swapping the var
	// back after Serve returns never races the read.
	readBack := readClipboard
	results := make(chan readback, 1)
	go func() {
		got, err := readBack(ctx)
		results <- readback{got, err}
	}()

	var err error
	select {
	case r := <-results:
		switch {
		case r.err != nil:
			err = fmt.Errorf("could not read the copied text back: %w", r.err)
		case !bytes.Equal(r.got, want):
			err = fmt.Errorf("clipboard holds %d bytes after copying %d", len(r.got), len(want))
		}
	case <-replaced:
		err = errors.New("lost the clipboard before the copy could be verified")
	case <-ctx.Done():
		err = errors.New("reading the copied text back timed out")
	}
	if err != nil && len(want) > x11MaxPayload {
		err = fmt.Errorf("%w (X11 carries at most %d bytes per transfer)", err, x11MaxPayload)
	}
	return err
}

// reportReady writes the one-line result to ready and closes it so the
// parent's read returns. A failed write has nowhere else to go.
func reportReady(ready io.WriteCloser, err error) {
	if ready == nil {
		return
	}
	if err == nil {
		_, _ = io.WriteString(ready, "ok\n")
	} else {
		_, _ = io.WriteString(ready, err.Error()+"\n")
	}
	_ = ready.Close()
}
