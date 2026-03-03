package opencode

import (
	"encoding/json"
	"os"
)

func envLookup() []string { return os.Environ() }

// jsonUnmarshal is a package-level alias so helpers can call it without importing encoding/json.
var jsonUnmarshal = json.Unmarshal
