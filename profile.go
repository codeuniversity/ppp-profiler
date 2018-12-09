package profiler

import (
	"github.com/codeuniversity/ppp-mhist"

	"github.com/robertkrimen/otto"
)

//Profile is user-scripted aggregation of data
type Profile struct {
	Definition ProfileDefinition      `json:"definition"`
	Store      map[string]interface{} `json:"store"`
	Display    map[string]interface{} `json:"display"`
}

//NewProfile with the provided definition, including a script
func NewProfile(definition ProfileDefinition) *Profile {
	return &Profile{
		Definition: definition,
		Store:      make(map[string]interface{}),
		Display:    make(map[string]interface{}),
	}
}

//ProfileData ...
type ProfileData struct {
	Definition ProfileDefinition      `json:"definition"`
	Display    map[string]interface{} `json:"display"`
}

//ProfileDisplayValue is filled by javascript
type ProfileDisplayValue struct {
	ID   string      `json:"id"`
	Data ProfileData `json:"data"`
}

//Value is the current display state of the profile. Completely generated in javascript
func (p *Profile) Value() ProfileDisplayValue {
	return ProfileDisplayValue{
		ID: p.Definition.ID,
		Data: ProfileData{
			Definition: p.Definition,
			Display:    p.Display,
		},
	}
}

//Eval message with script
func (p *Profile) Eval(message *mhist.Message) {
	if !p.Definition.Filter.IsInNames(message.Name) {
		return
	}
	p.Display = map[string]interface{}{}

	vm := p.getJavascriptVMWithPresets(message)
	_, err := vm.Run(p.Definition.EvalScript)
	if err != nil {
		p.Display["error"] = err.Error()
	}
}

func (p *Profile) getJavascriptVMWithPresets(message *mhist.Message) *otto.Otto {
	vm := otto.New()
	vm.Set("set", p.putValueInStore)
	vm.Set("get", p.getValueFromStore)
	vm.Set("title", p.displaySetterForKey("title"))
	vm.Set("description", p.displaySetterForKey("description"))
	vm.Set("action", p.displaySetterForKey("action"))

	messageObject, _ := vm.Object("abc = {}")
	if message.Name != "" {
		messageObject.Set("name", message.Name)
	}
	if message.Timestamp != 0 {
		messageObject.Set("timestamp", message.Timestamp)
	}
	if message.Value != nil {
		messageObject.Set("value", message.Value)
	}
	vm.Set("message", messageObject)

	return vm
}

func (p *Profile) displaySetterForKey(displayKey string) func(call otto.FunctionCall) otto.Value {
	return func(call otto.FunctionCall) otto.Value {
		value, _ := call.Argument(0).Export()

		if value != nil {
			p.Display[displayKey] = value
		}

		return otto.Value{}
	}
}

func (p *Profile) putValueInStore(call otto.FunctionCall) otto.Value {
	key := call.Argument(0).String()
	value, _ := call.Argument(1).Export()

	if key != "" && value != nil {
		p.Store[key] = value
	}

	return otto.Value{}
}

func (p *Profile) getValueFromStore(call otto.FunctionCall) otto.Value {
	key := call.Argument(0).String()
	defaultValue := call.Argument(1)

	if key == "" {
		return otto.Value{}
	}

	storedValue := p.Store[key]
	if storedValue == nil && defaultValue.IsDefined() {
		return defaultValue
	}

	value, err := otto.ToValue(storedValue)
	if err != nil {
		return otto.Value{}
	}

	return value
}
