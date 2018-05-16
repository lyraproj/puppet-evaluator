package eval_test

import (
	"fmt"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-semver/semver"
	"os"
	"reflect"
	"regexp"
	"time"

	// Initialize pcore
	_ "github.com/puppetlabs/go-evaluator/pcore"
	"testing"
)

func ExampleWrap() {
	// Wrap native Go types
	str := eval.Wrap("hello")
	idx := eval.Wrap(23)
	flt := eval.Wrap(123.34)
	bl := eval.Wrap(true)
	und := eval.UNDEF

	fmt.Printf("'%s' is a %s\n", str, str.Type())
	fmt.Printf("'%s' is a %s\n", idx, idx.Type())
	fmt.Printf("'%s' is a %s\n", flt, flt.Type())
	fmt.Printf("'%s' is a %s\n", bl, bl.Type())
	fmt.Printf("'%s' is a %s\n", und, und.Type())

	// Output:
	// 'hello' is a String
	// '23' is a Integer[23, 23]
	// '123.34' is a Float[123.340, 123.340]
	// 'true' is a Boolean
	// 'undef' is a Undef
}

func ExampleWrap_slice() {
	// Wrap native Go slice
	arr := eval.Wrap([]interface{}{1, 2.2, true, nil, "hello"})
	fmt.Printf("%s is an %s\n", arr, arr.Type())

	// Output:
	// [1, 2.20000, true, undef, 'hello'] is an Array[5, 5]
}

func ExampleWrap_hash() {
	// Wrap native Go hash
	hsh := eval.Wrap(map[string]interface{}{
		"first":  1,
		"second": 2.2,
		"third":  "three",
		"nested": []string{"hello", "world"},
	})
	nst, _ := hsh.(eval.KeyedValue).Get4("nested")
	fmt.Printf("'%s' is a %s\n", hsh, hsh.Type())
	fmt.Printf("hsh['nested'] == %s, an instance of %s\n", nst, nst.Type())

	// Output:
	// '{'first' => 1, 'nested' => ['hello', 'world'], 'second' => 2.20000, 'third' => 'three'}' is a Hash[Enum['first', 'nested', 'second', 'third'], Data, 4, 4]
	// hsh['nested'] == ['hello', 'world'], an instance of Array[Enum['hello', 'world'], 2, 2]
}

func ExamplePcore_parseType() {
	eval.Puppet.Do(func(ctx eval.Context) error {
		pcoreType := ctx.ParseType2("Enum[foo,fee,fum]")
		fmt.Printf("%s is an instance of %s\n", pcoreType, pcoreType.Type())
		return nil
	})
	// Output:
	// Enum['foo', 'fee', 'fum'] is an instance of Type[Enum['foo', 'fee', 'fum']]
}

func ExamplePcore_isInstance() {
	eval.Puppet.Do(func(ctx eval.Context) error {
		pcoreType := ctx.ParseType2("Enum[foo,fee,fum]")
		fmt.Println(eval.IsInstance(ctx, pcoreType, eval.Wrap("foo")))
		fmt.Println(eval.IsInstance(ctx, pcoreType, eval.Wrap("bar")))
		return nil
	})
	// Output:
	// true
	// false
}

