package tools

import "errors"

// Shared sentinel errors for tools package.
var (
	// ErrPathParameterRequired is returned when the path parameter is missing or empty.
	ErrPathParameterRequired = errors.New("path parameter must be a non-empty string")
	// ErrContentParameterRequired is a sentinel error.
	ErrContentParameterRequired = errors.New("content parameter must be a string")
	// ErrOperationParameterRequired is a sentinel error.
	ErrOperationParameterRequired = errors.New("operation parameter is required")
	// ErrUnknownOperation is a sentinel error.
	ErrUnknownOperation = errors.New("unknown operation")
	// ErrKeyParameterRequiredForPut is a sentinel error.
	ErrKeyParameterRequiredForPut = errors.New("key parameter is required for put operation")
	// ErrValueParameterRequiredForPut is a sentinel error.
	ErrValueParameterRequiredForPut = errors.New("value parameter is required for put operation")
	// ErrKeyParameterRequiredForGet is a sentinel error.
	ErrKeyParameterRequiredForGet = errors.New("key parameter is required for get operation")
	// ErrKeyParameterRequiredForDelete is a sentinel error.
	ErrKeyParameterRequiredForDelete = errors.New("key parameter is required for delete operation")
	// ErrQueryParameterRequiredForSearch is a sentinel error.
	ErrQueryParameterRequiredForSearch = errors.New("query parameter is required for search operation")
	// ErrKeyNotFound is a sentinel error.
	ErrKeyNotFound = errors.New("key not found")
	// ErrKeyParameterRequiredForPin is a sentinel error.
	ErrKeyParameterRequiredForPin = errors.New("key parameter is required for pin operation")
	// ErrKeyParameterRequiredForUnpin is a sentinel error.
	ErrKeyParameterRequiredForUnpin = errors.New("key parameter is required for unpin operation")
	// ErrCommandCannotBeEmpty is returned when the command string is empty.
	ErrCommandCannotBeEmpty = errors.New("command cannot be empty")
)
