package types

import (
	"fmt"
	"github.com/lyraproj/puppet-evaluator/utils"
)

func Example_scan() {
	const src = `# This is scanned code.
  constants => {
    first => 0,
    second => 0x32,
    third => 2e4,
    fourth => 2.3e-2,
    fifth => 'hello',
    sixth => "world",
    type => Foo::Bar,
    value => "String\nWith \\Escape",
    array => [a, b, c],
    call => Boo::Bar(x, 3)
  }`
	tf := func(t token) error {
		fmt.Println(t)
		return nil
	}
	err := scan(utils.NewStringReader(src), tf)
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	//identifier: 'constants'
	//rocket: '=>'
	//leftCurlyBrace: '{'
	//identifier: 'first'
	//rocket: '=>'
	//integer: '0'
	//comma: ','
	//identifier: 'second'
	//rocket: '=>'
	//integer: '0x32'
	//comma: ','
	//identifier: 'third'
	//rocket: '=>'
	//float: '2e4'
	//comma: ','
	//identifier: 'fourth'
	//rocket: '=>'
	//float: '2.3e-2'
	//comma: ','
	//identifier: 'fifth'
	//rocket: '=>'
	//string: 'hello'
	//comma: ','
	//identifier: 'sixth'
	//rocket: '=>'
	//string: 'world'
	//comma: ','
	//identifier: 'type'
	//rocket: '=>'
	//name: 'Foo::Bar'
	//comma: ','
	//identifier: 'value'
	//rocket: '=>'
	//string: 'String
	//With \Escape'
	//comma: ','
	//identifier: 'array'
	//rocket: '=>'
	//leftBracket: '['
	//identifier: 'a'
	//comma: ','
	//identifier: 'b'
	//comma: ','
	//identifier: 'c'
	//rightBracket: ']'
	//comma: ','
	//identifier: 'call'
	//rocket: '=>'
	//name: 'Boo::Bar'
	//leftParen: '('
	//identifier: 'x'
	//comma: ','
	//integer: '3'
	//rightParen: ')'
	//rightCurlyBrace: '}'
	//end: ''
}
