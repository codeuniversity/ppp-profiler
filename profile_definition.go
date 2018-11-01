package profiler

//ProfileDefinition is the information that is sent to the server to define a profile
type ProfileDefinition struct {
	ID         int    `json:"id"`
	EvalScript string `json:"eval_script"`
}
