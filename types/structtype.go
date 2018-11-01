package types

import (
	"io"
	"sync"

	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/eval"
)

type (
	StructElement struct {
		name  string
		key   eval.Type
		value eval.Type
	}

	StructType struct {
		lock          sync.Mutex
		elements      []*StructElement
		hashedMembers map[string]*StructElement
	}
)

var Struct_Element eval.Type

var Struct_Type eval.ObjectType

func init() {
	Struct_Element = newObjectType(`Pcore::StructElement`,
		`{
	attributes => {
		key_type => Type,
    value_type => Type
	}
}`, func(ctx eval.Context, args []eval.Value) eval.Value {
			return NewStructElement(args[0], args[1].(eval.Type))
		})

	Struct_Type = newObjectType(`Pcore::StructType`,
		`Pcore::AnyType {
	attributes => {
		elements => Array[Pcore::StructElement]
	}
}`, func(ctx eval.Context, args []eval.Value) eval.Value {
			return NewStructType2(args[0].(*ArrayValue).AppendTo([]eval.Value{})...)
		})
}

func NewStructElement(key eval.Value, value eval.Type) *StructElement {

	var (
		name    string
		keyType eval.Type
	)

	switch key.(type) {
	case *StringValue:
		v := key.(*StringValue)
		keyType = v.PType()
		if isAssignable(value, DefaultUndefType()) {
			keyType = NewOptionalType(keyType)
		}
		name = v.String()
	case *StringType:
		strType := key.(*StringType)
		name = strType.value
		keyType = strType
	case *OptionalType:
		optType := key.(*OptionalType)
		if strType, ok := optType.typ.(*StringType); ok {
			name = strType.value
			keyType = optType
		}
	}

	if keyType == nil || name == `` {
		panic(NewIllegalArgumentType2(`StructElement`, 0, `Variant[String[1], Type[String[1]], , Type[Optional[String[1]]]]`, key))
	}
	return &StructElement{name, keyType, value}
}

func NewStructElement2(key string, value eval.Type) *StructElement {
	return NewStructElement(WrapString(key), value)
}

func DefaultStructType() *StructType {
	return structType_DEFAULT
}

func NewStructType(elements []*StructElement) *StructType {
	if len(elements) == 0 {
		return DefaultStructType()
	}
	return &StructType{elements: elements}
}

func NewStructType2(args ...eval.Value) *StructType {
	switch len(args) {
	case 0:
		return DefaultStructType()
	case 1:
		hash, ok := args[0].(eval.OrderedMap)
		if !ok {
			panic(NewIllegalArgumentType2(`Struct[]`, 0, `Hash[Variant[String[1], Optional[String[1]]], Type]`, args[0]))
		}
		top := hash.Len()
		elems := make([]*StructElement, top)
		hash.EachWithIndex(func(v eval.Value, idx int) {
			e := v.(*HashEntry)
			vt, ok := e.Value().(eval.Type)
			if !ok {
				panic(NewIllegalArgumentType2(`StructElement`, 1, `Type`, v))
			}
			elems[idx] = NewStructElement(e.Key(), vt)
		})
		return NewStructType(elems)
	default:
		panic(errors.NewIllegalArgumentCount(`Struct`, `0 - 1`, len(args)))
	}
}

func (s *StructElement) Accept(v eval.Visitor, g eval.Guard) {
	s.key.Accept(v, g)
	s.value.Accept(v, g)
}

func (s *StructElement) ActualKeyType() eval.Type {
	if ot, ok := s.key.(*OptionalType); ok {
		return ot.typ
	}
	return s.key
}

func (s *StructElement) Equals(o interface{}, g eval.Guard) bool {
	if ose, ok := o.(*StructElement); ok {
		return s.key.Equals(ose.key, g) && s.value.Equals(ose.value, g)
	}
	return false
}

func (s *StructElement) String() string {
	return eval.ToString(s)
}

func (s *StructElement) PType() eval.Type {
	return Struct_Element
}

func (s *StructElement) Key() eval.Type {
	return s.key
}

func (s *StructElement) Name() string {
	return s.name
}

func (s *StructElement) Optional() bool {
	_, ok := s.key.(*OptionalType)
	return ok
}

func (s *StructElement) resolve(c eval.Context) {
	s.key = resolve(c, s.key)
	s.value = resolve(c, s.value)
}

func (s *StructElement) ToString(bld io.Writer, format eval.FormatContext, g eval.RDetect) {
	optionalValue := isAssignable(s.value, undefType_DEFAULT)
	if _, ok := s.key.(*OptionalType); ok {
		if optionalValue {
			io.WriteString(bld, s.name)
		} else {
			s.key.ToString(bld, format, g)
		}
	} else {
		if optionalValue {
			NewNotUndefType(s.key).ToString(bld, format, g)
		} else {
			io.WriteString(bld, s.name)
		}
	}
	io.WriteString(bld, ` => `)
	s.value.ToString(bld, format, g)
}

