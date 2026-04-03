package validate

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

var errCustomValidation = errors.New("custom validation failed")

func TestChain_Required(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		err := NewChain().Required("name", "alice").Err()
		require.NoError(t, err)
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		err := NewChain().Required("name", "").Err()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrRequired)
		require.Contains(t, err.Error(), "name")
	})

	t.Run("whitespace_only", func(t *testing.T) {
		t.Parallel()

		err := NewChain().Required("name", "   ").Err()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrRequired)
	})
}

func TestChain_Positive(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		err := NewChain().Positive("count", 5).Err()
		require.NoError(t, err)
	})

	t.Run("zero", func(t *testing.T) {
		t.Parallel()

		err := NewChain().Positive("count", 0).Err()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrPositive)
	})

	t.Run("negative", func(t *testing.T) {
		t.Parallel()

		err := NewChain().Positive("count", -1).Err()
		require.Error(t, err)
	})
}

func TestChain_InRange(t *testing.T) {
	t.Parallel()

	t.Run("within", func(t *testing.T) {
		t.Parallel()

		err := NewChain().InRange("port", 8080, 1, 65535).Err()
		require.NoError(t, err)
	})

	t.Run("below", func(t *testing.T) {
		t.Parallel()

		err := NewChain().InRange("port", 0, 1, 65535).Err()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrOutOfRange)
	})

	t.Run("above", func(t *testing.T) {
		t.Parallel()

		err := NewChain().InRange("port", 99999, 1, 65535).Err()
		require.Error(t, err)
	})

	t.Run("at_boundary", func(t *testing.T) {
		t.Parallel()

		err := NewChain().InRange("port", 1, 1, 65535).Err()
		require.NoError(t, err)
	})
}

func TestChain_OneOf(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		err := NewChain().OneOf("mode", "fast", "fast", "slow", "auto").Err()
		require.NoError(t, err)
	})

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()

		err := NewChain().OneOf("mode", "turbo", "fast", "slow").Err()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotOneOf)
	})
}

func TestChain_MultipleErrors(t *testing.T) {
	t.Parallel()

	err := NewChain().
		Required("name", "").
		Positive("count", -1).
		Required("email", "").
		Err()

	require.Error(t, err)
	require.ErrorIs(t, err, ErrRequired)
	require.ErrorIs(t, err, ErrPositive)
}

func TestChain_NoErrors(t *testing.T) {
	t.Parallel()

	err := NewChain().
		Required("name", "alice").
		Positive("count", 5).
		Err()

	require.NoError(t, err)
}

func TestChain_HasErrors(t *testing.T) {
	t.Parallel()

	c := NewChain()
	require.False(t, c.HasErrors())

	c.Required("name", "")
	require.True(t, c.HasErrors())
}

func TestChain_Check(t *testing.T) {
	t.Parallel()

	t.Run("nil_error", func(t *testing.T) {
		t.Parallel()

		err := NewChain().Check(nil).Err()
		require.NoError(t, err)
	})

	t.Run("non_nil_error", func(t *testing.T) {
		t.Parallel()

		err := NewChain().Check(errCustomValidation).Err()
		require.ErrorIs(t, err, errCustomValidation)
	})
}

func TestChain_InRangeFloat(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		err := NewChain().InRangeFloat("temp", 0.7, 0.0, 1.0).Err()
		require.NoError(t, err)
	})

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()

		err := NewChain().InRangeFloat("temp", 1.5, 0.0, 1.0).Err()
		require.Error(t, err)
	})
}

func TestChain_MinLength(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		err := NewChain().MinLength("password", "secret123", 8).Err()
		require.NoError(t, err)
	})

	t.Run("too_short", func(t *testing.T) {
		t.Parallel()

		err := NewChain().MinLength("password", "abc", 8).Err()
		require.Error(t, err)
		require.ErrorIs(t, err, ErrMinLength)
	})
}

func TestChain_NonNegative(t *testing.T) {
	t.Parallel()

	t.Run("zero_ok", func(t *testing.T) {
		t.Parallel()

		err := NewChain().NonNegative("offset", 0).Err()
		require.NoError(t, err)
	})

	t.Run("negative", func(t *testing.T) {
		t.Parallel()

		err := NewChain().NonNegative("offset", -1).Err()
		require.Error(t, err)
	})
}
