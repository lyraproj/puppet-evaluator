package types

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"io"
	"strings"
)

type typedName struct {
	namespace eval.Namespace
	authority eval.URI
	compound  string
	canonical string
	parts     []string
}

var TypedName_Type eval.Type

func init() {
	TypedName_Type = newObjectType(`TypedName`, `{
    attributes => {
      'namespace' => String,
      'name' => String,
      'authority' => { type => Optional[URI], value => undef },
      'parts' => { type => Array[String], kind => derived },
      'is_qualified' => { type => Boolean, kind => derived },
      'child' => { type => Optional[TypedName], kind => derived },
      'parent' => { type => Optional[TypedName], kind => derived }
    },
    functions => {
      'is_parent' => Callable[[TypedName],Boolean],
      'relative_to' => Callable[[TypedName],Optional[TypedName]]
    }
  }`, func(ctx eval.Context, args []eval.Value) eval.Value {
		ns := eval.Namespace(args[0].(*StringValue).String())
		n := args[1].(*StringValue).String()
		if len(args) > 2 {
			return newTypedName2(ns, n, eval.URI(args[2].(*UriValue).String()))
		}
		return newTypedName(ns, n)
	}, func(ctx eval.Context, args []eval.Value) eval.Value {
		h := args[0].(*HashValue)
		ns := eval.Namespace(h.Get5(`namespace`, eval.EMPTY_STRING).(*StringValue).String())
		n := h.Get5(`name`, eval.EMPTY_STRING).(*StringValue).String()
		if x, ok := h.Get4(`authority`); ok {
			return newTypedName2(ns, n, eval.URI(x.(*UriValue).String()))
		}
		return newTypedName(ns, n)
	})
}

func (t *typedName) ToString(bld io.Writer, format eval.FormatContext, g eval.RDetect) {
	ObjectToString(t, format, bld, g)
}

func (t *typedName) PType() eval.Type {
	return TypedName_Type
}

func (t *typedName) Call(c eval.Context, method string, args []eval.Value, block eval.Lambda) (result eval.Value, ok bool) {
	switch method {
	case `is_parent`:
		return WrapBoolean(t.IsParent(args[0].(eval.TypedName))), true
	case `relative_to`:
		if r, ok := t.RelativeTo(args[0].(eval.TypedName)); ok {
			return r, true
		}
		return _UNDEF, true
	}
	return nil, false
}

func (t *typedName) Get(key string) (value eval.Value, ok bool) {
	switch key {
	case `namespace`:
		return WrapString(string(t.namespace)), true
	case `authority`:
		if t.authority == eval.RUNTIME_NAME_AUTHORITY {
			return eval.UNDEF, true
		}
		return WrapURI2(string(t.authority)), true
	case `name`:
		return WrapString(t.Name()), true
	case `parts`:
		return t.PartsList(), true
	case `is_qualified`:
		return WrapBoolean(t.IsQualified()), true
	case `parent`:
		p := t.Parent()
		if p == nil {
			return _UNDEF, true
		}
		return p, true
	case `child`:
		p := t.Child()
		if p == nil {
			return _UNDEF, true
		}
		return p, true
	}
	return nil, false
}

func (t *typedName) InitHash() eval.OrderedMap {
	es := make([]*HashEntry, 0, 3)
	es = append(es, WrapHashEntry2(`namespace`, WrapString(string(t.Namespace()))))
	es = append(es, WrapHashEntry2(`name`, WrapString(t.Name())))
	if t.authority != eval.RUNTIME_NAME_AUTHORITY {
		es = append(es, WrapHashEntry2(`authority`, WrapURI2(string(t.authority))))
	}
	return WrapHash(es)
}

func newTypedName(namespace eval.Namespace, name string) eval.TypedName {
	return newTypedName2(namespace, name, eval.RUNTIME_NAME_AUTHORITY)
}

func newTypedName2(namespace eval.Namespace, name string, nameAuthority eval.URI) eval.TypedName {
	tn := typedName{}

	parts := strings.Split(strings.ToLower(name), `::`)
	if len(parts) > 0 && parts[0] == `` && len(name) > 2 {
		parts = parts[1:]
		name = name[2:]
	}
	tn.parts = parts
	tn.namespace = namespace
	tn.authority = nameAuthority
	tn.compound = string(nameAuthority) + `/` + string(namespace) + `/` + name
	tn.canonical = strings.ToLower(tn.compound)
	return &tn
}

func (t *typedName) Child() eval.TypedName {
	if !t.IsQualified() {
		return nil
	}
	return t.child(1)
}

func (t *typedName) child(stripCount int) eval.TypedName {
	if !t.IsQualified() {
		return nil
	}

	compound := t.compound
	sx := 0
	for i := 0; i < stripCount; i++ {
		sx = strings.Index(compound, `::`)
		if sx < 0 {
			return nil
		}
		compound = compound[sx + 2:]
	}
	return &typedName{
		parts:     t.parts[stripCount:],
		namespace: t.namespace,
		authority: t.authority,
		compound:  compound,
		canonical: t.canonical[len(t.canonical) - len(compound):]}
}

func (t *typedName) Parent() eval.TypedName {
	if !t.IsQualified() {
		return nil
	}
	lx := strings.LastIndex(t.compound, `::`)
	return &typedName{
		parts:     t.parts[:len(t.parts)-1],
		namespace: t.namespace,
		authority: t.authority,
		compound:  t.compound[:lx],
		canonical: t.canonical[:lx]}
}

func (t *typedName) Equals(other interface{}, g eval.Guard) bool {
	if tn, ok := other.(eval.TypedName); ok {
		return t.canonical == tn.MapKey()
	}
	return false
}

func (t *typedName) Name() string {
	cn := t.compound
	return cn[strings.LastIndex(cn, `/`)+1:]
}

func (t *typedName) IsParent(o eval.TypedName) bool {
	tps := t.parts
	ops := o.Parts()
	top := len(tps)
	if top < len(ops) {
		for idx := 0; idx < top; idx++ {
			if tps[idx] != ops[idx] {
				return false
			}
		}
		return true
	}
	return false
}

func (t *typedName) RelativeTo(parent eval.TypedName) (eval.TypedName, bool) {
	if parent.IsParent(t) {
		return t.child(len(parent.Parts())), true
	}
	return nil, false
}

func (t *typedName) IsQualified() bool {
	return len(t.parts) > 1
}

func (t *typedName) MapKey() string {
	return t.canonical
}

func (t *typedName) Parts() []string {
	return t.parts
}

func (t *typedName) PartsList() eval.List {
	elems := make([]eval.Value, len(t.parts))
	for i, p := range t.parts {
		elems[i] = WrapString(p)
	}
	return WrapArray(elems)
}

func (t *typedName) String() string {
	return eval.ToString(t)
}

func (t *typedName) Namespace() eval.Namespace {
	return t.namespace
}

func (t *typedName) Authority() eval.URI {
	return t.authority
}