func (s *StructElement) Value() eval.Type {
	return s.value
}

func (t *StructType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
	for _, element := range t.elements {
		element.Accept(v, g)
	}
}

func (t *StructType) Default() eval.Type {
	return structType_DEFAULT
}

func (t *StructType) Equals(o interface{}, g eval.Guard) bool {
	if ot, ok := o.(*StructType); ok && len(t.elements) == len(ot.elements) {
		for idx, element := range t.elements {
			if !element.Equals(ot.elements[idx], g) {
				return false
			}
		}
		return true
	}
	return false
}

func (t *StructType) Generic() eval.Type {
	al := make([]*StructElement, len(t.elements))
	for idx, e := range t.elements {
		al[idx] = &StructElement{e.name, eval.GenericType(e.key), eval.GenericType(e.value)}
	}
	return NewStructType(al)
}

func (t *StructType) Elements() []*StructElement {
	return t.elements
}

func (t *StructType) Get(key string) (value eval.Value, ok bool) {
	switch key {
	case `elements`:
		els := make([]eval.Value, len(t.elements))
		for i, e := range t.elements {
			els[i] = e
		}
		return WrapArray(els), true
	}
	return nil, false
}

func (t *StructType) HashedMembers() map[string]*StructElement {
	t.lock.Lock()
	if t.hashedMembers == nil {
		t.hashedMembers = t.HashedMembersCloned()
	}
	t.lock.Unlock()
	return t.hashedMembers
}

func (t *StructType) HashedMembersCloned() map[string]*StructElement {
	hashedMembers := make(map[string]*StructElement, len(t.elements))
	for _, elem := range t.elements {
		hashedMembers[elem.name] = elem
	}
	return hashedMembers
}

func (t *StructType) IsAssignable(o eval.Type, g eval.Guard) bool {
	switch o.(type) {
	case *StructType:
		st := o.(*StructType)
		hm := st.HashedMembers()
		matched := 0
		for _, e1 := range t.elements {
			e2 := hm[e1.name]
			if e2 == nil {
				if !GuardedIsAssignable(e1.key, undefType_DEFAULT, g) {
					return false
				}
			} else {
				if !(GuardedIsAssignable(e1.key, e2.key, g) && GuardedIsAssignable(e1.value, e2.value, g)) {
					return false
				}
				matched++
			}
		}
		return matched == len(hm)
	case *HashType:
		ht := o.(*HashType)
		required := 0
		for _, e := range t.elements {
			if !GuardedIsAssignable(e.key, undefType_DEFAULT, g) {
				if !GuardedIsAssignable(e.value, ht.valueType, g) {
					return false
				}
				required++
			}
		}
		if required > 0 && !GuardedIsAssignable(stringType_DEFAULT, ht.keyType, g) {
			return false
		}
		return GuardedIsAssignable(NewIntegerType(int64(required), int64(len(t.elements))), ht.size, g)
	default:
		return false
	}
}

func (t *StructType) IsInstance(o eval.Value, g eval.Guard) bool {
	ov, ok := o.(*HashValue)
	if !ok {
		return false
	}
	matched := 0
	for _, element := range t.elements {
		key := element.name
		v, ok := ov.Get(WrapString(key))
		if !ok {
			if !GuardedIsAssignable(element.key, undefType_DEFAULT, g) {
				return false
			}
		} else {
			if !GuardedIsInstance(element.value, v, g) {
				return false
			}
			matched++
		}
	}
	return matched == ov.Len()
}

func (t *StructType) MetaType() eval.ObjectType {
	return Struct_Type
}

func (t *StructType) Name() string {
	return `Struct`
}

func (t *StructType) Parameters() []eval.Value {
	top := len(t.elements)
	if top == 0 {
		return eval.EMPTY_VALUES
	}
	entries := make([]*HashEntry, top)
	for idx, s := range t.elements {
		optionalValue := isAssignable(s.value, undefType_DEFAULT)
		var key eval.Value
		if _, ok := s.key.(*OptionalType); ok {
			if optionalValue {
				key = WrapString(s.name)
			} else {
				key = s.key
			}
		} else {
			if optionalValue {
				key = NewNotUndefType(s.key)
			} else {
				key = WrapString(s.name)
			}
		}
		entries[idx] = WrapHashEntry(key, s.value)
	}
	return []eval.Value{WrapHash(entries)}
}

func (t *StructType) Resolve(c eval.Context) eval.Type {
	for _, e := range t.elements {
		e.resolve(c)
	}
	return t
}

func (t *StructType) Size() *IntegerType {
	required := 0
	for _, e := range t.elements {
		if !GuardedIsAssignable(e.key, undefType_DEFAULT, nil) {
			required++
		}
	}
	return NewIntegerType(int64(required), int64(len(t.elements)))
}

func (t *StructType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *StructType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *StructType) PType() eval.Type {
	return &TypeType{t}
}

var structType_DEFAULT = &StructType{elements: []*StructElement{}}
