package closer

import (
	"fmt"
	"io"
)

type Func func() error

func (f Func) Close() error {
	return f()
}

type MultiCloser []io.Closer

func (c *MultiCloser) AddCloser(closer io.Closer) {
	*c = append(*c, closer)
}

func (c *MultiCloser) Close() (err error) {
	for _, closer := range *c {
		err2 := closer.Close()
		if err2 != nil {
			if err != nil {
				err = fmt.Errorf("%w, %w", err, err2)
			} else {
				err = err2
			}
		}
	}
	return err
}
