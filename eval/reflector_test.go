package eval_test

import (
	"fmt"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
	"github.com/lyraproj/semver/semver"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"time"

	// Initialize pcore
	_ "github.com/lyraproj/puppet-evaluator/pcore"
)

func ExampleReflector_reflectArray() {
	c := eval.Puppet.RootContext()

	av := eval.Wrap(nil, []interface{}{`hello`, 3}).(*types.ArrayValue)
	ar := c.Reflector().Reflect(av)
	fmt.Printf("%s %v\n", ar.Type(), ar)

	av = eval.Wrap(nil, []interface{}{`hello`, `world`}).(*types.ArrayValue)
	ar = c.Reflector().Reflect(av)
	fmt.Printf("%s %v\n", ar.Type(), ar)
	// Output:
	// []interface {} [hello 3]
	// []string [hello world]
}

func ExampleReflector_reflectToArray() {
	type TestStruct struct {
		Strings    []string
		Interfaces []interface{}
		Values     []eval.Value
	}

	c := eval.Puppet.RootContext()
	ts := &TestStruct{}
	rv := reflect.ValueOf(ts).Elem()

	av := eval.Wrap(nil, []string{`hello`, `world`})

	rf := c.Reflector()
	rf.ReflectTo(av, rv.Field(0))
	rf.ReflectTo(av, rv.Field(1))
	rf.ReflectTo(av, rv.Field(2))
	fmt.Println(ts)

	rf.ReflectTo(eval.UNDEF, rv.Field(0))
	rf.ReflectTo(eval.UNDEF, rv.Field(1))
	rf.ReflectTo(eval.UNDEF, rv.Field(2))
	fmt.Println(ts)
	// Output:
	// &{[hello world] [hello world] [hello world]}
	// &{[] [] []}
}

func ExampleReflector_reflectToHash() {
	var strings map[string]string
	var interfaces map[string]interface{}
	var values map[string]eval.Value

	c := eval.Puppet.RootContext()
	hv := eval.Wrap(nil, map[string]string{`x`: `hello`, `y`: `world`})

	rf := c.Reflector()
	rf.ReflectTo(hv, reflect.ValueOf(&strings).Elem())
	rf.ReflectTo(hv, reflect.ValueOf(&interfaces).Elem())
	rf.ReflectTo(hv, reflect.ValueOf(&values).Elem())
	fmt.Printf("%s %s\n", strings[`x`], strings[`y`])
	fmt.Printf("%s %s\n", interfaces[`x`], interfaces[`y`])
	fmt.Printf("%s %s\n", values[`x`], values[`y`])

	rf.ReflectTo(eval.UNDEF, reflect.ValueOf(&strings).Elem())
	rf.ReflectTo(eval.UNDEF, reflect.ValueOf(&interfaces).Elem())
	rf.ReflectTo(eval.UNDEF, reflect.ValueOf(&values).Elem())
	fmt.Println(strings)
	fmt.Println(interfaces)
	fmt.Println(values)
	// Output:
	// hello world
	// hello world
	// hello world
	// map[]
	// map[]
	// map[]
}

func ExampleReflector_reflectToBytes() {
	var buf []byte

	c := eval.Puppet.RootContext()

	rf := c.Reflector()
	bv := eval.Wrap(nil, []byte{1, 2, 3})
	rf.ReflectTo(bv, reflect.ValueOf(&buf).Elem())
	fmt.Println(buf)

	rf.ReflectTo(eval.UNDEF, reflect.ValueOf(&buf).Elem())
	fmt.Println(buf)
	// Output:
	// [1 2 3]
	// []
}

func ExampleReflector_reflectToFloat() {
	type TestStruct struct {
		SmallFloat float32
		BigFloat   float64
		APValue    eval.Value
		IValue     interface{}
	}

	c := eval.Puppet.RootContext()
	rf := c.Reflector()
	ts := &TestStruct{}
	rv := reflect.ValueOf(ts).Elem()
	n := rv.NumField()
	for i := 0; i < n; i++ {
		fv := eval.Wrap(nil, float64(10+i+1)/10.0)
		rf.ReflectTo(fv, rv.Field(i))
	}
	fmt.Println(ts)
	// Output: &{1.1 1.2 1.3 1.4}
}

