package status

import "github.com/dmytrogajewski/spin/pkg/ui/spinner"

type (
	// SpinnerStyle is an alias for [spinner.Style].
	SpinnerStyle = spinner.Style
	// Spinner is an alias for [spinner.Spinner].
	Spinner = spinner.Spinner
	// ActivitySpinner is an alias for [spinner.ActivitySpinner].
	ActivitySpinner = spinner.ActivitySpinner
)

// Style constant aliases.
const (
	SpinnerDots    = spinner.StyleDots
	SpinnerBraille = spinner.StyleBraille
	SpinnerCircle  = spinner.StyleCircle
	SpinnerPulse   = spinner.SpinnerPulse
)

// Function aliases.
var (
	NewSpinner         = spinner.NewSpinner
	NewActivitySpinner = spinner.NewActivitySpinner
)
