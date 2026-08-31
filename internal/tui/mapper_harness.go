package tui

import (
	"github.com/dmytrogajewski/spin/internal/events"
	"github.com/dmytrogajewski/spin/internal/ui/blocks"
)

func (m *Mapper) handleHarnessEvent(event events.Event) error {
	switch event.Type {
	case events.EventSubagentSpawn, events.EventSubagentComplete:
		return m.handleSubagentEvent(event)
	case events.EventBackgroundTaskStarted, events.EventBackgroundTaskStopped:
		return m.handleTaskEvent(event)
	case events.EventCompactionTriggered:
		return m.handleCompactEvent(event)
	case events.EventHookVeto:
		return m.handleHookVeto(event)
	default:
		return nil
	}
}

func (m *Mapper) handleSubagentEvent(event events.Event) error {
	block := blocks.NewBlock(blocks.BlockTypeSubagent)
	block.ID = generateBlockID()

	if data, ok := event.SubagentSpawnData(); ok {
		block.Title = data.AgentType
		block.Body = data.Query
	}

	if data, ok := event.SubagentCompleteData(); ok {
		block.Title = data.AgentType
		block.Body = data.Summary
	}

	return m.ui.AppendBlock(block)
}

func (m *Mapper) handleTaskEvent(event events.Event) error {
	data, ok := event.TaskStateData()
	if !ok {
		return nil
	}

	block := blocks.NewBlock(blocks.BlockTypeTask)
	block.ID = data.TaskID

	if block.ID == "" {
		block.ID = generateBlockID()
	}

	block.Title = data.TaskID
	block.Body = data.State

	return m.ui.AppendBlock(block)
}

func (m *Mapper) handleCompactEvent(event events.Event) error {
	block := blocks.NewBlock(blocks.BlockTypeCompact)
	block.ID = generateBlockID()

	if data, ok := event.CompactionTriggeredData(); ok {
		block.Title = data.Stage
	}

	return m.ui.AppendBlock(block)
}

func (m *Mapper) handleHookVeto(event events.Event) error {
	data, ok := event.HookVetoData()
	if !ok {
		return nil
	}

	block := blocks.NewBlock(blocks.BlockTypeHook)
	block.ID = generateBlockID()
	block.Title = data.Event
	block.Body = data.Reason
	block.Severity = blocks.SeverityError

	return m.ui.AppendBlock(block)
}