func ExampleReflector_reflectToInt() {
	type TestStruct struct {
		AnInt    int
		AnInt8   int8
		AnInt16  int16
		AnInt32  int32
		AnInt64  int64
		AnUInt   uint
		AnUInt8  uint8
		AnUInt16 uint16
		AnUInt32 uint32
		AnUInt64 uint64
		APValue  eval.Value
		IValue   interface{}
	}

	c := eval.Puppet.RootContext()
	rf := c.Reflector()
	ts := &TestStruct{}
	rv := reflect.ValueOf(ts).Elem()
	n := rv.NumField()
	for i := 0; i < n; i++ {
		rf.ReflectTo(eval.Wrap(nil, 10+i), rv.Field(i))
	}
	fmt.Println(ts)
	// Output: &{10 11 12 13 14 15 16 17 18 19 20 21}
}

func ExampleReflector_reflectToRegexp() {
	var expr regexp.Regexp

	rx := eval.Wrap(nil, regexp.MustCompile("[a-z]*"))
	eval.Puppet.RootContext().Reflector().ReflectTo(rx, reflect.ValueOf(&expr).Elem())

	fmt.Println(expr.String())
	// Output: [a-z]*
}

func ExampleReflector_reflectToTimespan() {
	var span time.Duration

	rx := eval.Wrap(nil, time.Duration(1234))
	eval.Puppet.RootContext().Reflector().ReflectTo(rx, reflect.ValueOf(&span).Elem())

	fmt.Println(span)
	// Output: 1.234µs
}

func ExampleReflector_reflectToTimestamp() {
	var tx time.Time

	tm, _ := time.Parse(time.RFC3339, "2018-05-11T06:31:22+01:00")
	tv := eval.Wrap(nil, tm)
	eval.Puppet.RootContext().Reflector().ReflectTo(tv, reflect.ValueOf(&tx).Elem())

	fmt.Println(tx.Format(time.RFC3339))
	// Output: 2018-05-11T06:31:22+01:00
}

func ExampleReflector_reflectToVersion() {
	var version semver.Version

	ver, _ := semver.ParseVersion(`1.2.3`)
	vv := eval.Wrap(nil, ver)
	eval.Puppet.RootContext().Reflector().ReflectTo(vv, reflect.ValueOf(&version).Elem())

	fmt.Println(version)
	// Output: 1.2.3
}

func ExampleReflector_ReflectToPuppetObject() {
	type TestStruct struct {
		Message   string
		Kind      string
		IssueCode string
	}

	c := eval.Puppet.RootContext()
	ts := &TestStruct{}

	ev := eval.NewError(c, `the message`, `THE_KIND`, `THE_CODE`, nil, nil)
	c.Reflector().ReflectTo(ev, reflect.ValueOf(ts).Elem())
	fmt.Printf("message: %s, kind %s, issueCode %s\n", ts.Message, ts.Kind, ts.IssueCode)
	// Output: message: the message, kind THE_KIND, issueCode THE_CODE
}

func ExampleReflector_typeFromReflect() {
	type TestAddress struct {
		Street string
		Zip    string
	}
	type TestPerson struct {
		Name    string
		Address *TestAddress
	}
	type TestExtendedPerson struct {
		TestPerson
		Age    *int
		Active bool `puppet:"name=>enabled"`
	}

	c := eval.Puppet.RootContext()
	rtAddress := reflect.TypeOf(&TestAddress{})
	rtPerson := reflect.TypeOf(&TestPerson{})
	rtExtPerson := reflect.TypeOf(&TestExtendedPerson{})

	rf := c.Reflector()
	tAddress := rf.TypeFromTagged(`My::Address`, nil, eval.NewTaggedType(rtAddress, map[string]string{`Zip`: `name=>zip_code`}), nil)
	tPerson := rf.TypeFromReflect(`My::Person`, nil, rtPerson)
	tExtPerson := rf.TypeFromReflect(`My::ExtendedPerson`, tPerson, rtExtPerson)
	c.AddTypes(tAddress, tPerson, tExtPerson)

	tAddress.ToString(os.Stdout, types.EXPANDED, nil)
	fmt.Println()
	tPerson.ToString(os.Stdout, types.EXPANDED, nil)
	fmt.Println()
	tExtPerson.ToString(os.Stdout, types.EXPANDED, nil)
	fmt.Println()

	age := 34
	ts := &TestExtendedPerson{TestPerson{`Bob Tester`, &TestAddress{`Example Road 23`, `12345`}}, &age, true}
	ev := eval.Wrap(c, ts)
	fmt.Println(ev)

	// Output:
	// Object[{name => 'My::Address', attributes => {'street' => String, 'zip_code' => String}}]
	// Object[{name => 'My::Person', attributes => {'name' => String, 'address' => {'type' => Optional[My::Address], 'value' => undef}}}]
	// Object[{name => 'My::ExtendedPerson', parent => My::Person, attributes => {'age' => {'type' => Optional[Integer], 'value' => undef}, 'enabled' => Boolean}}]
	// My::ExtendedPerson('name' => 'Bob Tester', 'enabled' => true, 'address' => My::Address('street' => 'Example Road 23', 'zip_code' => '12345'), 'age' => 34)
}

