# Clipboard backend: golang.design/x/clipboard

`clip copy` and `clip paste` move text through `github.com/atotto/clipboard`.
This change replaces it with `golang.design/x/clipboard` behind a small
`internal/clipboard` package and makes `clip copy` leave a detached process
owning the clipboard on X11 and Wayland.

## The problem

`atotto/clipboard` picks its Linux backend once, in `init()`: when
`WAYLAND_DISPLAY` is set it shells out to `wl-copy` and `wl-paste`, otherwise
to `xclip` or `xsel`. `wl-copy` and `wl-paste` need the Wayland data-control
protocol (`ext-data-control-v1` or `wlr-data-control-unstable-v1`). GNOME's
Mutter does not implement either (no source in the tree through 51.1; the
upstream issue asking for it was closed unmerged), so under a GNOME Wayland
session `clip copy` and `clip paste` hang or fail depending on the `wl-clipboard`
version, while `xclip` against XWayland would have worked.

## Before and after

| Area | Before | After | Why |
|---|---|---|---|
| Library | `atotto/clipboard`, exec of `wl-copy`/`wl-paste`/`xclip`/`xsel` | `golang.design/x/clipboard`, pure-Go Wayland and X11 protocol clients, no Cgo | no runtime dependency on external tools; backend chosen per call site instead of in `init()` |
| Backend choice on Linux | `WAYLAND_DISPLAY` set means Wayland tools, whatever the compositor | Wayland data-control when the compositor advertises it, otherwise X11 (`DISPLAY`), which under GNOME means XWayland | GNOME bridges XWayland selections to its clipboard; data-control it does not have |
| Lifetime of a copy | the exec'd `wl-copy` forked and kept serving; `xclip`/`xsel` likewise | `clip copy` starts `clip serve-clipboard` detached from the terminal (an argument `main` handles ahead of kingpin, so it is absent from help and completion) and exits once that process reports the text is on the clipboard; the helper exits when another application replaces the clipboard or the display goes away | on X11 and Wayland the writing process serves paste requests; the data is gone when it exits unless a clipboard manager kept a copy |
| macOS, Windows | library calls | library calls, no helper | the system holds the data |
| No display | an error from the exec'd tool, or a hang | `clipboard unavailable: no X11 display or Wayland data-control compositor reachable (check DISPLAY and WAYLAND_DISPLAY)`, exit 1 | one line instead of the library's multi-line Xvfb tutorial |
| Empty clipboard on paste | empty line | empty line | unchanged |
| Helper failure | n/a | `clip copy` prints the helper's error and exits 1; the helper is reaped | the helper reports before `clip copy` exits, so nothing fails silently in the background |
| Payload over X11's single-request limit (262116 bytes) | `xclip`/`xsel` switched to INCR, so it worked | the helper reads the copy back before reporting; `clip copy` exits 1 naming the limit. Wayland data-control is unaffected | the library sends the whole selection in one core request (no BIG-REQUESTS, no INCR) and takes the selection before any data moves, so the failure only shows on paste; it does not expose which backend it chose, so reading back is the one check that works on both |

## Architecture

```
internal/clipboard/
  clipboard.go       Paste, Serve, ServeCommandName, the library wrapper
  copy_detached.go   Copy via the helper process (linux, freebsd, openbsd, netbsd)
  copy_direct.go     Copy via a direct library call (everything else)
cmd/clip/main.go       runs serve-clipboard ahead of kingpin
```

The platform split is a build constraint rather than a runtime check, so each
file carries a real implementation and nothing is compiled that a platform
cannot use (`syscall.SysProcAttr.Setsid` does not exist on Windows).

`Copy(text)` on X11 and Wayland runs `os.Executable()` with the arguments
`serve-clipboard <byte count>`, the text on the child's stdin and a pipe on the
child's file descriptor 3. It waits only for that pipe: the helper writes `ok` after the
library's `Write` has returned (the data is on the clipboard at that point) and
closes the pipe, or writes its error and exits. `Copy` returns `nil` on `ok`,
the helper's own message on anything else, and gives up after a bounded wait,
killing the helper, if nothing arrives; the stdin write runs under the same
bound, and a helper that exits without reporting shows its exit status. The
helper's stdout and stderr are `/dev/null`, its working directory is `/`, and
it is a session leader, so `$(clip copy ...)`, a closing terminal, and
unmounting the directory `clip` ran from are all unaffected.

`Serve(stdin, size, fd3)` is the helper: read exactly `size` bytes (fewer is an
error, so a parent killed mid-write never serves a truncated copy), `Init`,
`Write`, read the clipboard back and compare, report, return the read-back's
memory to the OS, then block on the channel `Write` returns, which fires when
the clipboard is replaced or the connection ends. The read-back is bounded to
half the parent's wait so the helper's own error wins the race. The exact-size
read and the release after verification keep the helper's resident memory at
the payload's size for as long as it serves. `main` runs `serve-clipboard`
before kingpin parses, so the helper never creates first-run files.

`Paste()` maps the library's `ErrNoData` to a nil result and its
initialization failure to the one-line unavailable error.

## Tests

| Test | Pins |
|---|---|
| `internal/clipboard` `TestStartHelper*` | the handshake across a real process boundary with the test binary as the helper: `ok` leaves the helper running with the text intact; the helper's error, a read-back mismatch, and a silent exit (with its status) are `Copy`'s error; a helper that never reports and one that never drains stdin are killed after the bounded wait; every failure leaves no process behind |
| `internal/clipboard` `TestServe*` | the branches that need no process: a short read refused, the clipboard replaced before verification, and the X11 limit named for an oversized payload |
| `internal/clipboard` `TestPasteWithoutADisplayIsUnavailable` | the one-line error when neither display variable is set (skipped when a display is available) |

The live paths (X11 selection ownership, SelectionClear ending the helper, the
XWayland fallback when `WAYLAND_DISPLAY` names no usable compositor, and the
Wayland data-control path) are not unit-tested.

## Out of scope

- Images and other formats: `clip` is a text tool.
- A persistence option (keeping the helper after another application takes
  the clipboard, or a time limit on it): the helper mirrors `wl-copy`.
- Primary selection support.
- Large X11 transfers (BIG-REQUESTS, INCR): a change to the library's X11
  client; `clip` detects the failure and names the limit.
