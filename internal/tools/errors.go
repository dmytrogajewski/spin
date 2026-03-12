package tools

import "errors"

// Shared sentinel errors for tools package.
var (
	// Common parameter errors.
	ErrPathParameterRequired          = errors.New("path parameter must be a non-empty string")
	ErrContentParameterRequired       = errors.New("content parameter must be a string")
	ErrOperationParameterRequired     = errors.New("operation parameter is required")
	ErrUnknownOperation               = errors.New("unknown operation")
	ErrKeyParameterRequiredForPut     = errors.New("key parameter is required for put operation")
	ErrValueParameterRequiredForPut   = errors.New("value parameter is required for put operation")
	ErrKeyParameterRequiredForGet     = errors.New("key parameter is required for get operation")
	ErrKeyParameterRequiredForDelete  = errors.New("key parameter is required for delete operation")
	ErrQueryParameterRequiredForSearch = errors.New("query parameter is required for search operation")
	ErrKeyNotFound                    = errors.New("key not found")
	ErrKeyParameterRequiredForPin     = errors.New("key parameter is required for pin operation")
	ErrKeyParameterRequiredForUnpin   = errors.New("key parameter is required for unpin operation")
)