type A interface {
	Int() int
	X() string
	Y(A) A
}

type B int

func NewA(x int) A {
	return B(x)
}

func (b B) X() string {
	return strconv.Itoa(int(b) + 5)
}

func (b B) Int() int {
	return int(b)
}

func (b B) Y(a A) A {
	return NewA(int(b) + a.Int())
}

func ExampleReflector_typeFromReflectInterface() {
	c := eval.Puppet.RootContext()

	// Create ObjectType from reflected type
	xa := c.Reflector().TypeFromReflect(`X::A`, nil, reflect.TypeOf((*A)(nil)).Elem())
	xb := c.Reflector().TypeFromReflect(`X::B`, nil, reflect.TypeOf(B(0)))

	// Ensure that the type is resolved
	c.AddTypes(xa, xb)

	// Print the created Interface Type in human readable form
	xa.ToString(os.Stdout, eval.PRETTY_EXPANDED, nil)
	fmt.Println()

	// Print the created Implementation Type in human readable form
	xb.ToString(os.Stdout, eval.PRETTY_EXPANDED, nil)
	fmt.Println()

	// Invoke method 'x' on the interface on a receiver
	m, _ := xb.Member(`x`)
	gm := m.(eval.CallableGoMember)

	fmt.Println(gm.CallGo(c, NewA(32))[0])
	fmt.Println(gm.CallGo(c, B(25))[0])

	// Invoke method 'x' on the interface on a receiver
	m, _ = xb.Member(`y`)

	// Call Go function using CallableGoMember
	gv := m.(eval.CallableGoMember).CallGo(c, B(25), NewA(15))[0]
	fmt.Printf("%T %v\n", gv, gv)

	// Call Go function using CallableMember and eval.Value
	pv := m.Call(c, eval.New(c, xb, eval.Wrap(c, 25)), nil, []eval.Value{eval.New(c, xb, eval.Wrap(c, 15))})
	fmt.Println(pv)

	// Output:
	// Object[{
	//   name => 'X::A',
	//   functions => {
	//     'int' => Callable[
	//       [0, 0],
	//       Integer],
	//     'x' => Callable[
	//       [0, 0],
	//       String],
	//     'y' => Callable[
	//       [X::A],
	//       X::A]
	//   }
	// }]
	// Object[{
	//   name => 'X::B',
	//   attributes => {
	//     'value' => Integer
	//   },
	//   functions => {
	//     'int' => Callable[
	//       [0, 0],
	//       Integer],
	//     'x' => Callable[
	//       [0, 0],
	//       String],
	//     'y' => Callable[
	//       [X::A],
	//       X::A]
	//   }
	// }]
	// 37
	// 30
	// eval_test.B 40
	// X::B('value' => 40)
}

type Address struct {
	Street string
	Zip    string `puppet:"name=>zip_code"`
}
type Person struct {
	Name    string
	Address *Address
}
type ExtendedPerson struct {
	Person
	Birth          *time.Time
	TimeSinceVisit *time.Duration
	Active         bool `puppet:"name=>enabled"`
}

func (p *Person) Visit(v *Address) string {
	return "visited " + v.Street
}

