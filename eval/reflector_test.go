package eval_test

import (
	"strconv"
	"github.com/puppetlabs/go-evaluator/eval"
	"fmt"
	"reflect"
	"os"
	"github.com/puppetlabs/go-semver/semver"
	"time"
	"github.com/puppetlabs/go-evaluator/types"
	"regexp"

	// Initialize pcore
	_ "github.com/puppetlabs/go-evaluator/pcore"
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
		fv := eval.Wrap(nil, float64(10+i+1) / 10.0)
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

func ExampleReflector_objectTypeFromReflect() {
	type TestAddress struct {
		Street string
		Zip    string `puppet:"name=>zip_code"`
	}
	type TestPerson struct {
		Name    string
		Address *TestAddress
	}
	type TestExtendedPerson struct {
		TestPerson
		Age    int  `puppet:"type=>Optional[Integer],value=>undef"`
		Active bool `puppet:"name=>enabled"`
	}

	c := eval.Puppet.RootContext()
	rtAddress := reflect.TypeOf(&TestAddress{})
	rtPerson := reflect.TypeOf(&TestPerson{})
	rtExtPerson := reflect.TypeOf(&TestExtendedPerson{})

	rf := c.Reflector()
	tAddress := rf.ObjectTypeFromReflect(`My::Address`, nil, rtAddress)
	tPerson := rf.ObjectTypeFromReflect(`My::Person`, nil, rtPerson)
	tExtPerson := rf.ObjectTypeFromReflect(`My::ExtendedPerson`, tPerson, rtExtPerson)
	c.AddTypes(tAddress, tPerson, tExtPerson)

	fmt.Println(tAddress)
	fmt.Println(tPerson)
	fmt.Println(tExtPerson)

	ts := &TestExtendedPerson{TestPerson{`Bob Tester`, &TestAddress{`Example Road 23`, `12345`}}, 34, true}
	ev := eval.Wrap(c, ts)
	fmt.Println(ev)

	// Output:
	// Object[{name => 'My::Address', attributes => {'street' => String, 'zip_code' => String}}]
	// Object[{name => 'My::Person', attributes => {'name' => String, 'address' => My::Address}}]
	// Object[{name => 'My::ExtendedPerson', parent => My::Person, attributes => {'age' => {'type' => Optional[Integer], 'value' => undef}, 'enabled' => Boolean}}]
	// My::ExtendedPerson('name' => 'Bob Tester', 'address' => My::Address('street' => 'Example Road 23', 'zip_code' => '12345'), 'enabled' => true, 'age' => 34)
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

func ExampleReflector_objectTypeFromReflectInterface() {
	c := eval.Puppet.RootContext()

	// Create ObjectType from reflected type
	xa := c.Reflector().ObjectTypeFromReflect(`X::A`, nil, reflect.TypeOf((*A)(nil)).Elem())
	xb := c.Reflector().ObjectTypeFromReflect(`X::B`, nil, reflect.TypeOf(B(0)))

	// Ensure that the type is resolved
	c.AddTypes(xa, xb)

	// Print the created Interface Type in human readable form
	xa.ToString(os.Stdout, eval.PRETTY, nil)
	fmt.Println()

	// Print the created Implementation Type in human readable form
	xb.ToString(os.Stdout, eval.PRETTY, nil)
	fmt.Println()

	// Invoke method 'x' on the interface on a receiver
	m, _ := xb.Member(`x`)
	gm := m.(eval.CallableGoMember)

	fmt.Println(gm.CallGo(c, NewA(32)))
	fmt.Println(gm.CallGo(c, B(25)))

	// Invoke method 'x' on the interface on a receiver
	m, _ = xb.Member(`y`)

	// Call Go function using CallableGoMember
	gv := m.(eval.CallableGoMember).CallGo(c, B(25), NewA(15))
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
	Age    int  `puppet:"type=>Optional[Integer],value=>undef"`
	Active bool `puppet:"name=>enabled"`
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
		typeSet.ToString(os.Stdout, eval.PRETTY, nil)
		fmt.Println()

		// Create an instance of something included in the TypeSet
		ad := &Address{`Example Road 23`, `12345`}
		ep := &ExtendedPerson{Person{`Bob Tester`, ad}, 34, true}

		// Wrap the instance as a Value and print it
		v := eval.Wrap(c, ep)
		fmt.Println(v)

		m, _ := v.PType().(eval.TypeWithCallableMembers).Member(`visit`)
		fmt.Println(m.(eval.CallableGoMember).CallGo(c, ep, ad))
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
	//         'address' => Address
	//       },
	//       functions => {
	//         'visit' => Callable[
	//           [Address],
	//           String]
	//       }
	//     },
	//     ExtendedPerson => Person{
	//       attributes => {
	//         'age' => {
	//           'type' => Optional[Integer],
	//           'value' => undef
	//         },
	//         'enabled' => Boolean
	//       }
	//     }
	//   }
	// }]
	// My::Own::ExtendedPerson('name' => 'Bob Tester', 'address' => My::Own::Address('street' => 'Example Road 23', 'zip_code' => '12345'), 'enabled' => true, 'age' => 34)
	// visited Example Road 23
	// visited Example Road 23
}

type valueStruct struct {
	X eval.OrderedMap
	Y *types.ArrayValue
	P eval.PuppetObject
	O eval.Object
}

func (v *valueStruct) Get(key *types.IntegerValue, dflt eval.Value) *types.StringValue {
	return v.X.Get2(key, eval.UNDEF).(*types.StringValue)
}

func ExampleReflector_reflectPType() {
	eval.Puppet.Do(func(c eval.Context) {
		xm := c.Reflector().ObjectTypeFromReflect(`X::M`, nil, reflect.TypeOf(&valueStruct{}))
		c.AddTypes(xm)
		xm.ToString(os.Stdout, eval.PRETTY, nil)
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
