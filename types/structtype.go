package types

import (
	. "io"
	"sync"

	. "github.com/puppetlabs/go-evaluator/errors"
	. "github.com/puppetlabs/go-evaluator/evaluator"
)

type (
	StructElement struct {
		name  string
		key   PType
		value PType
	}

	StructType struct {
		lock          sync.Mutex
		elements      []*StructElement
		hashedMembers map[string]*StructElement
	}
)

func NewStructElement(key PValue, value PType) *StructElement {

	var (
		name    string
		keyType PType
	)

	switch key.(type) {
	case *StringValue:
		v := key.(*StringValue)
		keyType = v.Type()
		if isAssignable(value, DefaultUndefType()) {
			keyType = NewOptionalType(keyType)
		}
		name = v.String()
	case *OptionalType:
		optType := key.(*OptionalType)
		if strType, ok := optType.typ.(*StringType); ok {
			name = strType.value
			keyType = optType
		}
	}

	if keyType == nil || name == `` {
		panic(NewIllegalArgumentType2(`StructElement`, 0, `Variant[String[1], Optional[String[1]]]`, key))
	}
	return &StructElement{name, keyType, value}
}

func NewStructElement2(key string, value PType) *StructElement {
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

func NewStructType2(args ...PValue) *StructType {
	switch len(args) {
	case 0:
		return DefaultStructType()
	case 1:
		hash, ok := args[0].(KeyedValue)
		if !ok {
			panic(NewIllegalArgumentType2(`Struct[]`, 0, `Hash[Variant[String[1], Optional[String[1]]], Type]`, args[0]))
		}
		top := hash.Len()
		elems := make([]*StructElement, top)
		for idx, key := range hash.Keys().Elements() {
			var vt PType
			v, _ := hash.Get(key)
			if vt, ok = v.(PType); ok {
				elems[idx] = NewStructElement(key, vt)
				continue
			}
			panic(NewIllegalArgumentType2(`StructElement`, 1, `Type`, v))
		}
		return NewStructType(elems)
	default:
		panic(NewIllegalArgumentCount(`Struct`, `0 - 1`, len(args)))
	}
}

func (s *StructElement) ActualKeyType() PType {
	if ot, ok := s.key.(*OptionalType); ok {
		return ot.typ
	}
	return s.key
}

func (s *StructElement) Equals(o *StructElement, g Guard) bool {
	return s.key.Equals(o.key, g) && s.value.Equals(o.value, g)
}

func (s *StructElement) ToString(bld Writer, format FormatContext, g RDetect) {
	optionalValue := isAssignable(s.value, undefType_DEFAULT)
	if _, ok := s.key.(*OptionalType); ok {
		if optionalValue {
			WriteString(bld, s.name)
		} else {
			s.key.ToString(bld, format, g)
		}
	} else {
		if optionalValue {
			NewNotUndefType(s.key).ToString(bld, format, g)
		} else {
			WriteString(bld, s.name)
		}
	}
	WriteString(bld, ` => `)
	s.value.ToString(bld, format, g)
}

func (t *StructType) Default() PType {
	return structType_DEFAULT
}

func (t *StructType) Equals(o interface{}, g Guard) bool {
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

func (t *StructType) Generic() PType {
	al := make([]*StructElement, len(t.elements))
	for idx, e := range t.elements {
		al[idx] = &StructElement{e.name, GenericType(e.key), GenericType(e.value)}
	}
	return NewStructType(al)
}

func (t *StructType) HashedMembers() map[string]*StructElement {
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.hashedMembers == nil {
		t.hashedMembers = make(map[string]*StructElement, len(t.elements))
		for _, elem := range t.elements {
			t.hashedMembers[elem.name] = elem
		}
	}
	return t.hashedMembers
}

func (t *StructType) IsAssignable(o PType, g Guard) bool {
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
				if !GuardedIsAssignable(e1.key, e2.key, g) && GuardedIsAssignable(e1.value, e2.value, g) {
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

func (t *StructType) IsInstance(o PValue, g Guard) bool {
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

func (t *StructType) Name() string {
	return `Struct`
}

func (t *StructType) Parameters() []PValue {
	top := len(t.elements)
	if top == 0 {
		return EMPTY_VALUES
	}
	entries := make([]*HashEntry, top)
	for idx, s := range t.elements {
		optionalValue := isAssignable(s.value, undefType_DEFAULT)
		var key PValue
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
	return []PValue{WrapHash(entries)}
}

func (t *StructType) String() string {
	return ToString2(t, NONE)
}

func (t *StructType) ToString(b Writer, s FormatContext, g RDetect) {
	TypeToString(t, b, s, g)
}

func (t *StructType) Type() PType {
	return &TypeType{t}
}

var structType_DEFAULT = &StructType{elements: []*StructElement{}}
