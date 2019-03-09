package evaluator

import (
	"fmt"
	"reflect"

	"github.com/lyraproj/pcore/pcore"
	"github.com/lyraproj/pcore/px"
)

func ExampleNewError_reflectTo() {
	type TestStruct struct {
		Message   string
		Kind      string
		IssueCode string `puppet:"name => issue_code"`
	}

	c := pcore.RootContext()
	ts := &TestStruct{}

	ev := px.NewError(c, `the message`, `THE_KIND`, `THE_CODE`, nil, nil)
	c.Reflector().ReflectTo(ev, reflect.ValueOf(ts).Elem())
	fmt.Printf("message: %s, kind %s, issueCode %s\n", ts.Message, ts.Kind, ts.IssueCode)
	// Output: message: the message, kind THE_KIND, issueCode THE_CODE
}

func ExampleErrorMetaType() {
	type TestStruct struct {
		Message   string
		Kind      string
		IssueCode string `puppet:"name => issue_code"`
	}

	c := pcore.RootContext()
	ts := &TestStruct{`the message`, `THE_KIND`, `THE_CODE`}
	et, _ := px.Load(c, px.NewTypedName(px.NsType, `Error`))
	ev := et.(px.ObjectType).FromReflectedValue(c, reflect.ValueOf(ts).Elem())
	fmt.Println(ErrorMetaType.IsInstance(ev, nil))
	// Output: true
}
