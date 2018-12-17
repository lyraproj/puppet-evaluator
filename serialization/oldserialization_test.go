package serialization

import (
	"bytes"
	"fmt"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/impl"
	"github.com/lyraproj/puppet-evaluator/types"
	"github.com/lyraproj/semver/semver"

	_ "github.com/lyraproj/puppet-evaluator/pcore"
	"reflect"
)

func ExampleFromDataConverter_roundtrip() {
	eval.Puppet.Do(func(ctx eval.Context) {
		ver, _ := semver.NewVersion(1, 0, 0)
		v := types.WrapSemVer(ver)
		fmt.Printf("%T '%s'\n", v, v)

		dc := NewToDataConverter(types.SingletonHash2(`rich_data`, types.Boolean_TRUE))
		data := dc.Convert(v)

		buf := bytes.NewBufferString(``)
		DataToJson(data, buf)

		fc := NewFromDataConverter(ctx, eval.EMPTY_MAP)
		cl := NewCollector()
		JsonToData(`/tmp/sample.json`, buf, cl)
		v2 := fc.Convert(cl.Value())

		fmt.Printf("%T '%s'\n", v2, v2)
	})
	// Output:
	// *types.SemVerValue '1.0.0'
	// *types.SemVerValue '1.0.0'
}

func ExampleFromDataConverter_ObjectRoundtrip() {
	eval.Puppet.Do(func(ctx eval.Context) {
		p := impl.NewParameter(`p1`, ctx.ParseType2(`Type[String]`), nil, false)
		fmt.Println(p)
		data := NewToDataConverter(eval.EMPTY_MAP).Convert(types.WrapValues([]eval.Value{p, p}))

		buf := bytes.NewBufferString(``)
		DataToJson(data, buf)

		fc := NewFromDataConverter(ctx, eval.EMPTY_MAP)
		b := buf.String()
		fmt.Print(b)
		cl := NewCollector()
		JsonToData(`/tmp/sample.json`, buf, cl)
		p2 := fc.Convert(cl.Value()).(eval.List).At(0)

		fmt.Println(p2)
	})
	// Output:
	// Parameter('name' => 'p1', 'type' => Type[String])
	// [{"__ptype":"Parameter","name":"p1","type":{"__ptype":"Type","__pvalue":"Type[String]"}},{"__pref":1}]
	// Parameter('name' => 'p1', 'type' => Type[String])
}

func ExampleFromDataConverter_StructInArrayRoundtrip() {
	eval.Puppet.Do(func(ctx eval.Context) {
		p := types.WrapValues([]eval.Value{ctx.ParseType2(`Struct[a => String, b => Integer]`)})
		fmt.Println(p)
		data := NewToDataConverter(eval.EMPTY_MAP).Convert(p)

		buf := bytes.NewBufferString(``)
		DataToJson(data, buf)

		fc := NewFromDataConverter(ctx, eval.EMPTY_MAP)
		b := buf.String()
		fmt.Print(b)
		cl := NewCollector()
		JsonToData(`/tmp/sample.json`, buf, cl)
		p2 := fc.Convert(cl.Value())

		fmt.Println(p2)
	})
	// Output:
	// [Struct[{'a' => String, 'b' => Integer}]]
	// [{"__ptype":"Type","__pvalue":"Struct[{'a' =\u003e String, 'b' =\u003e Integer}]"}]
	// [Struct[{'a' => String, 'b' => Integer}]]
}

