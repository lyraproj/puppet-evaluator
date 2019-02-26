package parser_test

import (
	"fmt"
	"github.com/lyraproj/pcore-parser/parser"
)

func ExampleScan() {
	const src = `# This is scanned code.
  constants => {
    first => 0,
    second => 0x32,
    third => 2e4,
    fourth => 2.3e-2,
    fifth => 'hello',
    sixth => "world",
    type => Foo::Bar,
    value => "string\nwith \\escape",
    array => [a, b, c],
    call => Boo::Bar(x, 3)
  }`
	tf := func(t parser.Token) {
		fmt.Println(t)
	}
	parser.Scan(src, tf)
	// Output:
	//Identifier: 'constants'
	//Rocket: '=>'
	//Lc: '{'
	//Identifier: 'first'
	//Rocket: '=>'
	//Integer: '0'
	//Comma: ','
	//Identifier: 'second'
	//Rocket: '=>'
	//Integer: '0x32'
	//Comma: ','
	//Identifier: 'third'
	//Rocket: '=>'
	//Float: '2e4'
	//Comma: ','
	//Identifier: 'fourth'
	//Rocket: '=>'
	//Float: '2.3e-2'
	//Comma: ','
	//Identifier: 'fifth'
	//Rocket: '=>'
	//String: 'hello'
	//Comma: ','
	//Identifier: 'sixth'
	//Rocket: '=>'
	//String: 'world'
	//Comma: ','
	//Identifier: 'type'
	//Rocket: '=>'
	//TypeName: 'Foo::Bar'
	//Comma: ','
	//Identifier: 'value'
	//Rocket: '=>'
	//String: 'string
	//with \escape'
	//Comma: ','
	//Identifier: 'array'
	//Rocket: '=>'
	//Lb: '['
	//Identifier: 'a'
	//Comma: ','
	//Identifier: 'b'
	//Comma: ','
	//Identifier: 'c'
	//Rb: ']'
	//Comma: ','
	//Identifier: 'call'
	//Rocket: '=>'
	//TypeName: 'Boo::Bar'
	//Lp: '('
	//Identifier: 'x'
	//Comma: ','
	//Integer: '3'
	//Rp: ')'
	//Rc: '}'
	//End: ''
}


