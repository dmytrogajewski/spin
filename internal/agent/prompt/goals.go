package prompt

// Phase represents a phase in the software development lifecycle.
type Phase struct {
	Name  string
	Goals []string
}

// DeveloperGoals returns the 30 developer goals organized by SDLC phase.
func DeveloperGoals() []Phase {
	return []Phase{
		{
			Name: "Information Gathering",
			Goals: []string{
				"Ensure documentation is up to date",
				"Understand the context to complete a work item",
				"Explore technical solutions (e.g., bugs, design)",
				"Find information (e.g., documentation, codelabs, API examples)",
				"Find an expert",
			},
		},
		{
			Name: "Plan and Track Work, and Manage Approvals",
			Goals: []string{
				"Know what to work on next",
				"Coordinate work with peers",
				"Ensure my launch complies with legal, privacy, and security requirements",
				"Have my cross-functional team aligned on launch readiness",
				"Get my design approved",
				"Design and document a considered plan",
			},
		},
		{
			Name: "Develop, Test and Commit Code",
			Goals: []string{
				"Write high quality code",
				"Ensure the code contributed by others (e.g., teammates, AI) is high quality",
				"Understand the behavior of existing code",
				"Create or maintain holistic test coverage",
				"Investigate unexpected behavior locally",
				"Integrate new tools/technology into existing services and systems",
			},
		},
		{
			Name: "Experiment, Release and Rollout",
			Goals: []string{
				"Safely roll out changes to production (e.g., features, models, new releases)",
				"Run an experiment",
				"Analyze experiment results",
			},
		},
		{
			Name: "Monitoring, Reliability, and Configuring Infrastructure",
			Goals: []string{
				"Ensure my product stays within SLO commitments",
				"Investigate issues in production (e.g., crashes, unexpected behavior, outages)",
				"Improve system performance",
				"Manage compute resources",
				"Ensure my builds stay green (e.g., build gardening, rotations)",
				"Improve reliability and avoid production problems",
			},
		},
		{
			Name: "Data Management",
			Goals: []string{
				"Ensure data I'm responsible for is fresh, reliable, and of high quality",
				"Develop and manage data processing pipelines",
				"Ensure data I'm responsible for is secure and complies with regulations",
				"Analyze, visualize, and understand data to generate insights",
			},
		},
	}
}

// TotalGoalCount is the total number of developer goals across all phases.
const TotalGoalCount = 30
