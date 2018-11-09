package serialization

import (
	"bytes"
	"fmt"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/impl"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-semver/semver"

	_ "github.com/puppetlabs/go-evaluator/pcore"
	"reflect"
)

func ExampleDataRoundtrip() {
	eval.Puppet.Do(func(ctx eval.Context) error {
		ver, _ := semver.NewVersion(1, 0, 0)
		v := types.WrapSemVer(ver)
		fmt.Printf("%T '%s'\n", v, v)

		dc := NewToDataConverter(types.SingletonHash2(`rich_data`, types.Boolean_TRUE))
		data := dc.Convert(v)

		buf := bytes.NewBufferString(``)
		DataToJson(ctx, data, buf, eval.EMPTY_MAP)

		fc := NewFromDataConverter(ctx, eval.EMPTY_MAP)
		data2 := JsonToData(ctx, `/tmp/sample.json`, buf)
		v2 := fc.Convert(data2)

		fmt.Printf("%T '%s'\n", v2, v2)
		return nil
	})
	// Output:
	// *types.SemVerValue '1.0.0'
	// *types.SemVerValue '1.0.0'
}

func ExampleGoValueRoundtrip() {
	type MyInt int

	eval.Puppet.Do(func(ctx eval.Context) error {
		mi := MyInt(32)
		ctx.AddTypes(ctx.Reflector().ObjectTypeFromReflect(`Test::MyInt`, nil, reflect.TypeOf(mi)))

		v := eval.Wrap(ctx, mi)
		fmt.Println(v)

		dc := NewToDataConverter(types.SingletonHash2(`rich_data`, types.Boolean_TRUE))
		data := dc.Convert(v)

		buf := bytes.NewBufferString(``)
		DataToJson(ctx, data, buf, eval.EMPTY_MAP)

		fc := NewFromDataConverter(ctx, eval.EMPTY_MAP)
		data2 := JsonToData(ctx, `/tmp/sample.json`, buf)
		v2 := fc.Convert(data2)

		fmt.Println(v2)
		return nil
	})
	// Output:
	// Test::MyInt('value' => 32)
	// Test::MyInt('value' => 32)
}

func ExampleToDataConverter_Convert() {
	eval.Puppet.Do(func(ctx eval.Context) error {
		ver, _ := semver.NewVersion(1, 0, 0)
		fmt.Println(NewToDataConverter(types.SingletonHash2(`rich_data`, types.Boolean_TRUE)).Convert(types.WrapSemVer(ver)))
		return nil
	})
	// Output: {'__ptype' => 'SemVer', '__pvalue' => '1.0.0'}
}

func ExampleToDataConverter_Convert2() {
	eval.Puppet.Do(func(ctx eval.Context) error {
		param := impl.NewParameter(`p`, types.DefaultStringType(), types.WrapString(`v`), false)
		fmt.Println(NewToDataConverter(types.SingletonHash2(`rich_data`, types.Boolean_TRUE)).Convert(param))
		return nil
	})
	// Output: {'__ptype' => 'Parameter', 'name' => 'p', 'type' => {'__ptype' => 'Pcore::StringType', 'size_type_or_value' => {'__ptype' => 'Pcore::IntegerType', 'from' => 0}}, 'value' => 'v'}
}

func ExampleDataToJson() {
	eval.Puppet.Do(func(ctx eval.Context) error {
		buf := bytes.NewBufferString(``)
		DataToJson(ctx, types.WrapStringToInterfaceMap(ctx, map[string]interface{}{`__ptype`: `SemVer`, `__pvalue`: `1.0.0`}), buf, eval.EMPTY_MAP)
		fmt.Println(buf)
		return nil
	})
	// Output: {"__ptype":"SemVer","__pvalue":"1.0.0"}
}

func ExampleJsonToData() {
	eval.Puppet.Do(func(ctx eval.Context) error {
		buf := bytes.NewBufferString(`{"__ptype":"SemVer","__pvalue":"1.0.0"}`)
		data := JsonToData(ctx, `/tmp/ver.json`, buf)
		fmt.Println(data)
		return nil
	})
	// Output: {'__ptype' => 'SemVer', '__pvalue' => '1.0.0'}
}

func ExampleFromDataConverter_Convert() {
	eval.Puppet.Do(func(ctx eval.Context) error {
		data := types.WrapStringToInterfaceMap(ctx, map[string]interface{}{`__ptype`: `SemVer`, `__pvalue`: `1.0.0`})
		ver := NewFromDataConverter(ctx, eval.EMPTY_MAP).Convert(data)
		fmt.Printf("%T\n", ver)
		fmt.Println(ver)
		return nil
	})
	// Output:
	// *types.SemVerValue
	// 1.0.0
}

func ExampleFromDataConverter_Convert2() {
	eval.Puppet.Do(func(ctx eval.Context) error {
		data := types.WrapStringToInterfaceMap(ctx, map[string]interface{}{`__ptype`: `Parameter`, `name`: `p`, `type`: map[string]interface{}{`__ptype`: `Pcore::StringType`, `size_type_or_value`: map[string]interface{}{`__ptype`: `Pcore::IntegerType`, `from`: 0}}, `value`: `v`})
		ver := NewFromDataConverter(ctx, eval.EMPTY_MAP).Convert(data)
		fmt.Printf("%T\n", ver)
		fmt.Println(ver)
		return nil
	})
	// Output:
	// *impl.parameter
	// Parameter('name' => 'p', 'type' => String, 'value' => 'v')
}
