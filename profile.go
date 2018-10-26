package profiler

import (
	"fmt"

	"github.com/codeuniversity/ppp-mhist"

	"github.com/robertkrimen/otto"
)

//Profile is a scripted aggregation of data
type Profile struct {
	filter mhist.FilterDefinition
	script string
	store  map[string]interface{}
}

//NewProfile with the provided aggregation script
func NewProfile(script string, filterDefinition mhist.FilterDefinition) *Profile {
	return &Profile{
		script: script,
		store:  make(map[string]interface{}),
		filter: filterDefinition,
	}
}

//Value is the current state of the profile. Completely generated in javascript
func (p *Profile) Value() map[string]interface{} {
	return p.store
}

//Eval message with script
func (p *Profile) Eval(message *mhist.Message) {
	if !p.filter.IsInNames(message.Name) {
		return
	}

	vm := p.getJavascriptVMWithPresets(message)
	_, err := vm.Run(p.script)
	if err != nil {
		fmt.Println(err)
	}
}

func (p *Profile) getJavascriptVMWithPresets(message *mhist.Message) *otto.Otto {
	vm := otto.New()
	vm.Set("set", p.putValueInStore)
	vm.Set("get", p.getValueFromStore)

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
