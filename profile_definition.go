package profiler

//ProfileDefinition is the information that is sent to the server to define a profile
type ProfileDefinition struct {
	ID         string `json:"id"`
	EvalScript string `json:"eval_script"`
	IsLocal    bool   `json:"is_local"`
}
