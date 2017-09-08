package values

import (
	. "fmt"
	. "io"

	. "github.com/puppetlabs/go-evaluator/eval/errors"
	. "github.com/puppetlabs/go-evaluator/eval/evaluator"
	. "github.com/puppetlabs/go-evaluator/eval/values/api"
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

func (t *TypeAliasType) Resolve(resolver TypeResolver) {
	if t.resolvedType != nil {
		panic(Sprintf(`Attempt to resolve already resolved type %s`, t.name))
	}
	t.resolvedType = resolver.Resolve(t.typeExpression)
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

func (t *TypeAliasType) ToString(bld Writer, format FormatContext, g RDetect) {
	if t.name == `UnresolvedAlias` {
		WriteString(bld, `TypeAlias`)
	} else {
		WriteString(bld, t.name)
		if format == NONE {
			return
		}
		if g == nil {
			g = make(RDetect)
		} else if g[t] {
			return
		}
		g[t] = true
		WriteString(bld, ` = `)

		// TODO: Need to be adjusted when included in TypeSet
		t.resolvedType.ToString(bld, format, g)
	}
}

func (t *TypeAliasType) Type() PType {
	return &TypeType{t}
}

var typeAliasType_DEFAULT = &TypeAliasType{`UnresolvedAlias`, nil, defaultType_DEFAULT, nil}
