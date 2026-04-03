package process

import "errors"

// ErrProcessNotStarted is returned when trying to kill a process that hasn't been started.
var ErrProcessNotStarted = errors.New("process not started")

// ErrKillGroup is returned when killing the process group fails.
var ErrKillGroup = errors.New("failed to kill process group")
