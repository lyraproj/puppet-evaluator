package types

import (
	"io"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-issues/issue"
	"strings"
	"strconv"
)

type LikeType struct {
	baseType eval.PType
	resolved eval.PType
	navigation string
}

var Like_Type eval.ObjectType

func init() {
	Like_Type = newObjectType(`Pcore::Like`,
		`Pcore::AnyType {
	attributes => {
    base_type => Type,
		navigation => String[1]
	}
}`, func(ctx eval.Context, args []eval.PValue) eval.PValue {
			return NewLikeType2(args...)
		})
}

func DefaultLikeType() *LikeType {
	return typeOfType_DEFAULT
}

func NewLikeType(baseType eval.PType, navigation string) *LikeType {
	return &LikeType{baseType: baseType, navigation: navigation}
}

func NewLikeType2(args ...eval.PValue) *LikeType {
	switch len(args) {
	case 0:
		return DefaultLikeType()
	case 2:
		if tp, ok := args[0].(eval.PType); ok {
			if an, ok := args[1].(*StringValue); ok {
				return &LikeType{baseType: tp, navigation: an.String()}
			} else {
				panic(NewIllegalArgumentType2(`Like[]`, 1, `String`, args[1]))
			}
		} else {
			panic(NewIllegalArgumentType2(`Like[]`, 0, `Type`, args[1]))
		}
	default:
		panic(errors.NewIllegalArgumentCount(`Like[]`, `0 or 2`, len(args)))
	}
}

func (t *LikeType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
	t.baseType.Accept(v, g)
}

func (t *LikeType) Default() eval.PType {
	return typeOfType_DEFAULT
}

func (t *LikeType) Equals(o interface{}, g eval.Guard) bool {
	if ot, ok := o.(*LikeType); ok {
		return t.navigation == ot.navigation && t.baseType.Equals(ot.baseType, g)
	}
	return false
}

func (t *LikeType) Get(key string) (eval.PValue, bool) {
	switch key {
	case `base_type`:
		return t.baseType, true
	case `navigation`:
		return WrapString(t.navigation), true
	default:
		return nil, false
	}
}

func (t *LikeType) IsAssignable(o eval.PType, g eval.Guard) bool {
	return t.Resolve(nil).IsAssignable(o, g)
}

func (t *LikeType) IsInstance(o eval.PValue, g eval.Guard) bool {
	return t.Resolve(nil).IsInstance(o, g)
}

func (t *LikeType) MetaType() eval.ObjectType {
	return Like_Type
}

func (t *LikeType) Name() string {
	return `Like`
}

func (t *LikeType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *LikeType) Parameters() []eval.PValue {
	if *t == *typeOfType_DEFAULT {
		return eval.EMPTY_VALUES
	}
	return []eval.PValue{t.baseType, WrapString(t.navigation)}
}

func (t *LikeType) Resolve(c eval.Context) eval.PType {
	if t.resolved != nil {
		return t.resolved
	}
	bt := t.baseType
	bv := bt.(eval.PValue)
	ok := true
	for _, part := range strings.Split(t.navigation, `.`) {
		if c, bv, ok = navigate(c, bv, part); !ok {
			panic(eval.Error(eval.EVAL_UNRESOLVED_TYPE_OF, issue.H{`type`: t.baseType, `navigation`: t.navigation}))
		}
	}
	if bt, ok = bv.(eval.PType); ok {
		t.resolved = bt
		return bt
	}
	panic(eval.Error(eval.EVAL_UNRESOLVED_TYPE_OF, issue.H{`type`: t.baseType, `navigation`: t.navigation}))
}

func (t *LikeType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *LikeType) Type() eval.PType {
	return &TypeType{t}
}

func navigate(c eval.Context, value eval.PValue, member string) (eval.Context, eval.PValue, bool) {
	if typ, ok := value.(eval.PType); ok {
		if po, ok := typ.(eval.TypeWithCallableMembers); ok {
			if m, ok := po.Member(member); ok {
				if a, ok := m.(eval.Attribute); ok {
					return c, a.Type(), true
				}
				if f, ok := m.(eval.Function); ok {
					return c, f.Type().(*CallableType).ReturnType(), true
				}
			}
		} else if st, ok := typ.(*StructType); ok {
			if m, ok := st.HashedMembers()[member]; ok {
				return c, m.Value(), true
			}
		} else if tt, ok := typ.(*TupleType); ok {
			if n, err := strconv.ParseInt(member, 0, 64); err == nil {
				if et, ok := tt.At(int(n)).(eval.PType); ok {
					return c, et, true
				}
			}
		} else if ta, ok := typ.(*TypeAliasType); ok {
			return navigate(c, ta.ResolvedType(), member)
		} else {
			if m, ok := typ.MetaType().Member(member); ok {
				if c == nil {
					c = eval.CurrentContext()
				}
				return c, m.Call(c, typ, nil, []eval.PValue{}), true
			}
		}
	} else {
		if po, ok := value.Type().(eval.TypeWithCallableMembers); ok {
			if m, ok := po.Member(member); ok {
				if c == nil {
					c = eval.CurrentContext()
				}
				return c, m.Call(c, value, nil, []eval.PValue{}), true
			}
		}
	}
	return c, nil, false
}

var typeOfType_DEFAULT = &LikeType{baseType: DefaultAnyType()}
