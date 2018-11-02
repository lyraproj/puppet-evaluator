package eval_test

import (
	"fmt"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-semver/semver"
	"reflect"
	"testing"

	// Initialize pcore
	_ "github.com/puppetlabs/go-evaluator/pcore"
)

func ExampleWrap() {
	// Wrap native Go types
	str := eval.Wrap(nil, "hello")
	idx := eval.Wrap(nil, 23)
	flt := eval.Wrap(nil, 123.34)
	bl := eval.Wrap(nil, true)
	und := eval.UNDEF

	fmt.Printf("'%s' is a %s\n", str, str.PType())
	fmt.Printf("'%s' is a %s\n", idx, idx.PType())
	fmt.Printf("'%s' is a %s\n", flt, flt.PType())
	fmt.Printf("'%s' is a %s\n", bl, bl.PType())
	fmt.Printf("'%s' is a %s\n", und, und.PType())

	// Output:
	// 'hello' is a String
	// '23' is a Integer[23, 23]
	// '123.34' is a Float[123.340, 123.340]
	// 'true' is a Boolean
	// 'undef' is a Undef
}

func ExampleWrap_slice() {
	// Wrap native Go slice
	arr := eval.Wrap(nil, []interface{}{1, 2.2, true, nil, "hello"})
	fmt.Printf("%s is an %s\n", arr, arr.PType())

	// Output:
	// [1, 2.20000, true, undef, 'hello'] is an Array[5, 5]
}

func ExampleWrap_hash() {
	// Wrap native Go hash
	hsh := eval.Wrap(nil, map[string]interface{}{
		"first":  1,
		"second": 2.2,
		"third":  "three",
		"nested": []string{"hello", "world"},
	})
	nst, _ := hsh.(eval.OrderedMap).Get4("nested")
	fmt.Printf("'%s' is a %s\n", hsh, hsh.PType())
	fmt.Printf("hsh['nested'] == %s, an instance of %s\n", nst, nst.PType())

	// Output:
	// '{'first' => 1, 'nested' => ['hello', 'world'], 'second' => 2.20000, 'third' => 'three'}' is a Hash[Enum['first', 'nested', 'second', 'third'], Data, 4, 4]
	// hsh['nested'] == ['hello', 'world'], an instance of Array[Enum['hello', 'world'], 2, 2]
}

func ExamplePcore_parseType() {
	eval.Puppet.Do(func(ctx eval.Context) error {
		pcoreType := ctx.ParseType2("Enum[foo,fee,fum]")
		fmt.Printf("%s is an instance of %s\n", pcoreType, pcoreType.PType())
		return nil
	})
	// Output:
	// Enum['foo', 'fee', 'fum'] is an instance of Type[Enum['foo', 'fee', 'fum']]
}

func ExamplePcore_isInstance() {
	eval.Puppet.Do(func(ctx eval.Context) error {
		pcoreType := ctx.ParseType2("Enum[foo,fee,fum]")
		fmt.Println(eval.IsInstance(pcoreType, eval.Wrap(ctx, "foo")))
		fmt.Println(eval.IsInstance(pcoreType, eval.Wrap(ctx, "bar")))
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


func ExampleObjectType_fromReflectedValue() {
	type TestStruct struct {
		Message   string
		Kind      string
		IssueCode string
	}

	c := eval.Puppet.RootContext()
	ts := &TestStruct{`the message`, `THE_KIND`, `THE_CODE`}
	et, _ := eval.Load(c, eval.NewTypedName(eval.TYPE, `Error`))
	ev := et.(eval.ObjectType).FromReflectedValue(c, reflect.ValueOf(ts).Elem())
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
	ev := eval.Wrap(c, ts)
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
	ev := eval.Wrap(c, ts)
	fmt.Println(ev)
	// Output: My::Person('name' => 'Bob Tester', 'age' => 34, 'address' => My::Address('street' => 'Example Road 23', 'zip_code' => '12345'), 'enabled' => true)
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

	eval.Puppet.Do(func(c eval.Context) error {
		ir := c.ImplementationRegistry()
		ir.RegisterType(c, `My::Address`, reflect.TypeOf(&ObscurelyNamedAddress{}))

		typeSet := c.Reflector().TypeSetFromReflect(`My`, semver.MustParseVersion(`1.0.0`),
			reflect.TypeOf(&ObscurelyNamedAddress{}), reflect.TypeOf(&Person{}))
		c.AddTypes(typeSet)
		tss := typeSet.String()
		exp := `TypeSet[{pcore_uri => 'http://puppet.com/2016.1/pcore', pcore_version => '1.0.0', name_authority => 'http://puppet.com/2016.1/runtime', name => 'My', version => '1.0.0', types => {Address => {attributes => {'street' => String, 'zip_code' => String}}, Person => {attributes => {'name' => String, 'address' => Address}}}}]`
		if tss != exp {
			t.Errorf("Expected %s, got %s\n", exp, tss)
		}
		return nil
	})
}