package hooks

// HookEvent identifies lifecycle points where hooks can run.
type HookEvent string

const (
	PreChat        HookEvent = "PreChat"
	PostChat       HookEvent = "PostChat"
	PreToolExec    HookEvent = "PreToolExec"
	PostToolExec   HookEvent = "PostToolExec"
	PreFileRead    HookEvent = "PreFileRead"
	PostFileRead   HookEvent = "PostFileRead"
	PreFileWrite   HookEvent = "PreFileWrite"
	PostFileWrite  HookEvent = "PostFileWrite"
	OnSessionStart HookEvent = "OnSessionStart"
	OnSessionEnd   HookEvent = "OnSessionEnd"

	// Message lifecycle hooks
	MessageReceived HookEvent = "MessageReceived"
	MessageSending  HookEvent = "MessageSending"
	MessageSent     HookEvent = "MessageSent"
)
