package types

import (
	"io"
	"regexp"
	"strings"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/eval"
)

type typedName struct {
	namespace eval.Namespace
	authority eval.URI
	name      string
	canonical string
	parts     []string
}

var TypedNameMetaType eval.Type

func init() {
	TypedNameMetaType = newObjectType(`TypedName`, `{
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
		ns := eval.Namespace(args[0].String())
		n := args[1].String()
		if len(args) > 2 {
			return newTypedName2(ns, n, eval.URI(args[2].(*UriValue).String()))
		}
		return NewTypedName(ns, n)
	}, func(ctx eval.Context, args []eval.Value) eval.Value {
		h := args[0].(*HashValue)
		ns := eval.Namespace(h.Get5(`namespace`, eval.EmptyString).String())
		n := h.Get5(`name`, eval.EmptyString).String()
		if x, ok := h.Get4(`authority`); ok {
			return newTypedName2(ns, n, eval.URI(x.(*UriValue).String()))
		}
		return NewTypedName(ns, n)
	})
}

func (t *typedName) ToString(bld io.Writer, format eval.FormatContext, g eval.RDetect) {
	ObjectToString(t, format, bld, g)
}

func (t *typedName) PType() eval.Type {
	return TypedNameMetaType
}

func (t *typedName) Call(c eval.Context, method eval.ObjFunc, args []eval.Value, block eval.Lambda) (result eval.Value, ok bool) {
	switch method.Name() {
	case `is_parent`:
		return booleanValue(t.IsParent(args[0].(eval.TypedName))), true
	case `relative_to`:
		if r, ok := t.RelativeTo(args[0].(eval.TypedName)); ok {
			return r, true
		}
		return undef, true
	}
	return nil, false
}

func (t *typedName) Get(key string) (value eval.Value, ok bool) {
	switch key {
	case `namespace`:
		return stringValue(string(t.namespace)), true
	case `authority`:
		if t.authority == eval.RuntimeNameAuthority {
			return eval.Undef, true
		}
		return WrapURI2(string(t.authority)), true
	case `name`:
		return stringValue(t.Name()), true
	case `parts`:
		return t.PartsList(), true
	case `is_qualified`:
		return booleanValue(t.IsQualified()), true
	case `parent`:
		p := t.Parent()
		if p == nil {
			return undef, true
		}
		return p, true
	case `child`:
		p := t.Child()
		if p == nil {
			return undef, true
		}
		return p, true
	}
	return nil, false
}

func (t *typedName) InitHash() eval.OrderedMap {
	es := make([]*HashEntry, 0, 3)
	es = append(es, WrapHashEntry2(`namespace`, stringValue(string(t.Namespace()))))
	es = append(es, WrapHashEntry2(`name`, stringValue(t.Name())))
	if t.authority != eval.RuntimeNameAuthority {
		es = append(es, WrapHashEntry2(`authority`, WrapURI2(string(t.authority))))
	}
	return WrapHash(es)
}

func NewTypedName(namespace eval.Namespace, name string) eval.TypedName {
	return newTypedName2(namespace, name, eval.RuntimeNameAuthority)
}

var allowedCharacters = regexp.MustCompile(`\A[A-Za-z][0-9A-Z_a-z]*\z`)

func newTypedName2(namespace eval.Namespace, name string, nameAuthority eval.URI) eval.TypedName {
	tn := typedName{}
	tn.namespace = namespace
	tn.authority = nameAuthority
	tn.name = strings.TrimPrefix(name, `::`)
	return &tn
}

func typedNameFromMapKey(mapKey string) eval.TypedName {
	if i := strings.LastIndexByte(mapKey, '/'); i > 0 {
		pfx := mapKey[:i]
		name := mapKey[i+1:]
		if i = strings.LastIndexByte(pfx, '/'); i > 0 {
			return newTypedName2(eval.Namespace(pfx[i+1:]), name, eval.URI(pfx[:i]))
		}
	}
	panic(eval.Error(eval.InvalidTypedNameMapKey, issue.H{`mapKey`: mapKey}))
}

func (t *typedName) Child() eval.TypedName {
	if !t.IsQualified() {
		return nil
	}
	return t.child(1)
}

func (t *typedName) child(stripCount int) eval.TypedName {
	name := t.name
	sx := 0
	for i := 0; i < stripCount; i++ {
		sx = strings.Index(name, `::`)
		if sx < 0 {
			return nil
		}
		name = name[sx+2:]
	}

	tn := &typedName{
		namespace: t.namespace,
		authority: t.authority,
		name:      name}

	if t.canonical != `` {
		pfxLen := len(t.authority) + len(t.namespace) + 2
		diff := len(t.name) - len(name)
		tn.canonical = t.canonical[:pfxLen] + t.canonical[pfxLen+diff:]
	}
	if t.parts != nil {
		tn.parts = t.parts[stripCount:]
	}
	return tn
}

func (t *typedName) Parent() eval.TypedName {
	lx := strings.LastIndex(t.name, `::`)
	if lx < 0 {
		return nil
	}
	tn := &typedName{
		namespace: t.namespace,
		authority: t.authority,
		name:      t.name[:lx]}

	if t.canonical != `` {
		pfxLen := len(t.authority) + len(t.namespace) + 2
		tn.canonical = t.canonical[:pfxLen+lx]
	}
	if t.parts != nil {
		tn.parts = t.parts[:len(t.parts)-1]
	}
	return tn
}

func (t *typedName) Equals(other interface{}, g eval.Guard) bool {
	if tn, ok := other.(eval.TypedName); ok {
		return t.MapKey() == tn.MapKey()
	}
	return false
}

func (t *typedName) Name() string {
	return t.name
}

func (t *typedName) IsParent(o eval.TypedName) bool {
	tps := t.Parts()
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
	if t.parts == nil {
		return strings.Contains(t.name, `::`)
	}
	return len(t.parts) > 1
}

func (t *typedName) MapKey() string {
	if t.canonical == `` {
		t.canonical = strings.ToLower(string(t.authority) + `/` + string(t.namespace) + `/` + t.name)
	}
	return t.canonical
}

func (t *typedName) Parts() []string {
	if t.parts == nil {
		parts := strings.Split(strings.ToLower(t.name), `::`)
		for _, part := range parts {
			if !allowedCharacters.MatchString(part) {
				panic(eval.Error(eval.InvalidCharactersInName, issue.H{`name`: t.name}))
			}
		}
		t.parts = parts
	}
	return t.parts
}

func (t *typedName) PartsList() eval.List {
	parts := t.Parts()
	es := make([]eval.Value, len(parts))
	for i, p := range parts {
		es[i] = stringValue(p)
	}
	return WrapValues(es)
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
