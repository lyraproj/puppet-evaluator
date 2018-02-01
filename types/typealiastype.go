package types

import (
	. "fmt"
	. "io"

	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/eval"
	. "github.com/puppetlabs/go-parser/parser"
)

type TypeAliasType struct {
	name           string
	typeExpression Expression
	resolvedType   PType
	loader         Loader
}

func DefaultTypeAliasType() *TypeAliasType {
	return typeAliasType_DEFAULT
}

func NewTypeAliasType(name string, typeExpression Expression, resolvedType PType) *TypeAliasType {
	return &TypeAliasType{name, typeExpression, resolvedType, nil}
}

func NewTypeAliasType2(args ...PValue) *TypeAliasType {
	switch len(args) {
	case 0:
		return DefaultTypeAliasType()
	case 2:
		name, ok := args[0].(*StringValue)
		if !ok {
			panic(NewIllegalArgumentType2(`TypeAlias`, 0, `String`, args[0]))
		}
		var pt PType
		if pt, ok = args[1].(PType); ok {
			return NewTypeAliasType(name.String(), nil, pt)
		}
		var rt *RuntimeValue
		if rt, ok = args[1].(*RuntimeValue); ok {
			var ex Expression
			if ex, ok = rt.Interface().(Expression); ok {
				return NewTypeAliasType(name.String(), ex, nil)
			}
		}
		panic(NewIllegalArgumentType2(`TypeAlias[]`, 1, `Type or Expression`, args[1]))
	default:
		panic(NewIllegalArgumentCount(`TypeAlias[]`, `0 or 2`, len(args)))
	}
}

func (t *TypeAliasType) Accept(v Visitor, g Guard) {
	if g == nil {
		g = make(Guard)
	}
	if g.Seen(t, nil) {
		return
	}
	v(t)
	t.resolvedType.Accept(v, g)
}

func (t *TypeAliasType) Default() PType {
	return typeAliasType_DEFAULT
}

func (t *TypeAliasType) Equals(o interface{}, g Guard) bool {
	if ot, ok := o.(*TypeAliasType); ok && t.name == ot.name {
		if g == nil {
			g = make(Guard)
		}
		if g.Seen(t, ot) {
			return true
		}
		tr := t.resolvedType
		otr := ot.resolvedType
		return tr.Equals(otr, g)
	}
	return false
}

func (t *TypeAliasType) Loader() Loader {
	return t.loader
}

func (t *TypeAliasType) IsAssignable(o PType, g Guard) bool {
	if g == nil {
		g = make(Guard)
	}
	if g.Seen(t, o) {
		return true
	}
	return GuardedIsAssignable(t.ResolvedType(), o, g)
}

func (t *TypeAliasType) IsInstance(o PValue, g Guard) bool {
	if g == nil {
		g = make(Guard)
	}
	if g.Seen(t, o) {
		return true
	}
	return GuardedIsInstance(t.ResolvedType(), o, g)
}

func (t *TypeAliasType) Name() string {
	return t.name
}

func (t *TypeAliasType) Resolve(c EvalContext) PType {
	if t.resolvedType != nil {
		panic(Sprintf(`Attempt to resolve already resolved type %s`, t.name))
	}
	t.resolvedType = c.ResolveType(t.typeExpression)
	t.loader = c.Loader()
	return t
}

func (t *TypeAliasType) ResolvedType() PType {
	if t.resolvedType == nil {
		panic(Sprintf("Reference to unresolved type '%s'", t.name))
	}
	return t.resolvedType
}

func (t *TypeAliasType) String() string {
	return ToString2(t, EXPANDED)
}

func (t *TypeAliasType) ToString(b Writer, s FormatContext, g RDetect) {
	f := GetFormat(s.FormatMap(), t.Type())
	if t.name == `UnresolvedAlias` {
		WriteString(b, `TypeAlias`)
	} else {
		WriteString(b, t.name)
		if !(f.IsAlt() && f.FormatChar() == 'b') {
			return
		}
		if g == nil {
			g = make(RDetect)
		} else if g[t] {
			return
		}
		g[t] = true
		WriteString(b, ` = `)

		// TODO: Need to be adjusted when included in TypeSet
		t.resolvedType.ToString(b, s, g)
	}
}

func (t *TypeAliasType) Type() PType {
	return &TypeType{t}
}

var typeAliasType_DEFAULT = &TypeAliasType{`UnresolvedAlias`, nil, defaultType_DEFAULT, nil}
