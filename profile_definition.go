package profiler

import (
	"github.com/codeuniversity/ppp-mhist"
)

//ProfileDefinition is the information that is sent to the server to define a profile
type ProfileDefinition struct {
	ID         string                 `json:"id"`
	EvalScript string                 `json:"eval_script"`
	IsLocal    bool                   `json:"is_local"`
	Filter     mhist.FilterDefinition `json:"filter"`
	LibraryID  string                 `json:"library_id"`
}

//ProfileDefinitionUpdate is ProfileDefinition with pointers which can be nil for the update endpoint
type ProfileDefinitionUpdate struct {
	EvalScript *string                 `json:"eval_script,omitempty"`
	IsLocal    *bool                   `json:"is_local,omitempty"`
	Filter     *mhist.FilterDefinition `json:"filter,omitempty"`
	LibraryID  *string                 `json:"library_id,omitempty"`
}
