package reminder

import "github.com/dmytrogajewski/spin/internal/message"

// Injector evaluates detectors and produces reminder messages.
type Injector struct {
	detectors []Detector
	templates map[string]string
	counters  map[string]int
}

// NewInjector creates an Injector with the given detectors and their templates.
func NewInjector(detectors []Detector, templates map[string]string) *Injector {
	return &Injector{
		detectors: detectors,
		templates: templates,
		counters:  make(map[string]int),
	}
}

// Inject evaluates all detectors and returns reminder messages for those that fire.
// Respects the MaxFires cap for each detector.
func (inj *Injector) Inject(ctx CheckContext) []message.Message {
	var reminders []message.Message

	for _, det := range inj.detectors {
		name := det.Name()

		if inj.counters[name] >= det.MaxFires() {
			continue
		}

		if !det.Check(ctx) {
			continue
		}

		inj.counters[name]++

		tmpl, ok := inj.templates[name]
		if !ok {
			continue
		}

		reminders = append(reminders, message.Message{
			Role:    message.RoleUser,
			Content: tmpl,
		})
	}

	return reminders
}

// Reset clears all fire counters.
func (inj *Injector) Reset() {
	clear(inj.counters)
}
