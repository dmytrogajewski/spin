package tools

import "errors"

// Sentinel errors for tools package (unexported — internal use only).
var (
	errPathParameterRequired        = errors.New("path parameter must be a non-empty string")
	errOperationParameterRequired   = errors.New("operation parameter is required")
	errUnknownOperation             = errors.New("unknown operation")
	errKeyParameterRequiredForPut   = errors.New("key parameter is required for put operation")
	errValueParameterRequiredForPut = errors.New("value parameter is required for put operation")
	errKeyParameterRequiredForGet   = errors.New("key parameter is required for get operation")
	errKeyParameterRequiredForDel   = errors.New("key parameter is required for delete operation")
	errQueryParamRequiredForSearch  = errors.New("query parameter is required for search operation")
	errKeyNotFound                  = errors.New("key not found")
	errKeyParameterRequiredForPin   = errors.New("key parameter is required for pin operation")
	errKeyParameterRequiredForUnpin = errors.New("key parameter is required for unpin operation")
	errCommandCannotBeEmpty         = errors.New("command cannot be empty")
	errExecutorNotConfigured        = errors.New("executor not configured")
	errCommandParameterRequired     = errors.New("command parameter is required")
	errValidatorNotConfigured       = errors.New("validator not configured")
	errQueryParameterRequired       = errors.New("query parameter must be a non-empty string")
	errTaskIDParameterRequired      = errors.New("task_id parameter is required")
	errHTTPError                    = errors.New("HTTP error response")
)