func ExampleReflector_typeSetFromReflect() {
	eval.Puppet.Do(func(c eval.Context) {
		// Create a TypeSet from a list of Go structs
		typeSet := c.Reflector().TypeSetFromReflect(`My::Own`, semver.MustParseVersion(`1.0.0`), nil,
			reflect.TypeOf(&Address{}), reflect.TypeOf(&Person{}), reflect.TypeOf(&ExtendedPerson{}))

		// Make the types known to the current loader
		c.AddTypes(typeSet)

		// Print the TypeSet in human readable form
		typeSet.ToString(os.Stdout, eval.PRETTY_EXPANDED, nil)
		fmt.Println()

		// Create an instance of something included in the TypeSet
		ad := &Address{`Example Road 23`, `12345`}
		birth, _ := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
		tsv, _ := time.ParseDuration("1h20m")
		ep := &ExtendedPerson{Person{`Bob Tester`, ad}, &birth, &tsv, true}

		// Wrap the instance as a Value and print it
		v := eval.Wrap(c, ep)
		fmt.Println(v)

		m, _ := v.PType().(eval.TypeWithCallableMembers).Member(`visit`)
		fmt.Println(m.(eval.CallableGoMember).CallGo(c, ep, ad)[0])
		fmt.Println(m.Call(c, v, nil, []eval.Value{eval.Wrap(c, ad)}))
	})

	// Output:
	// TypeSet[{
	//   pcore_uri => 'http://puppet.com/2016.1/pcore',
	//   pcore_version => '1.0.0',
	//   name_authority => 'http://puppet.com/2016.1/runtime',
	//   name => 'My::Own',
	//   version => '1.0.0',
	//   types => {
	//     Address => {
	//       attributes => {
	//         'street' => String,
	//         'zip_code' => String
	//       }
	//     },
	//     Person => {
	//       attributes => {
	//         'name' => String,
	//         'address' => {
	//           'type' => Optional[Address],
	//           'value' => undef
	//         }
	//       },
	//       functions => {
	//         'visit' => Callable[
	//           [Optional[Address]],
	//           String]
	//       }
	//     },
	//     ExtendedPerson => Person{
	//       attributes => {
	//         'birth' => {
	//           'type' => Optional[Timestamp],
	//           'value' => undef
	//         },
	//         'time_since_visit' => {
	//           'type' => Optional[Timespan],
	//           'value' => undef
	//         },
	//         'enabled' => Boolean
	//       }
	//     }
	//   }
	// }]
	// My::Own::ExtendedPerson('name' => 'Bob Tester', 'enabled' => true, 'address' => My::Own::Address('street' => 'Example Road 23', 'zip_code' => '12345'), 'birth' => 2006-01-02T15:04:05.000000000 UTC, 'time_since_visit' => 0-01:20:00.0)
	// visited Example Road 23
	// visited Example Road 23
}

type twoValueReturn struct {
}

func (t *twoValueReturn) ReturnTwo() (string, int) {
	return "number", 42
}

func (t *twoValueReturn) ReturnTwoAndErrorOk() (string, int, error) {
	return "number", 42, nil
}

func (t *twoValueReturn) ReturnTwoAndErrorFail() (string, int, error) {
	return ``, 0, fmt.Errorf(`bad things happened`)
}

func ExampleReflector_twoValueReturn() {
	eval.Puppet.Do(func(c eval.Context) {
		gv := &twoValueReturn{}
		api := c.Reflector().TypeFromReflect(`A::B`, nil, reflect.TypeOf(gv))
		c.AddTypes(api)
		v := eval.Wrap(c, gv)
		r, _ := api.Member(`return_two`)
		fmt.Println(eval.ToPrettyString(r.Call(c, v, nil, eval.EMPTY_VALUES)))
	})
	// Output:
	// ['number', 42]
}

func ExampleReflectortwoValueReturnErrorOk() {
	eval.Puppet.Do(func(c eval.Context) {
		gv := &twoValueReturn{}
		api := c.Reflector().TypeFromReflect(`A::B`, nil, reflect.TypeOf(gv))
		c.AddTypes(api)
		v := eval.Wrap(c, gv)
		r, _ := api.Member(`return_two_and_error_ok`)
		fmt.Println(eval.ToPrettyString(r.Call(c, v, nil, eval.EMPTY_VALUES)))
	})
	// Output:
	// ['number', 42]
}

func ExampleReflectortwoValueReturnErrorFail() {
	err := eval.Puppet.Try(func(c eval.Context) error {
		gv := &twoValueReturn{}
		api := c.Reflector().TypeFromReflect(`A::B`, nil, reflect.TypeOf(gv))
		c.AddTypes(api)
		v := eval.Wrap(c, gv)
		r, _ := api.Member(`return_two_and_error_fail`)
		r.Call(c, v, nil, eval.EMPTY_VALUES)
		return nil
	})
	if err != nil {
		fmt.Println(err.Error())
	}
	// Output:
	// Go function ReturnTwoAndErrorFail returned error 'bad things happened'
}

