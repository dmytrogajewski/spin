package apperr

// Journey: specs/journeys/JOURNEY-R22.md.

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	errFirst  = errors.New("first error")
	errSecond = errors.New("second error")
)

func TestErrorList_empty_returns_nil(t *testing.T) {
	t.Parallel()

	var list ErrorList

	require.False(t, list.HasErrors())
	require.NoError(t, list.Err())
}

func TestErrorList_add_nil_skipped(t *testing.T) {
	t.Parallel()

	var list ErrorList

	list.Add(nil)
	list.Add(nil)

	require.False(t, list.HasErrors())
	require.NoError(t, list.Err())
}

func TestErrorList_single_error(t *testing.T) {
	t.Parallel()

	var list ErrorList

	list.Add(errFirst)

	require.True(t, list.HasErrors())

	err := list.Err()
	require.Error(t, err)
	require.ErrorIs(t, err, errFirst)
}

func TestErrorList_multiple_errors(t *testing.T) {
	t.Parallel()

	var list ErrorList

	list.Add(errFirst)
	list.Add(errSecond)

	require.True(t, list.HasErrors())

	err := list.Err()
	require.Error(t, err)
	require.ErrorIs(t, err, errFirst)
	require.ErrorIs(t, err, errSecond)
}

func TestErrorList_nil_errors_filtered(t *testing.T) {
	t.Parallel()

	var list ErrorList

	list.Add(nil)
	list.Add(errFirst)
	list.Add(nil)
	list.Add(errSecond)
	list.Add(nil)

	require.True(t, list.HasErrors())

	err := list.Err()
	require.ErrorIs(t, err, errFirst)
	require.ErrorIs(t, err, errSecond)
}
