package adapter

import (
	"context"

	"github.com/dmytrogajewski/spin/internal/contexteng/reminder"
	"github.com/dmytrogajewski/spin/internal/message"
)

// ReminderAdapter adapts reminder.Injector to the harness.ReminderInjector interface.
// It builds a CheckContext from the provided messages and turn number.
type ReminderAdapter struct {
	inner *reminder.Injector
}

// NewReminderAdapter creates a ReminderAdapter wrapping the given injector.
// A nil injector produces a no-op adapter that returns no reminders.
func NewReminderAdapter(inj *reminder.Injector) *ReminderAdapter {
	return &ReminderAdapter{inner: inj}
}

// InjectReminders builds a CheckContext from messages and turn, then delegates
// to the inner injector. The [context.Context] parameter is accepted for interface
// compliance but not used by the current reminder implementation.
func (a *ReminderAdapter) InjectReminders(
	_ context.Context, messages []message.Message, turn int,
) []message.Message {
	if a.inner == nil {
		return nil
	}

	checkCtx := reminder.CheckContext{
		Turn:     turn,
		Messages: messages,
	}

	return a.inner.Inject(checkCtx)
}
