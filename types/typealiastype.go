package types

import (
	"fmt"
	"io"

	"github.com/lyraproj/puppet-evaluator/utils"

	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-parser/parser"
)

type TypeAliasType struct {
	name           string
	typeExpression interface{}
	resolvedType   eval.Type
	loader         eval.Loader
}

var TypeAliasMetaType eval.ObjectType

func init() {
	TypeAliasMetaType = newObjectType(`Pcore::TypeAlias`,
		`Pcore::AnyType {
	attributes => {
		name => String[1],
		resolved_type => {
			type => Optional[Type],
			value => undef
		}
	}
}`, func(ctx eval.Context, args []eval.Value) eval.Value {
			return newTypeAliasType2(args...)
		})
}

func DefaultTypeAliasType() *TypeAliasType {
	return typeAliasTypeDefault
}

// NewTypeAliasType creates a new TypeAliasType from a name and a typeExpression which
// must either be a *DeferredType, a parser.Expression, or nil. If it is nil, the
// resolved Type must be given.
func NewTypeAliasType(name string, typeExpression interface{}, resolvedType eval.Type) *TypeAliasType {
	return &TypeAliasType{name, typeExpression, resolvedType, nil}
}

func newTypeAliasType2(args ...eval.Value) *TypeAliasType {
	switch len(args) {
	case 0:
		return DefaultTypeAliasType()
	case 2:
		name, ok := args[0].(stringValue)
		if !ok {
			panic(NewIllegalArgumentType(`TypeAlias`, 0, `String`, args[0]))
		}
		var pt eval.Type
		if pt, ok = args[1].(eval.Type); ok {
			return NewTypeAliasType(string(name), nil, pt)
		}
		var rt *RuntimeValue
		if rt, ok = args[1].(*RuntimeValue); ok {
			return NewTypeAliasType(string(name), rt.Interface(), nil)
		}
		panic(NewIllegalArgumentType(`TypeAlias[]`, 1, `Type or Expression`, args[1]))
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

func (t *TypeAliasType) Default() eval.Type {
	return typeAliasTypeDefault
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

func (t *TypeAliasType) Get(key string) (eval.Value, bool) {
	switch key {
	case `name`:
		return stringValue(t.name), true
	case `resolved_type`:
		return t.resolvedType, true
	default:
		return nil, false
	}
}

func (t *TypeAliasType) Loader() eval.Loader {
	return t.loader
}

func (t *TypeAliasType) IsAssignable(o eval.Type, g eval.Guard) bool {
	if g == nil {
		g = make(eval.Guard)
	}
	if g.Seen(t, o) {
		return true
	}
	return GuardedIsAssignable(t.ResolvedType(), o, g)
}

func (t *TypeAliasType) IsInstance(o eval.Value, g eval.Guard) bool {
	if g == nil {
		g = make(eval.Guard)
	}
	if g.Seen(t, o) {
		return true
	}
	return GuardedIsInstance(t.ResolvedType(), o, g)
}

func (t *TypeAliasType) MetaType() eval.ObjectType {
	return TypeAliasMetaType
}

func (t *TypeAliasType) Name() string {
	return t.name
}

func (t *TypeAliasType) Resolve(c eval.Context) eval.Type {
	if t.resolvedType == nil {
		if dt, ok := t.typeExpression.(*DeferredType); ok {
			t.resolvedType = dt.Resolve(c)
		} else {
			t.resolvedType = c.(eval.EvaluationContext).ResolveType(t.typeExpression.(parser.Expression))
		}
		t.loader = c.Loader()
	}
	return t
}

func (t *TypeAliasType) ResolvedType() eval.Type {
	if t.resolvedType == nil {
		panic(fmt.Sprintf("Reference to unresolved type '%s'", t.name))
	}
	return t.resolvedType
}

func (t *TypeAliasType) String() string {
	return eval.ToString2(t, Expanded)
}

func (t *TypeAliasType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	f := eval.GetFormat(s.FormatMap(), t.PType())
	if t.name == `UnresolvedAlias` {
		utils.WriteString(b, `TypeAlias`)
	} else {
		utils.WriteString(b, t.name)
		if !(f.IsAlt() && f.FormatChar() == 'b') {
			return
		}
		if g == nil {
			g = make(eval.RDetect)
		} else if g[t] {
			utils.WriteString(b, `<recursive reference>`)
			return
		}
		g[t] = true
		utils.WriteString(b, ` = `)

		// TODO: Need to be adjusted when included in TypeSet
		t.resolvedType.ToString(b, s, g)
		delete(g, t)
	}
}

func (t *TypeAliasType) PType() eval.Type {
	return &TypeType{t}
}

var typeAliasTypeDefault = &TypeAliasType{`UnresolvedAlias`, nil, defaultTypeDefault, nil}
