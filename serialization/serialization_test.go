package serialization

import (
	"bytes"
	"fmt"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-semver/semver"
	"github.com/puppetlabs/go-evaluator/types"

	_ "github.com/puppetlabs/go-evaluator/pcore"
)

func ExampleDataRoundtrip() {
	eval.Puppet.Do(func(ctx eval.Context) error {
		ver, _ := semver.NewVersion(1, 0, 0)
		v := types.WrapSemVer(ver)
		fmt.Printf("%T '%s'\n", v, v)

		dc := NewToDataConverter(ctx, types.SingletonHash2(`rich_data`, types.Boolean_TRUE))
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

func ExampleToDataConverter_Convert() {
	eval.Puppet.Do(func(ctx eval.Context) error {
		ver, _ := semver.NewVersion(1, 0, 0)
		fmt.Println(NewToDataConverter(ctx, types.SingletonHash2(`rich_data`, types.Boolean_TRUE)).Convert(types.WrapSemVer(ver)))
		return nil
	})
	// Output: {'__pcore_type__' => 'SemVer', '__pcore_value__' => '1.0.0'}
}

func ExampleDataToJson() {
	eval.Puppet.Do(func(ctx eval.Context) error {
		buf := bytes.NewBufferString(``)
		DataToJson(ctx, types.WrapHash4(map[string]interface{}{`__pcore_type__`: `SemVer`, `__pcore_value__`: `1.0.0`}), buf, eval.EMPTY_MAP)
		fmt.Println(buf)
		return nil
	})
	// Output: {"__pcore_type__":"SemVer","__pcore_value__":"1.0.0"}
}

func ExampleJsonToData() {
	eval.Puppet.Do(func(ctx eval.Context) error {
		buf := bytes.NewBufferString(`{"__pcore_type__":"SemVer","__pcore_value__":"1.0.0"}`)
		data := JsonToData(ctx, `/tmp/ver.json`, buf)
		fmt.Println(data)
		return nil
	})
	// Output: {'__pcore_type__' => 'SemVer', '__pcore_value__' => '1.0.0'}
}

func ExampleFromDataConverter_Convert() {
	data := types.WrapHash4(map[string]interface{}{`__pcore_type__`: `SemVer`, `__pcore_value__`: `1.0.0`})
	eval.Puppet.Do(func(ctx eval.Context) error {
		ver := NewFromDataConverter(ctx, eval.EMPTY_MAP).Convert(data)
		fmt.Printf("%T\n", ver)
		fmt.Println(ver)
		return nil
	})
	// Output:
	// *types.SemVerValue
	// 1.0.0
}