type valueStruct struct {
	X eval.OrderedMap
	Y *types.ArrayValue
	P eval.PuppetObject
	O eval.Object
}

func (v *valueStruct) Get(key eval.IntegerValue, dflt eval.Value) eval.StringValue {
	return v.X.Get2(key, eval.UNDEF).(eval.StringValue)
}

func ExampleReflector_reflectPType() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X::M`, nil, reflect.TypeOf(&valueStruct{}))
		c.AddTypes(xm)
		xm.ToString(os.Stdout, eval.PRETTY_EXPANDED, nil)
		fmt.Println()
	})
	// Output:
	// Object[{
	//   name => 'X::M',
	//   attributes => {
	//     'x' => Hash,
	//     'y' => Array,
	//     'p' => Object,
	//     'o' => Object
	//   },
	//   functions => {
	//     'get' => Callable[
	//       [Integer, Any],
	//       String]
	//   }
	// }]
}

type optionalString struct {
	A string
	B *string
}

func ExampleReflect_TypeFromReflect_optionalString() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalString{}))
		c.AddTypes(xm)
		xm.ToString(os.Stdout, eval.PRETTY_EXPANDED, nil)
		fmt.Println()
	})

	// Output:
	// Object[{
	//   name => 'X',
	//   attributes => {
	//     'a' => String,
	//     'b' => {
	//       'type' => Optional[String],
	//       'value' => undef
	//     }
	//   }
	// }]
}

func ExampleReflect_TypeFromReflect_optionalStringReflect() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalString{}))
		c.AddTypes(xm)
		w := `World`
		fmt.Println(eval.ToPrettyString(eval.Wrap(c, &optionalString{`Hello`, &w})))
	})

	// Output:
	// X(
	//   'a' => 'Hello',
	//   'b' => 'World'
	// )
}

func ExampleReflect_TypeFromReflect_optionalStringReflectZero() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalString{}))
		c.AddTypes(xm)
		var w string
		fmt.Println(eval.ToPrettyString(eval.Wrap(c, &optionalString{`Hello`, &w})))
	})

	// Output:
	// X(
	//   'a' => 'Hello',
	//   'b' => ''
	// )
}

func ExampleReflect_TypeFromReflect_optionalStringCreate() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalString{}))
		c.AddTypes(xm)
		fmt.Println(eval.ToPrettyString(eval.New(c, xm, types.WrapString(`Hello`), types.WrapString(`World`))))
	})

	// Output:
	// X(
	//   'a' => 'Hello',
	//   'b' => 'World'
	// )
}

func ExampleReflect_TypeFromReflect_optionalStringDefault() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalString{}))
		c.AddTypes(xm)
		fmt.Println(eval.ToPrettyString(eval.New(c, xm, types.WrapString(`Hello`))))
	})

	// Output:
	// X(
	//   'a' => 'Hello'
	// )
}

func ExampleReflect_TypeFromReflect_optionalStringDefaultZero() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalString{}))
		c.AddTypes(xm)
		var w string
		fmt.Println(eval.ToPrettyString(eval.New(c, xm, types.WrapString(`Hello`), types.WrapString(w))))
	})

	// Output:
	// X(
	//   'a' => 'Hello',
	//   'b' => ''
	// )
}

type optionalBoolean struct {
	A bool
	B *bool
}

func ExampleReflect_TypeFromReflect_optionalBoolean() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalBoolean{}))
		c.AddTypes(xm)
		xm.ToString(os.Stdout, eval.PRETTY_EXPANDED, nil)
		fmt.Println()
	})

	// Output:
	// Object[{
	//   name => 'X',
	//   attributes => {
	//     'a' => Boolean,
	//     'b' => {
	//       'type' => Optional[Boolean],
	//       'value' => undef
	//     }
	//   }
	// }]
}

func ExampleReflect_TypeFromReflect_optionalBooleanReflect() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalBoolean{}))
		c.AddTypes(xm)
		w := true
		fmt.Println(eval.ToPrettyString(eval.Wrap(c, &optionalBoolean{true, &w})))
	})

	// Output:
	// X(
	//   'a' => true,
	//   'b' => true
	// )
}

func ExampleReflect_TypeFromReflect_optionalBooleanReflectZero() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalBoolean{}))
		c.AddTypes(xm)
		var w bool
		fmt.Println(eval.ToPrettyString(eval.Wrap(c, &optionalBoolean{true, &w})))
	})

	// Output:
	// X(
	//   'a' => true,
	//   'b' => false
	// )
}

func ExampleReflect_TypeFromReflect_optionalBooleanCreate() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalBoolean{}))
		c.AddTypes(xm)
		fmt.Println(eval.ToPrettyString(eval.New(c, xm, types.WrapBoolean(true), types.WrapBoolean(true))))
	})

	// Output:
	// X(
	//   'a' => true,
	//   'b' => true
	// )
}

func ExampleReflect_TypeFromReflect_optionalBooleanDefault() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalBoolean{}))
		c.AddTypes(xm)
		fmt.Println(eval.ToPrettyString(eval.New(c, xm, types.WrapBoolean(true))))
	})

	// Output:
	// X(
	//   'a' => true
	// )
}

func ExampleReflect_TypeFromReflect_optionalBooleanZero() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalBoolean{}))
		c.AddTypes(xm)
		var w bool
		fmt.Println(eval.ToPrettyString(eval.New(c, xm, types.WrapBoolean(true), types.WrapBoolean(w))))
	})

	// Output:
	// X(
	//   'a' => true,
	//   'b' => false
	// )
}

type optionalInt struct {
	A int64
	B *int64
	C *byte
	D *int16
	E *uint32
}

func ExampleReflect_TypeFromReflect_optionalInt() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalInt{}))
		c.AddTypes(xm)
		xm.ToString(os.Stdout, eval.PRETTY_EXPANDED, nil)
		fmt.Println()
	})

	// Output:
	// Object[{
	//   name => 'X',
	//   attributes => {
	//     'a' => Integer,
	//     'b' => {
	//       'type' => Optional[Integer],
	//       'value' => undef
	//     },
	//     'c' => {
	//       'type' => Optional[Integer[0, 255]],
	//       'value' => undef
	//     },
	//     'd' => {
	//       'type' => Optional[Integer[-32768, 32767]],
	//       'value' => undef
	//     },
	//     'e' => {
	//       'type' => Optional[Integer[0, 4294967295]],
	//       'value' => undef
	//     }
	//   }
	// }]
}

func ExampleReflect_TypeFromReflect_optionalIntReflect() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalInt{}))
		c.AddTypes(xm)
		w1 := int64(2)
		w2 := byte(3)
		w3 := int16(4)
		w4 := uint32(5)
		fmt.Println(eval.ToPrettyString(eval.Wrap(c, &optionalInt{1, &w1, &w2, &w3, &w4})))
	})

	// Output:
	// X(
	//   'a' => 1,
	//   'b' => 2,
	//   'c' => 3,
	//   'd' => 4,
	//   'e' => 5
	// )
}

func ExampleReflect_TypeFromReflect_optionalIntReflectZero() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalInt{}))
		c.AddTypes(xm)
		var w1 int64
		var w2 byte
		var w3 int16
		var w4 uint32
		fmt.Println(eval.ToPrettyString(eval.Wrap(c, &optionalInt{1, &w1, &w2, &w3, &w4})))
	})

	// Output:
	// X(
	//   'a' => 1,
	//   'b' => 0,
	//   'c' => 0,
	//   'd' => 0,
	//   'e' => 0
	// )
}

func ExampleReflect_TypeFromReflect_optionalIntCreate() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalInt{}))
		c.AddTypes(xm)
		fmt.Println(eval.ToPrettyString(eval.New(c, xm, types.WrapInteger(1), types.WrapInteger(2), types.WrapInteger(3), types.WrapInteger(4), types.WrapInteger(5))))
	})

	// Output:
	// X(
	//   'a' => 1,
	//   'b' => 2,
	//   'c' => 3,
	//   'd' => 4,
	//   'e' => 5
	// )
}

func ExampleReflect_TypeFromReflect_optionalIntDefault() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalInt{}))
		c.AddTypes(xm)
		fmt.Println(eval.ToPrettyString(eval.New(c, xm, types.WrapInteger(1))))
	})

	// Output:
	// X(
	//   'a' => 1
	// )
}

func ExampleReflect_TypeFromReflect_optionalIntDefaultZero() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&optionalInt{}))
		c.AddTypes(xm)
		var w1 int64
		var w2 byte
		var w3 int16
		var w4 uint32
		fmt.Println(eval.ToPrettyString(eval.New(c, xm, types.WrapInteger(1), types.WrapInteger(w1), types.WrapInteger(int64(w2)), types.WrapInteger(int64(w3)), types.WrapInteger(int64(w4)))))
	})

	// Output:
	// X(
	//   'a' => 1,
	//   'b' => 0,
	//   'c' => 0,
	//   'd' => 0,
	//   'e' => 0
	// )
}

type optionalFloat struct {
	A float64
	B *float64
	C *float32
}

func ExampleReflect_TypeFromReflect_optionalFloat() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(optionalFloat{}))
		c.AddTypes(xm)
		xm.ToString(os.Stdout, eval.PRETTY_EXPANDED, nil)
		fmt.Println()
	})

	// Output:
	// Object[{
	//   name => 'X',
	//   attributes => {
	//     'a' => Float,
	//     'b' => {
	//       'type' => Optional[Float],
	//       'value' => undef
	//     },
	//     'c' => {
	//       'type' => Optional[Float[-3.4028234663852886e+38, 3.4028234663852886e+38]],
	//       'value' => undef
	//     }
	//   }
	// }]
}

func ExampleReflect_TypeFromReflect_optionalFloatReflect() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(optionalFloat{}))
		c.AddTypes(xm)
		w1 := float64(2)
		w2 := float32(3)
		fmt.Println(eval.ToPrettyString(eval.Wrap(c, &optionalFloat{1, &w1, &w2})))
	})

	// Output:
	// X(
	//   'a' => 1.00000,
	//   'b' => 2.00000,
	//   'c' => 3.00000
	// )
}

func ExampleReflect_TypeFromReflect_optionalFloatReflectZero() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(optionalFloat{}))
		c.AddTypes(xm)
		var w1 float64
		var w2 float32
		fmt.Println(eval.ToPrettyString(eval.Wrap(c, &optionalFloat{1, &w1, &w2})))
	})

	// Output:
	// X(
	//   'a' => 1.00000,
	//   'b' => 0.00000,
	//   'c' => 0.00000
	// )
}

func ExampleReflect_TypeFromReflect_optionalFloatCreate() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(optionalFloat{}))
		c.AddTypes(xm)
		fmt.Println(eval.ToPrettyString(eval.New(c, xm, types.WrapFloat(1), types.WrapFloat(2), types.WrapFloat(3))))
	})

	// Output:
	// X(
	//   'a' => 1.00000,
	//   'b' => 2.00000,
	//   'c' => 3.00000
	// )
}

func ExampleReflect_TypeFromReflect_optionalFloatDefault() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(optionalFloat{}))
		c.AddTypes(xm)
		fmt.Println(eval.ToPrettyString(eval.New(c, xm, types.WrapFloat(1))))
	})

	// Output:
	// X(
	//   'a' => 1.00000
	// )
}

func ExampleReflect_TypeFromReflect_optionalFloatZero() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(optionalFloat{}))
		c.AddTypes(xm)
		var w1 float64
		var w2 float32
		fmt.Println(eval.ToPrettyString(eval.New(c, xm, types.WrapFloat(1), types.WrapFloat(w1), types.WrapFloat(float64(w2)))))
	})

	// Output:
	// X(
	//   'a' => 1.00000,
	//   'b' => 0.00000,
	//   'c' => 0.00000
	// )
}

type optionalIntSlice struct {
	A []int64
	B *[]int64
	C *map[string]int32
}

func ExampleReflect_TypeFromReflect_optionalIntSlice() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(optionalIntSlice{}))
		c.AddTypes(xm)
		xm.ToString(os.Stdout, eval.PRETTY_EXPANDED, nil)
		fmt.Println()
	})

	// Output:
	// Object[{
	//   name => 'X',
	//   attributes => {
	//     'a' => Array[Integer],
	//     'b' => {
	//       'type' => Optional[Array[Integer]],
	//       'value' => undef
	//     },
	//     'c' => {
	//       'type' => Optional[Hash[String, Integer[-2147483648, 2147483647]]],
	//       'value' => undef
	//     }
	//   }
	// }]
}

func ExampleReflect_TypeFromReflect_optionalIntSliceAssign() {
	eval.Puppet.Do(func(c eval.Context) {
		ois := &optionalIntSlice{}
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(ois))
		c.AddTypes(xm)

		ois.A = []int64{1, 2, 3}
		ois.B = &[]int64{4, 5, 6}
		ois.C = &map[string]int32{`a`: 7, `b`: 8, `c`: 9}
		eval.Wrap(c, ois).ToString(os.Stdout, eval.PRETTY_EXPANDED, nil)
		fmt.Println()
	})

	// Output:
	// X(
	//   'a' => [1, 2, 3],
	//   'b' => [4, 5, 6],
	//   'c' => {
	//     'a' => 7,
	//     'b' => 8,
	//     'c' => 9
	//   }
	// )
}

func ExampleReflect_TypeFromReflect_optionalIntSlicePuppetAssign() {
	eval.Puppet.Do(func(c eval.Context) {
		ois := &optionalIntSlice{}
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(ois))
		c.AddTypes(xm)

		pv := eval.New(c, xm, eval.Wrap(c, map[string]interface{}{
			`a`: []int64{1, 2, 3},
			`b`: []int64{4, 5, 6},
			`c`: map[string]int32{`a`: 7, `b`: 8, `c`: 9}}))
		pv.ToString(os.Stdout, eval.PRETTY_EXPANDED, nil)
		fmt.Println()
	})

	// Output:
	// X(
	//   'a' => [1, 2, 3],
	//   'b' => [4, 5, 6],
	//   'c' => {
	//     'a' => 7,
	//     'b' => 8,
	//     'c' => 9
	//   }
	// )
}

type structSlice struct {
	A []optionalIntSlice
	B *[]optionalIntSlice
	C *[]*optionalIntSlice
}

func ExampleReflect_TypeFromReflect_structSlice() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(optionalIntSlice{}))
		ym := c.Reflector().TypeFromReflect(`Y`, nil, reflect.TypeOf(structSlice{}))
		c.AddTypes(xm, ym)
		ym.ToString(os.Stdout, eval.PRETTY_EXPANDED, nil)
		fmt.Println()
	})

	// Output:
	// Object[{
	//   name => 'Y',
	//   attributes => {
	//     'a' => Array[X],
	//     'b' => {
	//       'type' => Optional[Array[X]],
	//       'value' => undef
	//     },
	//     'c' => {
	//       'type' => Optional[Array[Optional[X]]],
	//       'value' => undef
	//     }
	//   }
	// }]
}

func ExampleReflect_TypeFromReflect_structSliceAssign() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(optionalIntSlice{}))
		ym := c.Reflector().TypeFromReflect(`Y`, nil, reflect.TypeOf(structSlice{}))
		c.AddTypes(xm, ym)
		ss := eval.Wrap(c, structSlice{
			A: []optionalIntSlice{{[]int64{1, 2, 3}, &[]int64{4, 5, 6}, &map[string]int32{`a`: 7, `b`: 8, `c`: 9}}},
			B: &[]optionalIntSlice{{[]int64{11, 12, 13}, &[]int64{14, 15, 16}, &map[string]int32{`a`: 17, `b`: 18, `c`: 19}}},
			C: &[]*optionalIntSlice{{[]int64{21, 22, 23}, &[]int64{24, 25, 26}, &map[string]int32{`a`: 27, `b`: 28, `c`: 29}}},
		})
		ss.ToString(os.Stdout, eval.PRETTY_EXPANDED, nil)
		fmt.Println()
	})

	// Output:
	// Y(
	//   'a' => [
	//     X(
	//       'a' => [1, 2, 3],
	//       'b' => [4, 5, 6],
	//       'c' => {
	//         'a' => 7,
	//         'b' => 8,
	//         'c' => 9
	//       }
	//     )],
	//   'b' => [
	//     X(
	//       'a' => [11, 12, 13],
	//       'b' => [14, 15, 16],
	//       'c' => {
	//         'a' => 17,
	//         'b' => 18,
	//         'c' => 19
	//       }
	//     )],
	//   'c' => [
	//     X(
	//       'a' => [21, 22, 23],
	//       'b' => [24, 25, 26],
	//       'c' => {
	//         'a' => 27,
	//         'b' => 28,
	//         'c' => 29
	//       }
	//     )]
	// )
}

// For third party vendors using anonymous fields to tag structs
type anon struct {
	_ struct{} `some:"tag here"`
	A string
}

func ExampleReflect_TypeFromReflect_structAnonField() {
	eval.Puppet.Do(func(c eval.Context) {
		x := c.Reflector().TypeFromReflect(`X`, nil, reflect.TypeOf(&anon{}))
		c.AddTypes(x)
		x.ToString(os.Stdout, eval.PRETTY_EXPANDED, nil)
		fmt.Println()
	})

	// Output:
	// Object[{
	//   name => 'X',
	//   attributes => {
	//     'a' => String
	//   }
	// }]
}