func ExampleFromDataConverter_TypeSetRoundtrip() {
	eval.Puppet.Do(func(ctx eval.Context) {
		p := ctx.ParseType2(`TypeSet[{
      name => 'Foo',
      version => '1.0.0',
      pcore_version => '1.0.0',
      types => {
        Bar => Object[
  attributes => {
    subnet_id => { type => Optional[String], value => 'FAKED_SUBNET_ID' },
    vpc_id => String,
    cidr_block => String,
    map_public_ip_on_launch => Boolean
  }
        ]
      }}]`)
		ctx.AddTypes(p)
		fmt.Println(p)
		data := NewToDataConverter(eval.EMPTY_MAP).Convert(p)

		buf := bytes.NewBufferString(``)
		DataToJson(data, buf)

		fc := NewFromDataConverter(ctx, eval.EMPTY_MAP)
		b := buf.String()
		fmt.Print(b)
		cl := NewCollector()
		JsonToData(`/tmp/sample.json`, buf, cl)
		p2 := fc.Convert(cl.Value())
		fmt.Println(p2)
	})
	// Output:
	// TypeSet[{pcore_version => '1.0.0', name_authority => 'http://puppet.com/2016.1/runtime', name => 'Foo', version => '1.0.0', types => {Bar => {attributes => {'subnet_id' => {'type' => Optional[String], 'value' => 'FAKED_SUBNET_ID'}, 'vpc_id' => String, 'cidr_block' => String, 'map_public_ip_on_launch' => Boolean}}}}]
	// {"__ptype":"Pcore::TypeSet","pcore_version":{"__ptype":"SemVer","__pvalue":"1.0.0"},"name_authority":{"__ptype":"URI","__pvalue":"http://puppet.com/2016.1/runtime"},"name":"Foo","version":{"__ptype":"SemVer","__pvalue":"1.0.0"},"types":{"Bar":{"__ptype":"Pcore::ObjectType","name":"Foo::Bar","attributes":{"subnet_id":{"type":{"__ptype":"Type","__pvalue":"Optional[String]"},"value":"FAKED_SUBNET_ID"},"vpc_id":{"__ptype":"Type","__pvalue":"String"},"cidr_block":{"__ptype":"Type","__pvalue":"String"},"map_public_ip_on_launch":{"__ptype":"Type","__pvalue":"Boolean"}}}}}
	// TypeSet[{pcore_version => '1.0.0', name_authority => 'http://puppet.com/2016.1/runtime', name => 'Foo', version => '1.0.0', types => {Bar => {attributes => {'subnet_id' => {'type' => Optional[String], 'value' => 'FAKED_SUBNET_ID'}, 'vpc_id' => String, 'cidr_block' => String, 'map_public_ip_on_launch' => Boolean}}}}]
}

func ExampleFromDataConverter_goValueRoundtrip() {
	type MyInt int

	eval.Puppet.Do(func(ctx eval.Context) {
		mi := MyInt(32)
		ctx.AddTypes(ctx.Reflector().ObjectTypeFromReflect(`Test::MyInt`, nil, reflect.TypeOf(mi)))

		v := eval.Wrap(ctx, mi)
		fmt.Println(v)

		dc := NewToDataConverter(types.SingletonHash2(`rich_data`, types.Boolean_TRUE))
		data := dc.Convert(v)

		buf := bytes.NewBufferString(``)
		DataToJson(data, buf)

		fc := NewFromDataConverter(ctx, eval.EMPTY_MAP)
		cl := NewCollector()
		JsonToData(`/tmp/sample.json`, buf, cl)
		v2 := fc.Convert(cl.Value())

		fmt.Println(v2)
	})
	// Output:
	// Test::MyInt('value' => 32)
	// Test::MyInt('value' => 32)
}

func ExampleFromDataConverter_goStructRoundtrip() {
	type MyStruct struct {
		X int
		Y string
	}

	eval.Puppet.Do(func(ctx eval.Context) {
		mi := &MyStruct{32, "hello"}
		ctx.AddTypes(ctx.Reflector().ObjectTypeFromReflect(`Test::MyStruct`, nil, reflect.TypeOf(mi)))

		v := eval.Wrap(ctx, mi)
		fmt.Println(v)

		dc := NewToDataConverter(types.SingletonHash2(`rich_data`, types.Boolean_TRUE))
		data := dc.Convert(v)

		buf := bytes.NewBufferString(``)
		DataToJson(data, buf)

		fc := NewFromDataConverter(ctx, eval.EMPTY_MAP)
		cl := NewCollector()
		JsonToData(`/tmp/sample.json`, buf, cl)
		v2 := fc.Convert(cl.Value())

		fmt.Println(v2)
		ms2 := v2.(eval.Reflected).Reflect(ctx).Interface()
		fmt.Printf("%T %v\n", ms2, ms2)
	})
	// Output:
	// Test::MyStruct('x' => 32, 'y' => 'hello')
	// Test::MyStruct('x' => 32, 'y' => 'hello')
	// serialization.MyStruct {32 hello}
}

