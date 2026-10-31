//go:build !(linux || freebsd || openbsd || netbsd)

package clipboard

// Copy puts text on the clipboard. macOS and Windows hold the data
// themselves, so nothing has to stay alive.
func Copy(text []byte) error {
	_, err := writeClipboard(text)
	return err
}
