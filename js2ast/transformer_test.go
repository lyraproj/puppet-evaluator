package js2ast

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"fmt"

	// Initialize pcore
	_ "github.com/puppetlabs/go-evaluator/pcore"
	"github.com/puppetlabs/go-semver/semver"
	"reflect"
	"testing"
	"io/ioutil"
	"github.com/puppetlabs/go-parser/parser"
)

func ExampleEvaluateJavaScript_binOp() {
	err := eval.Puppet.Try(func(c eval.Context) error {
		v, err := EvaluateJavaScript(c, `/test/file.js`, []byte(`var a = 23.5; var b = 8; a + b`))
		if err == nil {
			fmt.Printf(`%s %s`, v.PType(), v.String())
		}
		return err
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output: Float[31.5000, 31.5000] 31.5
}

func ExampleEvaluateJavaScript_func() {
	err := eval.Puppet.Try(func(c eval.Context) error {
		v, err := EvaluateJavaScript(c, `/test/file.js`, []byte(`function sum(a, b) { return a + b } sum(2, 3)`))
		if err == nil {
			fmt.Printf(`%s %s`, v.PType(), v.String())
		}
		return err
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output: Integer[5, 5] 5
}

func ExampleEvaluateJavaScript_obj() {
	err := eval.Puppet.Try(func(c eval.Context) error {
		v, err := EvaluateJavaScript(c, `/test/file.js`, []byte(`var p = { a:32, b:"hello" }; p;`))
		if err == nil {
			fmt.Println(v)
		}
		return err
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output: {'a' => 32, 'b' => 'hello'}
}

func ExampleEvaluateJavaScript_new() {
	type Baz struct {
		Region string
		Tags []string
	}

	bt := &Baz{}
	err := eval.Puppet.Try(func(c eval.Context) error {
		ts := c.Reflector().TypeSetFromReflect(`Foo::Bar`, semver.MustParseVersion(`1.0.0`), nil, reflect.TypeOf(bt))
		c.AddTypes(ts)
		v, err := EvaluateJavaScript(c, `/test/file.js`, []byte(`
var tags = ['a', 'b'];
var region = 'us-west';
new Foo.Bar.Baz({ region: region, tags: tags });
`))
		if err == nil {
			fmt.Println(v)
		}
		return err
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output: Foo::Bar::Baz('region' => 'us-west', 'tags' => ['a', 'b'])
}

func ExampleEvaluateJavaScript_lambda() {
	err := eval.Puppet.Try(func(c eval.Context) error {
		v, err := EvaluateJavaScript(c, `/test/file.js`, []byte(`[1, 2, 3].reduce(function(v, m) { return v + m; });`))
		if err == nil {
			fmt.Println(v)
		}
		return err
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output: 6
}

func ExampleEvaluateJavaScript_console() {
	err := eval.Puppet.Try(func(c eval.Context) error {
		_, err := EvaluateJavaScript(c, `/test/file.js`, []byte(`console.info('hello');`))
		return err
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output: info: hello
}

func ExampleEvaluateJavaScript_forInArray() {
	err := eval.Puppet.Try(func(c eval.Context) error {
		_, err := EvaluateJavaScript(c, `/test/file.js`, []byte(`
for(x in ['a', 'b', 'c']) {
  console.log(x);
}`))
		return err
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// notice: a
	// notice: b
	// notice: c
}

func ExampleEvaluateJavaScript_forInObject() {
	err := eval.Puppet.Try(func(c eval.Context) error {
		_, err := EvaluateJavaScript(c, `/test/file.js`, []byte(`
for(x in { a: 'A', b: 'B', c: 'C'}) {
  if(x == 'b')
    continue;
  console.log(x);
}`))
		return err
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// notice: a
	// notice: c
}

func ExampleEvaluateJavaScript_forInString() {
	err := eval.Puppet.Try(func(c eval.Context) error {
		_, err := EvaluateJavaScript(c, `/test/file.js`, []byte(`
for(x in 'abc') {
  console.log(x);
}`))
		return err
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// notice: a
	// notice: b
	// notice: c
}

func ExampleEvaluateJavaScript_preInc() {
	err := eval.Puppet.Try(func(c eval.Context) error {
		_, err := EvaluateJavaScript(c, `/test/file.js`, []byte(`
function f() {
  var i = 1;
  console.log(++i);
  console.log(i);
}
f();
`))
		return err
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// notice: 2
	// notice: 2
}

func ExampleEvaluateJavaScript_postInc() {
	err := eval.Puppet.Try(func(c eval.Context) error {
		_, err := EvaluateJavaScript(c, `/test/file.js`, []byte(`
function f() {
  var i = 1;
  console.log(i++);
  console.log(i);
}
f();
`))
		return err
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// notice: 1
	// notice: 2
}

func ExampleEvaluateJavaScript_addEq() {
	err := eval.Puppet.Try(func(c eval.Context) error {
		_, err := EvaluateJavaScript(c, `/test/file.js`, []byte(`
function f() {
  var i = 1;
  console.log(i);
  i += 2;
  console.log(i);
}
f();
`))
		return err
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// notice: 1
	// notice: 3
}

func ExampleEvaluateJavaScript_for() {
	err := eval.Puppet.Try(func(c eval.Context) error {
		_, err := EvaluateJavaScript(c, `/test/file.js`, []byte(`
for(var i = 0; i < 4; i++) {
  if(i == 1) {
    continue;
  } else if(i == 3) {
    break;
  }
  console.log(i);
}`))
		return err
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// notice: 0
	// notice: 2
}

func ExampleEvaluateJavaScript_while() {
	err := eval.Puppet.Try(func(c eval.Context) error {
		_, err := EvaluateJavaScript(c, `/test/file.js`, []byte(`
function f(v) {
  while(v > 0) {
    console.log(--v);
  }
}
f(3);
`))
		return err
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// notice: 2
	// notice: 1
	// notice: 0
}

func ExampleEvaluateJavaScript_doWhile() {
	err := eval.Puppet.Try(func(c eval.Context) error {
		_, err := EvaluateJavaScript(c, `/test/file.js`, []byte(`
function f(v) {
  do {
    console.log(--v);
  } while(v > 0)
}
f(3);
`))
		return err
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// notice: 2
	// notice: 1
	// notice: 0
}

func ExampleEvaluateJavaScript_if() {
	err := eval.Puppet.Try(func(c eval.Context) error {
		_, err := EvaluateJavaScript(c, `/test/file.js`, []byte(`
for(x in [1,2,3]) {
  if(x == 1)
    console.info(x);
  else {
    console.log(x);
    break;
  }
}`))
		return err
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// info: 1
	// notice: 2
}

func ExampleEvaluateJavaScript_ternary() {
	err := eval.Puppet.Try(func(c eval.Context) error {
		_, err := EvaluateJavaScript(c, `/test/file.js`, []byte(`
for(x in [1,2]) {
  x > 1 ? console.log(x) : console.info(x);
}`))
		return err
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// info: 1
	// notice: 2
}

func ExampleEvaluateJavaScript_switch() {
	err := eval.Puppet.Try(func(c eval.Context) error {
		_, err := EvaluateJavaScript(c, `/test/file.js`, []byte(`
for(x in [1,2,3]) {
  switch(x) {
  case 1:
    console.log('one');
    break;
  default:
    console.log('default');
  case 2:
    console.log('two');
  }
}`))
		return err
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// notice: one
	// notice: two
	// notice: default
	// notice: two
}

/*
TODO: Implement constructor to make this example work
func ExampleEvaluateJavaScript_ctor() {
	err := eval.Puppet.Try(func(c eval.Context) error {
		_, err := EvaluateJavaScript(c, `/test/file.js`, []byte(`
function MyType(num, str, cond) {
  this.num = num;
  this.str = str;
  if(cond > 0) {
    this.cond = true;
  } else {
    this.cond = false;
  }
}
var x = new MyType(1, 2, 3);
console.log(x);
`))
		return err
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// notice: one
	// notice: two
	// notice: default
	// notice: two
}
*/

func ExampleEvaluateJavaScript_instance() {
	err := eval.Puppet.Try(func(c eval.Context) error {
		_, err := EvaluateJavaScript(c, `/test/file.js`, []byte(`
console.info('hello' instanceof String);
console.info(23 instanceof String);
console.info({a: 'A'} instanceof Hash);
`))
		return err
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// info: true
	// info: false
	// info: true
}

func ExampleEvaluateJavaScript_eq() {
	err := eval.Puppet.Try(func(c eval.Context) error {
		_, err := EvaluateJavaScript(c, `/test/file.js`, []byte(`
console.log('a' == 'a')
console.log('a' == 'A')
console.log('1' == 1);
console.log(true == 1);
console.log(1.0 == 1);
console.log('1' === 1);
console.log(true === 1);
console.log(1.0 === 1);
console.log('a' != 'a')
console.log('a' != 'A')
console.log('1' != 1);
console.log(true != 1);
console.log(1.0 != 1);
console.log('1' !== 1);
console.log(true !== 1);
console.log(1.0 !== 1);
`))
		return err
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// notice: true
	// notice: false
	// notice: true
	// notice: true
	// notice: true
	// notice: false
	// notice: false
	// notice: true
	// notice: false
	// notice: true
	// notice: false
	// notice: false
	// notice: false
	// notice: true
	// notice: true
	// notice: false
}

func ExampleEvaluateJavaScript_cmp() {
	err := eval.Puppet.Try(func(c eval.Context) error {
		_, err := EvaluateJavaScript(c, `/test/file.js`, []byte(`
console.log('a' > 'a')
console.log('a' > 'A')
`))
		return err
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// notice: false
	// notice: true
}

func ExampleEvaluateJavaScript_in() {
	err := eval.Puppet.Try(func(c eval.Context) error {
		_, err := EvaluateJavaScript(c, `/test/file.js`, []byte(`
var car = {make: 'Honda', model: 'Accord', year: 1998};
console.log('make' in car)
console.log('Honda' in car)
console.log('Honda' in car.values())
`))
		return err
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// notice: true
	// notice: false
	// notice: true
}

func ExampleEvaluateJavaScript_sequential() {
	err := eval.Puppet.Try(func(c eval.Context) error {
		_, err := EvaluateJavaScript(c, `/test/file.js`, []byte(`
function tier1(){
  console.log("tier1 calling");
}
 
function tier2() {
  console.log("tier2 calling");
}
 
sequential(tier1,tier2)
`))
		return err
	})
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// notice: tier1 calling
	// notice: tier2 calling
}

func TestJavaScriptFile(t *testing.T) {
	err := eval.Puppet.Try(func(c eval.Context) error {
		typesFile := `testdata/resource_types.js`
		content, err := ioutil.ReadFile(typesFile)
		if err != nil {
			return err
		}
		ast1 := JavaScriptToAST(c, typesFile, content)
		resourcesFile := `testdata/resource.js`
		content, err = ioutil.ReadFile(resourcesFile)
		if err != nil {
			return err
		}
		ast2 := JavaScriptToAST(c, resourcesFile, content)

		c.AddDefinitions(ast1)
		c.AddDefinitions(ast2)
		c.Evaluator().Evaluate(c, parser.DefaultFactory().Block(
			[]parser.Expression{
				ast1.(*parser.Program).Body(),
				ast2.(*parser.Program).Body(),
			},
			ast1.Locator(), 0, 0))

		return err
	})
	if err != nil {
		t.Error(err)
	}
	// Output: info: hello
}
