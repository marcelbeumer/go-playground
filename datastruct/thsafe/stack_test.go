package thsafe

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStackPushValues(t *testing.T) {
	t.Run("Push() sets new value as Top()", func(t *testing.T) {
		s := Stack[int]{}

		top, ok := s.Top()
		require.Equal(t, top, 0) // zero value
		require.Equal(t, ok, false)

		s.Push(10)
		top, ok = s.Top()
		require.Equal(t, top, 10)
		require.Equal(t, ok, true)
	})
}
