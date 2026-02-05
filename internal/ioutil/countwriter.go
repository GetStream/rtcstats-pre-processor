package ioutil

import "io"

// CountWriter wraps an io.Writer and counts bytes written.
type CountWriter struct {
	W     io.Writer
	Count int64
}

func (cw *CountWriter) Write(p []byte) (int, error) {
	n, err := cw.W.Write(p)
	cw.Count += int64(n)
	return n, err
}
