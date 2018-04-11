package serialization

import (
	"bytes"
	"fmt"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/semver"
	"github.com/puppetlabs/go-evaluator/types"
	"testing"

	_ "github.com/puppetlabs/go-evaluator/pcore"
)

func TestRichDataRoundtrip(t *testing.T) {
	ver, _ := semver.NewVersion(1, 0, 0)
	v := types.WrapSemVer(ver)
	buf := bytes.NewBufferString(``)
	v2, _ := eval.Puppet.Produce(func(ctx eval.Context) (eval.PValue, error) {
		DataToJson(ctx, NewToDataConverter(ctx, types.SingletonHash2(`rich_data`, types.Boolean_TRUE)).Convert(v), buf, eval.EMPTY_MAP)
		return NewFromDataConverter(ctx, eval.EMPTY_MAP).Convert(JsonToData(ctx, ``, buf)), nil
	})
	if !eval.Equals(v, v2) {
		t.Errorf(`Expected %T '%s', got %T '%s'`, v, v, v2, v2)
	}
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
	ver, _ := eval.Puppet.Produce(func(ctx eval.Context) (eval.PValue, error) {
		return NewFromDataConverter(ctx, eval.EMPTY_MAP).Convert(data), nil
	})
	fmt.Printf("%T\n", ver)
	fmt.Println(ver)
	// Output:
	// *types.SemVerValue
	// 1.0.0
}
