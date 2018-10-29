package profiler

import (
	"fmt"

	"github.com/codeuniversity/ppp-mhist"

	"github.com/robertkrimen/otto"
)

//Profile is user-scripted aggregation of data
type Profile struct {
	definition ProfileDefinition
	store      map[string]interface{}
	display    map[string]interface{}
}

//NewProfile with the provided definition, including a script
func NewProfile(definition ProfileDefinition) *Profile {
	return &Profile{
		definition: definition,
		store:      make(map[string]interface{}),
		display:    make(map[string]interface{}),
	}
}

//ProfileDisplayValue is filled by javascript
type ProfileDisplayValue struct {
	ID   string                 `json:"id"`
	Data map[string]interface{} `json:"data"`
}

//Value is the current display state of the profile. Completely generated in javascript
func (p *Profile) Value() ProfileDisplayValue {
	return ProfileDisplayValue{
		ID:   p.definition.ID,
		Data: p.display,
	}
}

//Eval message with script
func (p *Profile) Eval(message *mhist.Message) {
	vm := p.getJavascriptVMWithPresets(message)
	_, err := vm.Run(p.definition.EvalScript)
	if err != nil {
		fmt.Println(err)
	}
}

func (p *Profile) getJavascriptVMWithPresets(message *mhist.Message) *otto.Otto {
	vm := otto.New()
	vm.Set("set", p.putValueInStore)
	vm.Set("get", p.getValueFromStore)
	vm.Set("display", p.putValueInDisplay)

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

func (p *Profile) putValueInDisplay(call otto.FunctionCall) otto.Value {
	key := call.Argument(0).String()
	value, _ := call.Argument(1).Export()

	if key != "" && value != nil {
		p.display[key] = value
	}

	return otto.Value{}
}

func (p *Profile) putValueInStore(call otto.FunctionCall) otto.Value {
	key := call.Argument(0).String()
	value, _ := call.Argument(1).Export()

	if key != "" && value != nil {
		p.store[key] = value
	}

	return otto.Value{}
}

func (p *Profile) getValueFromStore(call otto.FunctionCall) otto.Value {
	key := call.Argument(0).String()
	defaultValue := call.Argument(1)

	if key == "" {
		return otto.Value{}
	}

	storedValue := p.store[key]
	if storedValue == nil && defaultValue.IsDefined() {
		return defaultValue
	}

	value, err := otto.ToValue(storedValue)
	if err != nil {
		return otto.Value{}
	}

	return value
}
