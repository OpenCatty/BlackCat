package agent

type NextStep int

const (
	RunAgain NextStep = iota
	FinalOutput
	Interrupted
	Handoff
	Error
)