func ExampleFromDataConverter_goStructWithDynamicRoundtrip() {
	type MyStruct struct {
		X eval.List
		Y eval.OrderedMap
	}

	eval.Puppet.Do(func(ctx eval.Context) {
		mi := &MyStruct{eval.Wrap(ctx, []int{32}).(eval.List), eval.Wrap(ctx, map[string]string{"msg": "hello"}).(eval.OrderedMap)}
		ctx.AddTypes(ctx.Reflector().ObjectTypeFromReflect(`Test::MyStruct`, nil, reflect.TypeOf(mi)))

		v := eval.Wrap(ctx, mi)
		fmt.Println(v)

		dc := NewToDataConverter(types.SingletonHash2(`rich_data`, types.Boolean_TRUE))
		data := dc.Convert(v)

		buf := bytes.NewBufferString(``)
		DataToJson(data, buf)

		fc := NewFromDataConverter(ctx, eval.EMPTY_MAP)
		cl := NewCollector()
		JsonToData(`/tmp/sample.json`, buf, cl)
		v2 := fc.Convert(cl.Value())

		fmt.Println(v2)
		ms2 := v2.(eval.Reflected).Reflect(ctx).Interface()
		fmt.Printf("%T %v\n", ms2, ms2)
	})
	// Output:
	// Test::MyStruct('x' => [32], 'y' => {'msg' => 'hello'})
	// Test::MyStruct('x' => [32], 'y' => {'msg' => 'hello'})
	// serialization.MyStruct {[32] {'msg' => 'hello'}}
}

func ExampleToDataConverter_Convert() {
	eval.Puppet.Do(func(ctx eval.Context) {
		ver, _ := semver.NewVersion(1, 0, 0)
		fmt.Println(NewToDataConverter(types.SingletonHash2(`rich_data`, types.Boolean_TRUE)).Convert(types.WrapSemVer(ver)))
	})
	// Output: {'__ptype' => 'SemVer', '__pvalue' => '1.0.0'}
}

func ExampleToDataConverter_Convert2() {
	eval.Puppet.Do(func(ctx eval.Context) {
		param := impl.NewParameter(`p`, types.DefaultStringType(), types.WrapString(`v`), false)
		fmt.Println(NewToDataConverter(types.SingletonHash2(`rich_data`, types.Boolean_TRUE)).Convert(param))
	})
	// Output: {'__ptype' => 'Parameter', 'name' => 'p', 'type' => {'__ptype' => 'Type', '__pvalue' => 'String'}, 'value' => 'v'}
}

func ExampleDataToJson() {
	eval.Puppet.Do(func(ctx eval.Context) {
		buf := bytes.NewBufferString(``)
		DataToJson(types.WrapStringToInterfaceMap(ctx, map[string]interface{}{`__ptype`: `SemVer`, `__pvalue`: `1.0.0`}), buf)
		fmt.Println(buf)
	})
	// Output: {"__ptype":"SemVer","__pvalue":"1.0.0"}
}

func ExampleFromDataConverter_Convert_semver() {
	eval.Puppet.Do(func(ctx eval.Context) {
		data := types.WrapStringToInterfaceMap(ctx, map[string]interface{}{`__ptype`: `SemVer`, `__pvalue`: `1.0.0`})
		ver := NewFromDataConverter(ctx, eval.EMPTY_MAP).Convert(data)
		fmt.Printf("%T\n", ver)
		fmt.Println(ver)
	})
	// Output:
	// *types.SemVerValue
	// 1.0.0
}

func ExampleFromDataConverter_Convert_hash() {
	eval.Puppet.Do(func(ctx eval.Context) {
		data := types.WrapStringToInterfaceMap(ctx, map[string]interface{}{`__ptype`: `Parameter`, `name`: `p`, `type`: map[string]interface{}{`__ptype`: `Pcore::StringType`, `size_type_or_value`: map[string]interface{}{`__ptype`: `Pcore::IntegerType`, `from`: 0}}, `value`: `v`})
		ver := NewFromDataConverter(ctx, eval.EMPTY_MAP).Convert(data)
		fmt.Printf("%T\n", ver)
		fmt.Println(ver)
	})
	// Output:
	// *impl.parameter
	// Parameter('name' => 'p', 'type' => String, 'value' => 'v')
}
