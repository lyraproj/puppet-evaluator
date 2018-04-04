package serialization

import (
	"testing"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-evaluator/semver"
	"bytes"
	"github.com/puppetlabs/go-evaluator/pcore"
	"fmt"
)

func TestRichDataRoundtrip(t *testing.T) {
	ver, _ := semver.NewVersion(1, 0, 0)
	v := types.WrapSemVer(ver)
	buf := bytes.NewBufferString(``)
	pcore.InitializePuppet()
	DataToJson(NewToDataConverter(types.SingletonHash2(`rich_data`, types.Boolean_TRUE)).Convert(v), buf, eval.EMPTY_MAP)
	v2 := NewFromDataConverter(eval.NewParentedLoader(eval.Puppet.SystemLoader()), eval.EMPTY_MAP).Convert(JsonToData(``, buf))
	if !eval.Equals(v, v2) {
		t.Errorf(`Expected %T '%s', got %T '%s'`, v, v, v2, v2)
	}
}

func ExampleToDataConverter_Convert() {
	ver, _ := semver.NewVersion(1, 0, 0)
	fmt.Println(NewToDataConverter(types.SingletonHash2(`rich_data`, types.Boolean_TRUE)).Convert(types.WrapSemVer(ver)))
	// Output: {'__pcore_type__' => 'SemVer', '__pcore_value__' => '1.0.0'}
}

func ExampleDataToJson() {
	buf := bytes.NewBufferString(``)
	DataToJson(types.WrapHash4(map[string]interface{}{`__pcore_type__`: `SemVer`, `__pcore_value__`: `1.0.0`}), buf, eval.EMPTY_MAP)
	fmt.Println(buf)
	// Output: {"__pcore_type__":"SemVer","__pcore_value__":"1.0.0"}
}

func ExampleJsonToData() {
	buf := bytes.NewBufferString(`{"__pcore_type__":"SemVer","__pcore_value__":"1.0.0"}`)
	data := JsonToData(`/tmp/ver.json`, buf)
	fmt.Println(data)
	// Output: {'__pcore_type__' => 'SemVer', '__pcore_value__' => '1.0.0'}
}

func ExampleFromDataConverter_Convert() {
	pcore.InitializePuppet()
	data := types.WrapHash4(map[string]interface{}{`__pcore_type__`: `SemVer`, `__pcore_value__`: `1.0.0`})
	ver := NewFromDataConverter(eval.NewParentedLoader(eval.Puppet.SystemLoader()), eval.EMPTY_MAP).Convert(data)
	fmt.Printf("%T\n", ver)
	fmt.Println(ver)
	// Output:
	// *types.SemVerValue
	// 1.0.0
}
