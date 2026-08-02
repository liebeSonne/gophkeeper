package closer

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCloser struct {
	called bool
	err    error
}

func (c *testCloser) Close() error {
	c.called = true
	return c.err
}

func TestFunc_Close(t *testing.T) {
	testCases := []struct {
		name       string
		fn         Func
		wantErr    bool
		errMatches bool
	}{
		{
			name:    "func returns nil",
			fn:      Func(func() error { return nil }),
			wantErr: false,
		},
		{
			name:    "func returns error",
			fn:      Func(func() error { return errors.New("close error") }),
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.fn.Close()

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMultiCloser_AddCloser(t *testing.T) {
	testCases := []struct {
		name       string
		closers    int
		wantLength int
	}{
		{
			name:       "add single closer",
			closers:    1,
			wantLength: 1,
		},
		{
			name:       "add multiple closers",
			closers:    5,
			wantLength: 5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mc := MultiCloser{}

			for i := 0; i < tc.closers; i++ {
				mc.AddCloser(&testCloser{})
			}

			assert.Len(t, mc, tc.wantLength)
		})
	}
}

func TestMultiCloser_Close(t *testing.T) {
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")

	testCases := []struct {
		name       string
		closers    []io.Closer
		wantErr    bool
		errMatches error
	}{
		{
			name:    "empty closers",
			closers: []io.Closer{},
			wantErr: false,
		},
		{
			name: "single closer succeeds",
			closers: []io.Closer{
				&testCloser{err: nil},
			},
			wantErr: false,
		},
		{
			name: "all closers succeed",
			closers: []io.Closer{
				&testCloser{err: nil},
				&testCloser{err: nil},
				&testCloser{err: nil},
			},
			wantErr: false,
		},
		{
			name: "one closer fails",
			closers: []io.Closer{
				&testCloser{err: nil},
				&testCloser{err: err1},
				&testCloser{err: nil},
			},
			wantErr:    true,
			errMatches: err1,
		},
		{
			name: "first closer fails",
			closers: []io.Closer{
				&testCloser{err: err1},
				&testCloser{err: nil},
			},
			wantErr:    true,
			errMatches: err1,
		},
		{
			name: "multiple closers fail",
			closers: []io.Closer{
				&testCloser{err: err1},
				&testCloser{err: err2},
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mc := MultiCloser{}
			for _, c := range tc.closers {
				mc.AddCloser(c)
			}

			err := mc.Close()

			if tc.wantErr {
				assert.Error(t, err)
				if tc.errMatches != nil {
					assert.ErrorIs(t, err, tc.errMatches)
				}
			} else {
				assert.NoError(t, err)
			}

			for i, c := range tc.closers {
				if tc, ok := c.(*testCloser); ok {
					assert.True(t, tc.called, "closer %d should have been called", i)
				}
			}
		})
	}
}
