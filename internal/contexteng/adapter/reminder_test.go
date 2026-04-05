package adapter_test

import (
	"context"
	"testing"

	"github.com/dmytrogajewski/spin/internal/contexteng/adapter"
	"github.com/dmytrogajewski/spin/internal/contexteng/reminder"
	"github.com/dmytrogajewski/spin/internal/message"
)

// stubDetector is a test detector that fires when Turn >= threshold.
type stubDetector struct {
	name      string
	threshold int
	maxFires  int
}

func (d *stubDetector) Name() string {
	return d.name
}

func (d *stubDetector) Check(ctx reminder.CheckContext) bool {
	return ctx.Turn >= d.threshold
}

func (d *stubDetector) MaxFires() int {
	return d.maxFires
}

func TestReminderAdapter_InjectReminders_NoFire(t *testing.T) {
	t.Parallel()

	det := &stubDetector{name: "test", threshold: 5, maxFires: 1}
	templates := map[string]string{"test": "reminder text"}
	inj := reminder.NewInjector([]reminder.Detector{det}, templates)
	adapt := adapter.NewReminderAdapter(inj)

	msgs := []message.Message{
		{Role: message.RoleUser, Content: "hello"},
	}

	result := adapt.InjectReminders(context.Background(), msgs, 0)
	if len(result) != 0 {
		t.Errorf("expected 0 reminders, got %d", len(result))
	}
}

func TestReminderAdapter_InjectReminders_Fires(t *testing.T) {
	t.Parallel()

	det := &stubDetector{name: "test", threshold: 3, maxFires: 2}
	templates := map[string]string{"test": "you should do X"}
	inj := reminder.NewInjector([]reminder.Detector{det}, templates)
	adapt := adapter.NewReminderAdapter(inj)

	msgs := []message.Message{
		{Role: message.RoleUser, Content: "hello"},
	}

	result := adapt.InjectReminders(context.Background(), msgs, 5)
	if len(result) != 1 {
		t.Fatalf("expected 1 reminder, got %d", len(result))
	}

	if result[0].Content != "you should do X" {
		t.Errorf("unexpected content: %s", result[0].Content)
	}

	if result[0].Role != message.RoleUser {
		t.Errorf("expected role user, got %s", result[0].Role)
	}
}

func TestReminderAdapter_NilInjector(t *testing.T) {
	t.Parallel()

	adapt := adapter.NewReminderAdapter(nil)

	result := adapt.InjectReminders(context.Background(), nil, 0)
	if len(result) != 0 {
		t.Errorf("expected 0 reminders for nil injector, got %d", len(result))
	}
}

func TestReminderAdapter_BuildsCheckContext(t *testing.T) {
	t.Parallel()

	// Detector that checks Turn field is properly set.
	det := &stubDetector{name: "turn_check", threshold: 7, maxFires: 1}
	templates := map[string]string{"turn_check": "turn is 7+"}
	inj := reminder.NewInjector([]reminder.Detector{det}, templates)
	adapt := adapter.NewReminderAdapter(inj)

	msgs := []message.Message{
		{Role: message.RoleAssistant, Content: "thinking"},
	}

	// Turn 6 should not fire.
	result := adapt.InjectReminders(context.Background(), msgs, 6)
	if len(result) != 0 {
		t.Errorf("expected 0 reminders at turn 6, got %d", len(result))
	}

	// Turn 7 should fire.
	result = adapt.InjectReminders(context.Background(), msgs, 7)
	if len(result) != 1 {
		t.Fatalf("expected 1 reminder at turn 7, got %d", len(result))
	}
}