func ExamplePcore_parseTypeError() {
	err := eval.Puppet.Do(func(ctx eval.Context) error {
		ctx.ParseType2("Enum[foo") // Missing end bracket
		return nil
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output: expected one of ',' or ']', got 'EOF' (line: 1, column: 9)
}

func ExampleReflector_reflectArray() {
	c := eval.Puppet.RootContext()

	av := eval.Wrap([]interface{}{`hello`, 3}).(*types.ArrayValue)
	ar := c.Reflector().Reflect(av, nil)
	fmt.Printf("%s %v\n", ar.Type(), ar)

	av = eval.Wrap([]interface{}{`hello`, `world`}).(*types.ArrayValue)
	ar = c.Reflector().Reflect(av, nil)
	fmt.Printf("%s %v\n", ar.Type(), ar)
	// Output:
	// []interface {} [hello 3]
	// []string [hello world]
}

func ExampleReflector_reflectToArray() {
	type TestStruct struct {
		Strings    []string
		Interfaces []interface{}
		Values     []eval.PValue
	}

	c := eval.Puppet.RootContext()
	ts := &TestStruct{}
	rv := reflect.ValueOf(ts).Elem()

	av := eval.Wrap([]string{`hello`, `world`})

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
	var values map[string]eval.PValue

	c := eval.Puppet.RootContext()
	hv := eval.Wrap(map[string]string{`x`: `hello`, `y`: `world`})

	rf := c.Reflector()
	rf.ReflectTo(hv, reflect.ValueOf(&strings))
	rf.ReflectTo(hv, reflect.ValueOf(&interfaces))
	rf.ReflectTo(hv, reflect.ValueOf(&values))
	fmt.Printf("%s %s\n", strings[`x`], strings[`y`])
	fmt.Printf("%s %s\n", interfaces[`x`], interfaces[`y`])
	fmt.Printf("%s %s\n", values[`x`], values[`y`])

	rf.ReflectTo(eval.UNDEF, reflect.ValueOf(&strings))
	rf.ReflectTo(eval.UNDEF, reflect.ValueOf(&interfaces))
	rf.ReflectTo(eval.UNDEF, reflect.ValueOf(&values))
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
	bv := eval.Wrap([]byte{1, 2, 3})
	rf.ReflectTo(bv, reflect.ValueOf(&buf))
	fmt.Println(buf)

	rf.ReflectTo(eval.UNDEF, reflect.ValueOf(&buf))
	fmt.Println(buf)
	// Output:
	// [1 2 3]
	// []
}

func ExampleReflector_reflectToFloat() {
	type TestStruct struct {
		SmallFloat float32
		BigFloat   float64
		APValue    eval.PValue
		IValue     interface{}
	}

	c := eval.Puppet.RootContext()
	rf := c.Reflector()
	ts := &TestStruct{}
	rv := reflect.ValueOf(ts).Elem()
	n := rv.NumField()
	for i := 0; i < n; i++ {
		fv := eval.Wrap(float64(10+i+1) / 10.0)
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
		APValue  eval.PValue
		IValue   interface{}
	}

	c := eval.Puppet.RootContext()
	rf := c.Reflector()
	ts := &TestStruct{}
	rv := reflect.ValueOf(ts).Elem()
	n := rv.NumField()
	for i := 0; i < n; i++ {
		rf.ReflectTo(eval.Wrap(10+i), rv.Field(i))
	}
	fmt.Println(ts)
	// Output: &{10 11 12 13 14 15 16 17 18 19 20 21}
}

func ExampleReflector_reflectToRegexp() {
	var expr *regexp.Regexp

	rx := eval.Wrap(regexp.MustCompile("[a-z]*"))
	eval.Puppet.RootContext().Reflector().ReflectTo(rx, reflect.ValueOf(&expr))

	fmt.Println(expr)
	// Output: [a-z]*
}

func ExampleReflector_reflectToTimespan() {
	var span time.Duration

	rx := eval.Wrap(time.Duration(1234))
	eval.Puppet.RootContext().Reflector().ReflectTo(rx, reflect.ValueOf(&span))

	fmt.Println(span)
	// Output: 1.234µs
}

func ExampleReflector_reflectToTimestamp() {
	var tx time.Time

	tm, _ := time.Parse(time.RFC3339, "2018-05-11T06:31:22+01:00")
	tv := eval.Wrap(tm)
	eval.Puppet.RootContext().Reflector().ReflectTo(tv, reflect.ValueOf(&tx))

	fmt.Println(tx.Format(time.RFC3339))
	// Output: 2018-05-11T06:31:22+01:00
}

func ExampleReflector_reflectToTersion() {
	var version semver.Version

	ver, _ := semver.ParseVersion(`1.2.3`)
	vv := eval.Wrap(ver)
	eval.Puppet.RootContext().Reflector().ReflectTo(vv, reflect.ValueOf(&version))

	fmt.Println(version)
	// Output: 1.2.3
}

func ExampleReflector_reflectToPuppetObject() {
	type TestStruct struct {
		Message   string
		Kind      string
		IssueCode string
	}

	c := eval.Puppet.RootContext()
	ts := &TestStruct{}

	ev := eval.NewError(c, `the message`, `THE_KIND`, `THE_CODE`, nil, nil)
	c.Reflector().ReflectTo(ev, reflect.ValueOf(ts))
	fmt.Printf("message: %s, kind %s, issueCode %s\n", ts.Message, ts.Kind, ts.IssueCode)
	// Output: message: the message, kind THE_KIND, issueCode THE_CODE
}

func ExampleObjectType_fromReflectedValue() {
	type TestStruct struct {
		Message   string
		Kind      string
		IssueCode string
	}

	c := eval.Puppet.RootContext()
	ts := &TestStruct{`the message`, `THE_KIND`, `THE_CODE`}
	et, _ := eval.Load(c, eval.NewTypedName(eval.TYPE, `Error`))
	ev := et.(eval.ObjectType).FromReflectedValue(c, reflect.ValueOf(ts))
	fmt.Println(ev)
	// Output: Error('message' => 'the message', 'kind' => 'THE_KIND', 'issue_code' => 'THE_CODE')
}

func ExampleImplementationRegistry() {
	type TestAddress struct {
		Street string
		Zip    string
	}
	type TestPerson struct {
		Name    string
		Age     int
		Address *TestAddress
		Active  bool
	}

	c := eval.Puppet.RootContext()
	c.AddDefinitions(c.ParseAndValidate(``, `
  type My::Address = {
    attributes => {
      street => String,
      zip => String,
    }
  }
  type My::Person = {
		attributes => {
      name => String,
      age => Integer,
      address => My::Address,
      active => Boolean,
		}
  }`, false))
	c.ResolveDefinitions()

	ir := c.ImplementationRegistry()
	ir.RegisterType(c, `My::Address`, reflect.TypeOf(&TestAddress{}))
	ir.RegisterType(c, `My::Person`, reflect.TypeOf(&TestPerson{}))

	ts := &TestPerson{`Bob Tester`, 34, &TestAddress{`Example Road 23`, `12345`}, true}
	ev := eval.Wrap2(c, ts)
	fmt.Println(ev)
	// Output: My::Person('name' => 'Bob Tester', 'age' => 34, 'address' => My::Address('street' => 'Example Road 23', 'zip' => '12345'), 'active' => true)
}

func ExampleImplementationRegistry_tags() {
	type TestAddress struct {
		Street string
		Zip    string `puppet:"name=>zip_code"`
	}
	type TestPerson struct {
		Name    string
		Age     int
		Address *TestAddress
		Active  bool `puppet:"name=>enabled"`
	}

	c := eval.Puppet.RootContext()
	c.AddDefinitions(c.ParseAndValidate(``, `
  type My::Address = {
    attributes => {
      street => String,
      zip_code => Optional[String],
    }
  }
  type My::Person = {
		attributes => {
      name => String,
      age => Integer,
      address => My::Address,
      enabled => Boolean,
		}
  }`, false))
	c.ResolveDefinitions()

	ir := c.ImplementationRegistry()
	ir.RegisterType(c, `My::Address`, reflect.TypeOf(&TestAddress{}))
	ir.RegisterType(c, `My::Person`, reflect.TypeOf(&TestPerson{}))

	ts := &TestPerson{`Bob Tester`, 34, &TestAddress{`Example Road 23`, `12345`}, true}
	ev := eval.Wrap2(c, ts)
	fmt.Println(ev)
	// Output: My::Person('name' => 'Bob Tester', 'age' => 34, 'address' => My::Address('street' => 'Example Road 23', 'zip_code' => '12345'), 'enabled' => true)
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
	ev := eval.Wrap2(c, ts)
	fmt.Println(ev)

	// Output:
	// Object{name => 'My::Address', attributes => {'street' => String, 'zip_code' => String}}
	// Object{name => 'My::Person', attributes => {'name' => String, 'address' => My::Address}}
	// Object{name => 'My::ExtendedPerson', parent => My::Person, attributes => {'age' => {'type' => Optional[Integer], 'value' => undef}, 'enabled' => Boolean}}
	// My::ExtendedPerson('name' => 'Bob Tester', 'address' => My::Address('street' => 'Example Road 23', 'zip_code' => '12345'), 'enabled' => true, 'age' => 34)
}

func ExampleReflector_typeSetFromReflect() {
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

	c := eval.Puppet.RootContext()

	// Create a TypeSet from a list of Go structs
	typeSet := c.Reflector().TypeSetFromReflect(`My`, semver.MustParseVersion(`1.0.0`),
		reflect.TypeOf(&Address{}), reflect.TypeOf(&Person{}), reflect.TypeOf(&ExtendedPerson{}))

	// Make the types known to the current loader
	c.AddTypes(typeSet)

	// Print the TypeSet in human readable form
	typeSet.ToString(os.Stdout, eval.PRETTY, nil)
	fmt.Println()

	// Create an instance of something included in the TypeSet
	ep := &ExtendedPerson{Person{`Bob Tester`, &Address{`Example Road 23`, `12345`}}, 34, true}

	// Wrap the instance as a PValue and print it
	fmt.Println(eval.Wrap2(c, ep))

	// Output:
	// TypeSet{
	//   pcore_uri => 'http://puppet.com/2016.1/pcore',
	//   pcore_version => '1.0.0',
	//   name_authority => 'http://puppet.com/2016.1/runtime',
	//   name => 'My',
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
	// }
	// My::ExtendedPerson('name' => 'Bob Tester', 'address' => My::Address('street' => 'Example Road 23', 'zip_code' => '12345'), 'enabled' => true, 'age' => 34)
}

func TestReflectorAndImplRepo(t *testing.T) {
	type ObscurelyNamedAddress struct {
		Street string
		Zip    string `puppet:"name=>zip_code"`
	}
	type Person struct {
		Name    string
		Address *ObscurelyNamedAddress
	}

	c := eval.Puppet.RootContext()
	ir := c.ImplementationRegistry()
	ir.RegisterType(c, `My::Address`, reflect.TypeOf(&ObscurelyNamedAddress{}))

	typeSet := c.Reflector().TypeSetFromReflect(`My`, semver.MustParseVersion(`1.0.0`),
		reflect.TypeOf(&ObscurelyNamedAddress{}), reflect.TypeOf(&Person{}))
	c.AddTypes(typeSet)
	tss := typeSet.String()
	exp := `TypeSet{pcore_uri => 'http://puppet.com/2016.1/pcore', pcore_version => '1.0.0', name_authority => 'http://puppet.com/2016.1/runtime', name => 'My', version => '1.0.0', types => {Address => {attributes => {'street' => String, 'zip_code' => String}}, Person => {attributes => {'name' => String, 'address' => Address}}}}`
	if tss != exp {
		t.Errorf("Expected %s, got %s\n", exp, tss)
	}
}