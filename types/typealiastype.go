package types

import (
	"fmt"
	"io"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-parser/parser"
)

type TypeAliasType struct {
	name           string
	typeExpression parser.Expression
	resolvedType   eval.PType
	loader         eval.Loader
}

var TypeAlias_Type eval.ObjectType

func init() {
	TypeAlias_Type = newObjectType(`Pcore::TypeAlias`,
		`Pcore::AnyType {
	attributes => {
		name => String[1],
		resolved_type => {
			type => Optional[Type],
			value => undef
		}
	}
}`, func(ctx eval.Context, args []eval.PValue) eval.PValue {
			return NewTypeAliasType2(args...)
		})
}

func DefaultTypeAliasType() *TypeAliasType {
	return typeAliasType_DEFAULT
}

func NewTypeAliasType(name string, typeExpression parser.Expression, resolvedType eval.PType) *TypeAliasType {
	return &TypeAliasType{name, typeExpression, resolvedType, nil}
}

func NewTypeAliasType2(args ...eval.PValue) *TypeAliasType {
	switch len(args) {
	case 0:
		return DefaultTypeAliasType()
	case 2:
		name, ok := args[0].(*StringValue)
		if !ok {
			panic(NewIllegalArgumentType2(`TypeAlias`, 0, `String`, args[0]))
		}
		var pt eval.PType
		if pt, ok = args[1].(eval.PType); ok {
			return NewTypeAliasType(name.String(), nil, pt)
		}
		var rt *RuntimeValue
		if rt, ok = args[1].(*RuntimeValue); ok {
			var ex parser.Expression
			if ex, ok = rt.Interface().(parser.Expression); ok {
				return NewTypeAliasType(name.String(), ex, nil)
			}
		}
		panic(NewIllegalArgumentType2(`TypeAlias[]`, 1, `Type or Expression`, args[1]))
	default:
		panic(errors.NewIllegalArgumentCount(`TypeAlias[]`, `0 or 2`, len(args)))
	}
}

func (t *TypeAliasType) Accept(v eval.Visitor, g eval.Guard) {
	if g == nil {
		g = make(eval.Guard)
	}
	if g.Seen(t, nil) {
		return
	}
	v(t)
	t.resolvedType.Accept(v, g)
}

func (t *TypeAliasType) Default() eval.PType {
	return typeAliasType_DEFAULT
}

func (t *TypeAliasType) Equals(o interface{}, g eval.Guard) bool {
	if ot, ok := o.(*TypeAliasType); ok && t.name == ot.name {
		if g == nil {
			g = make(eval.Guard)
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

func (t *TypeAliasType) Get(key string) (eval.PValue, bool) {
	switch key {
	case `name`:
		return WrapString(t.name), true
	case `resolved_type`:
		return t.resolvedType, true
	default:
		return nil, false
	}
}

func (t *TypeAliasType) Loader() eval.Loader {
	return t.loader
}

func (t *TypeAliasType) IsAssignable(o eval.PType, g eval.Guard) bool {
	if g == nil {
		g = make(eval.Guard)
	}
	if g.Seen(t, o) {
		return true
	}
	return GuardedIsAssignable(t.ResolvedType(), o, g)
}

func (t *TypeAliasType) IsInstance(o eval.PValue, g eval.Guard) bool {
	if g == nil {
		g = make(eval.Guard)
	}
	if g.Seen(t, o) {
		return true
	}
	return GuardedIsInstance(t.ResolvedType(), o, g)
}

func (t *TypeAliasType) MetaType() eval.ObjectType {
	return TypeAlias_Type
}

func (t *TypeAliasType) Name() string {
	return t.name
}

func (t *TypeAliasType) Resolve(c eval.Context) eval.PType {
	if t.resolvedType != nil {
		panic(fmt.Sprintf(`Attempt to resolve already resolved type %s`, t.name))
	}
	t.resolvedType = c.ResolveType(t.typeExpression)
	t.loader = c.Loader()
	return t
}

func (t *TypeAliasType) ResolvedType() eval.PType {
	if t.resolvedType == nil {
		panic(fmt.Sprintf("Reference to unresolved type '%s'", t.name))
	}
	return t.resolvedType
}

func (t *TypeAliasType) String() string {
	return eval.ToString2(t, EXPANDED)
}

func (t *TypeAliasType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	f := eval.GetFormat(s.FormatMap(), t.Type())
	if t.name == `UnresolvedAlias` {
		io.WriteString(b, `TypeAlias`)
	} else {
		io.WriteString(b, t.name)
		if !(f.IsAlt() && f.FormatChar() == 'b') {
			return
		}
		if g == nil {
			g = make(eval.RDetect)
		} else if g[t] {
			return
		}
		g[t] = true
		io.WriteString(b, ` = `)

		// TODO: Need to be adjusted when included in TypeSet
		t.resolvedType.ToString(b, s, g)
	}
}

func (t *TypeAliasType) Type() eval.PType {
	return &TypeType{t}
}

var typeAliasType_DEFAULT = &TypeAliasType{`UnresolvedAlias`, nil, defaultType_DEFAULT, nil}
