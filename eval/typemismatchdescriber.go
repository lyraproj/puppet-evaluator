package eval

import (
	"fmt"
	. "github.com/puppetlabs/go-evaluator/evaluator"
	"strings"
	"reflect"
	. "github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-parser/parser"
	"strconv"
	"math"
	"github.com/puppetlabs/go-evaluator/utils"
)

type (
	pathElement struct {
		key string
		pathType pathType
	}

	pathType string

	mismatchClass string

	mismatch interface {
		canonicalPath() []*pathElement
		path() []*pathElement
		pathString() string
		setPath(path []*pathElement)
		text() string
		class() mismatchClass
		equals(other mismatch) bool
	}

	expectedActualMismatch interface {
		mismatch
		actual() PType
		expected() PType
		setExpected(expected PType)
	}

	sizeMismatch interface {
		expectedActualMismatch
		from() int64
		to() int64
	}

	sizeMismatchFunc func(path []*pathElement, expected *IntegerType, actual *IntegerType) mismatch

  basicMismatch struct {
		vpath []*pathElement
	}

	unexpectedBlock struct { basicMismatch }

	missingRequiredBlock struct { basicMismatch }

	keyMismatch struct {
		basicMismatch
		key string
	}

	missingKey struct { keyMismatch }
	missingParameter struct { keyMismatch }
	extraneousKey struct { keyMismatch }
	invalidParameter struct { keyMismatch }
	unresolvedTypeReference struct { keyMismatch }

	basicEAMismatch struct {
		basicMismatch
		actualType PType
		expectedType PType
	}

	typeMismatch struct { basicEAMismatch }
	patternMismatch struct { typeMismatch }
	basicSizeMismatch struct { basicEAMismatch }
	countMismatch struct { basicSizeMismatch }
)

var NO_MISMATCH = []mismatch{}

const (
	subject = pathType(``)
	entry  = pathType(`entry`)
	entryKey = pathType(`key of entry`)
	_parameter = pathType(`parameter`)
	_return = pathType(`return`)
	block = pathType(`block`)
	index = pathType(`index`)
	variant = pathType(`variant`)
	signature = pathType(`singature`)

	countMismatchClass = mismatchClass(`countMismatch`)
	missingKeyClass = mismatchClass(`missingKey`)
	missingParameterClass = mismatchClass(`missingParameter`)
	missingRequiredBlockClass = mismatchClass(`missingRequiredBlock`)
	extraneousKeyClass = mismatchClass(`extraneousKey`)
	invalidParameterClass = mismatchClass(`invalidParameter`)
	patternMismatchClass = mismatchClass(`patternMismatch`)
	sizeMismatchClass = mismatchClass(`sizeMismatch`)
	typeMismatchClass = mismatchClass(`typeMismatch`)
	unexpectedBlockClass = mismatchClass(`unexpectedBlock`)
	unresolvedTypeReferenceClass = mismatchClass(`unresolvedTypeReference`)
)

func (p pathType) format(key string) string {
	if p == subject || p == signature {
		return key
	}
	if p == block && key == `block` {
		return key
	}
	if p == _parameter && utils.IsDecimalInteger(key) {
		return fmt.Sprintf("%s %s", p, key)
	}
	return fmt.Sprintf("%s '%s'", p, key)
}

func (pe *pathElement) String() string {
	return pe.pathType.format(pe.key)
}

func copyMismatch(m mismatch) mismatch {
	orig := reflect.Indirect(reflect.ValueOf(m))
	c := reflect.New(orig.Type())
	c.Elem().Set(orig)
	return c.Interface().(mismatch)
}

func withPath(m mismatch, path []*pathElement) mismatch {
	m = copyMismatch(m)
	m.setPath(path)
	return m
}

func chopPath(m mismatch, index int) mismatch {
	p := m.path()
	if index >= len(p) {
		return m
	}
	cp := make([]*pathElement, 0)
	for i, pe := range p {
		if i != index {
			cp = append(cp, pe)
		}
	}
	return withPath(m, cp)
}

func merge(m mismatch, o mismatch, path []*pathElement) mismatch {
	m = withPath(m, path)
	switch m.(type) {
	case sizeMismatch:
		esm := m.(sizeMismatch)
		if osm, ok := o.(sizeMismatch); ok {
			min := esm.from()
			if min > osm.from() {
				min = osm.from()
			}
			max := esm.to()
			if max < osm.to() {
				max = osm.to()
			}
			esm.setExpected(NewIntegerType(min, max))
		}
	case expectedActualMismatch:
		eam := m.(expectedActualMismatch)
		if oam, ok := o.(expectedActualMismatch); ok {
			eam.setExpected(oam.expected())
		}
	}
	return m
}

func joinPath(path []*pathElement) string {
	s := make([]string, len(path))
	for i, p := range path {
		s[i] = p.String()
	}
	return strings.Join(s, ` `)
}

func format(m mismatch) string {
	p := m.path()
	variant := ``
	position := ``
	if len(p) > 0 {
		f := p[0]
		if f.pathType == signature {
			variant = fmt.Sprintf(` %s`, f.String())
			p = p[1:]
		}
		if len(p) > 0 {
			position = fmt.Sprintf(` %s`, joinPath(p))
		}
	}
	return message(m, variant, position)
}

func message(m mismatch, variant string, position string) string {
	if variant == `` && position == `` {
		return m.text()
	}
	return fmt.Sprintf("%s%s %s", variant, position, m.text())
}

func (m *basicMismatch) canonicalPath() []*pathElement {
	result := make([]*pathElement, 0)
	for _, p := range m.vpath {
		if p.pathType != variant && p.pathType != signature {
			result = append(result, p)
		}
	}
	return result
}

func (m *basicMismatch) class() mismatchClass {
	return ``
}

func (m *basicMismatch) path() []*pathElement {
	return m.vpath
}

func (m *basicMismatch) setPath(path []*pathElement) {
	m.vpath = path
}

func (m *basicMismatch) pathString() string {
	return joinPath(m.vpath)
}

func (m *basicMismatch) text() string {
	return ``
}

func (m *basicMismatch) equals(other mismatch) bool {
	return m.class() == other.class() && pathEquals(m.vpath, other.path())
}

func newUnexpectedBlock(path []*pathElement) mismatch {
	return &unexpectedBlock{basicMismatch{vpath:path}}
}

func (*unexpectedBlock) class() mismatchClass {
	return unexpectedBlockClass
}

func (*unexpectedBlock) text() string {
	return `does not expect a block`
}

func newMissingRequiredBlock(path []*pathElement) mismatch {
	return &missingRequiredBlock{basicMismatch{vpath:path}}
}

func (*missingRequiredBlock) class() mismatchClass {
	return missingRequiredBlockClass
}

func (*missingRequiredBlock) text() string {
	return `expects a block`
}

func (m *keyMismatch) equals(other mismatch) bool {
	if om, ok := other.(*keyMismatch); ok && pathEquals(m.vpath, other.path()) {
		return m.key == om.key
	}
	return false
}

func newMissingKey(path []*pathElement, key string) mismatch {
	return &missingKey{keyMismatch{basicMismatch{vpath:path}, key}}
}

func (*missingKey) class() mismatchClass {
	return missingKeyClass
}

func (m *missingKey) text() string {
	return fmt.Sprintf(`expects a value for key '%s'`, m.key)
}

func newMissingParameter(path []*pathElement, key string) mismatch {
	return &missingParameter{keyMismatch{basicMismatch{vpath:path}, key}}
}

func (*missingParameter) class() mismatchClass {
	return missingParameterClass
}

func (m *missingParameter) text() string {
	return fmt.Sprintf(`expects a value for parameter '%s'`, m.key)
}

func newExtraneousKey(path []*pathElement, key string) mismatch {
	return &extraneousKey{keyMismatch{basicMismatch{vpath:path}, key}}
}

func (*extraneousKey) class() mismatchClass {
	return extraneousKeyClass
}

func (m *extraneousKey) text() string {
	return fmt.Sprintf(`unrecognized key '%s'`, m.key)
}

func newInvalidParameter(path []*pathElement, key string) mismatch {
	return &invalidParameter{keyMismatch{basicMismatch{vpath:path}, key}}
}

func (*invalidParameter) class() mismatchClass {
	return invalidParameterClass
}

func (m *invalidParameter) text() string {
	return fmt.Sprintf(`has no parameter named '%s'`, m.key)
}

func newUnresolvedTypeReference(path []*pathElement, key string) mismatch {
	return &unresolvedTypeReference{keyMismatch{basicMismatch{vpath:path}, key}}
}

func (*unresolvedTypeReference) class() mismatchClass {
	return unresolvedTypeReferenceClass
}

func (m *unresolvedTypeReference) text() string {
	return fmt.Sprintf(`references an unresolved type '%s'`, m.key)
}

func (m *basicEAMismatch) equals(other mismatch) bool {
	if om, ok := other.(*basicEAMismatch); ok && pathEquals(m.vpath, other.path()) {
		return m.expectedType == om.expectedType && m.actualType == om.actualType
	}
	return false
}

func (ea *basicEAMismatch) expected() PType {
	return ea.expectedType
}

func (ea *basicEAMismatch) actual() PType {
	return ea.actualType
}

func (ea *basicEAMismatch) setExpected(expected PType) {
	ea.expectedType = expected
}

func newTypeMismatch(path []*pathElement, expected PType, actual PType) mismatch {
	return &typeMismatch{basicEAMismatch{basicMismatch{vpath:path}, actual, expected}}
}

func (*typeMismatch) class() mismatchClass {
	return typeMismatchClass
}

func (tm *typeMismatch) text() string {
	e := tm.expectedType
	a := tm.actualType
	multi := false
	optional := false
	if opt, ok := e.(*OptionalType); ok {
		e = opt.ContainedType()
		optional = true
	}
	as := ``
	es := ``
	if vt, ok := e.(*VariantType); ok {
		el := vt.Types()
		els := make([]string, 0, len(el))
		if reportDetailed(el, a) {
			as = detailedToActualToS(el, a)
			for i, e := range el {
				els[i] = e.String()
			}
		} else {
			for i, e := range el {
				els[i] = shortName(e)
			}
			as = shortName(a)
		}
		if optional {
			els = append([]string{`Undef`}, els...)
		}
		switch len(els) {
		case 1:
			es = els[0]
		case 2:
			es = fmt.Sprintf(`%s or %s`, els[0], els[1])
			multi = true
		default:
			es = fmt.Sprintf(`%s, or %s`, strings.Join(els[0:len(els)-1], `, `), els[len(els)-1])
			multi = true
		}
	} else {
		el := []PType{e}
		if reportDetailed(el, a) {
			as = detailedToActualToS(el, a)
			es = ToString2(e, EXPANDED)
		} else {
			as = shortName(a)
			es = shortName(e)
		}
	}

	if multi {
    return fmt.Sprintf(`expects a value of type %s, got %s`, es, as)
	}
	return fmt.Sprintf(`expects %s %s value, got %s`, parser.Article(es), es, as)
}

func shortName(t PType) string {
	if tc, ok := t.(TypeWithContainedType); ok && !(tc.ContainedType() == nil || tc.ContainedType() == DefaultAnyType()) {
		return fmt.Sprintf("%s[%s]", t.Name(), tc.ContainedType().Name())
	}
	return t.Name()
}

func detailedToActualToS(es []PType, a PType) string {
	es = allResolved(es)
	if alwaysFullyDetailed(es, a) {
		return ToString2(a, EXPANDED)
	}
	if anyAssignable(es, Generalize(a)) {
		return ToString2(a, EXPANDED)
	}
	return a.Name()
}

func anyAssignable(es []PType, a PType) bool {
	for _, e := range es {
		if IsAssignable(e, a) {
			return true
		}
	}
	return false
}

func alwaysFullyDetailed(es []PType, a PType) bool {
	for _, e := range es {
		if reflect.TypeOf(e) == reflect.TypeOf(a) {
			return true
		}
		if _, ok := e.(*TypeAliasType); ok {
			return true
		}
		if _, ok := a.(*TypeAliasType); ok {
			return true
		}
		if specialization(e, a) {
			return true
		}
	}
	return false
}

func specialization(e PType, a PType) (result bool) {
	switch e.(type) {
	case *StructType:
		_, result = a.(*HashType)
	case *TupleType:
		_, result = a.(*ArrayType)
	default:
		result = false
	}
	return
}

func allResolved(es []PType) []PType {
	rs := make([]PType, len(es))
	for i, e := range es {
		if ea, ok := e.(*TypeAliasType); ok {
			e = ea.ResolvedType()
		}
		rs[i] = e
	}
	return rs
}

func reportDetailed(e []PType, a PType) bool {
	return alwaysFullyDetailed(e, a) || assignableToDefault(e, a)
}

func assignableToDefault(es []PType, a PType) bool {
	for _, e := range es {
		if ea, ok := e.(*TypeAliasType); ok {
			e = ea.ResolvedType()
		}
		if IsAssignable(DefaultFor(e), a) {
      return true
		}
	}
	return false
}

func newPatternMismatch(path []*pathElement, expected PType, actual PType) mismatch {
	return &patternMismatch{
		typeMismatch{
			basicEAMismatch{
				basicMismatch{vpath:path}, actual, expected}}}
}

func (*patternMismatch) class() mismatchClass {
	return patternMismatchClass
}

func (m *patternMismatch) text() string {
	e := m.expectedType
	valuePfx := ``
	if oe, ok := e.(*OptionalType); ok {
		e = oe.ContainedType()
		valuePfx = `an undef value or `
	}
	return fmt.Sprintf(`expects %sa match for %s, got %s`,
		valuePfx, ToString2(e, EXPANDED), m.actualString())
}

func (m *patternMismatch) actualString() string {
	a := m.actualType
	if as, ok := a.(*StringType); ok && as.Value() != `` {
		return fmt.Sprintf(`'%s'`, as.Value())
	}
	return shortName(a)
}

func newSizeMismatch(path []*pathElement, expected *IntegerType, actual *IntegerType) mismatch {
	return &basicSizeMismatch{
			basicEAMismatch{
				basicMismatch{vpath:path}, actual, expected}}
}

func (*basicSizeMismatch) class() mismatchClass {
	return sizeMismatchClass
}

func (m *basicSizeMismatch) from() int64 {
	return m.expectedType.(*IntegerType).Min()
}

func (m *basicSizeMismatch) to() int64 {
	return m.expectedType.(*IntegerType).Max()
}

func (tm *basicSizeMismatch) text() string {
	return fmt.Sprintf(`expects size to be #{range_to_s(expected, '0')}, got #{range_to_s(actual, '0')}`,
		rangeToS(tm.expectedType.(*IntegerType), `0`),
		rangeToS(tm.actualType.(*IntegerType), `0`))
}

func rangeToS(rng *IntegerType, zeroString string) string {
	if rng.Min() == rng.Max() {
		if rng.Min() == 0 {
			return zeroString
		}
		return strconv.FormatInt(rng.Min(), 10)
	} else if rng.Min() == 0 {
		if rng.Max() == math.MaxInt64 {
			return `unbounded`
		}
		return fmt.Sprintf(`at most %d`, rng.Max())
	} else if rng.Max() == math.MaxInt64 {
		return fmt.Sprintf(`at least %d`, rng.Min())
	} else {
		return fmt.Sprintf("between %d and %d", rng.Min(), rng.Max())
	}
}

func newCountMismatch(path []*pathElement, expected *IntegerType, actual *IntegerType) mismatch {
	return &countMismatch{basicSizeMismatch{
		basicEAMismatch{
			basicMismatch{vpath:path}, actual, expected}}}
}

func (*countMismatch) class() mismatchClass {
	return countMismatchClass
}

func (tm *countMismatch) text() string {
	ei := tm.expectedType.(*IntegerType)
	suffix := `s`
	if ei.Min() == 1 && (ei.Max() == 1 || ei.Max() == math.MaxInt64) || ei.Min() == 0 && ei.Max() == 1 {
		suffix = ``
	}

	return fmt.Sprintf(`expects %s argument%s, got %s}`,
		rangeToS(ei, `no`), suffix,
		rangeToS(tm.actualType.(*IntegerType), `none`))
}

func describeOptionalType(expected *OptionalType, original, actual PType, path []*pathElement) []mismatch {
  if _, ok := actual.(*UndefType); ok {
  	return NO_MISMATCH
	}
	if _, ok := original.(*TypeAliasType); !ok {
		// If the original expectation is an alias, it must now track the optional type instead
		original = expected
	}
	return internalDescribe(expected.ContainedType(), original, actual, path)
}

func describeEnumType(expected *EnumType, original, actual PType, path []*pathElement) []mismatch {
	return []mismatch{newPatternMismatch(path, original, actual)}
}

func describePatternType(expected *PatternType, original, actual PType, path []*pathElement) []mismatch {
	return []mismatch{newPatternMismatch(path, original, actual)}
}

func describeTypeAliasType(expected *TypeAliasType, original, actual PType, path []*pathElement) []mismatch {
	return internalDescribe(Normalize(expected.ResolvedType()), expected, actual, path)
}

func describeArrayType(expected *ArrayType, original, actual PType, path []*pathElement) []mismatch {
	descriptions := make([]mismatch, 0, 4)
	et := expected.ElementType()
	if ta, ok := actual.(*TupleType); ok {
		if IsAssignable(expected.Size(), ta.Size()) {
			for ax, at := range ta.Types() {
				if !IsAssignable(et, at) {
					descriptions = append(descriptions, internalDescribe(et, et, at, pathWith(path, &pathElement{strconv.Itoa(ax), index}))...)
				}
			}
		} else {
			descriptions = append(descriptions, newSizeMismatch(path, expected.Size(), ta.Size()))
		}
	} else if aa, ok := actual.(*ArrayType); ok {
		if IsAssignable(expected.Size(), aa.Size()) {
			descriptions = append(descriptions, newTypeMismatch(path, original, NewArrayType(aa.ElementType(), nil)))
		} else {
			descriptions = append(descriptions, newSizeMismatch(path, expected.Size(), aa.Size()))
		}
	} else {
		descriptions = append(descriptions, newTypeMismatch(path, original, actual))
	}
	return descriptions
}

func describeHashType(expected *HashType, original, actual PType, path []*pathElement) []mismatch {
	descriptions := make([]mismatch, 0, 4)
	kt := expected.KeyType()
	vt := expected.ValueType()
	if sa, ok := actual.(*StructType); ok {
		if IsAssignable(expected.Size(), sa.Size()) {
			for _, al := range sa.Elements() {
				descriptions = append(descriptions, internalDescribe(kt, kt, al.Key(), pathWith(path, &pathElement{al.Name(), entryKey}))...)
				descriptions = append(descriptions, internalDescribe(vt, vt, al.Value(), pathWith(path, &pathElement{al.Name(), entry}))...)
			}
		} else {
			descriptions = append(descriptions, newSizeMismatch(path, expected.Size(), sa.Size()))
		}
	} else if ha, ok := actual.(*HashType); ok {
		if IsAssignable(expected.Size(), ha.Size()) {
			descriptions = append(descriptions, newTypeMismatch(path, original, NewHashType(ha.KeyType(), ha.ValueType(), nil)))
		} else {
			descriptions = append(descriptions, newSizeMismatch(path, expected.Size(), ha.Size()))
		}
	} else {
		descriptions = append(descriptions, newTypeMismatch(path, original, actual))
	}
	return descriptions
}

func describeStructType(expected *StructType, original, actual PType, path []*pathElement) []mismatch {
	descriptions := make([]mismatch, 0, 4)
	if sa, ok := actual.(*StructType); ok {
		h2 := sa.HashedMembersCloned()
		for _, e1 := range expected.Elements() {
			key := e1.Name()
			e2, ok := h2[key]
			if ok {
				delete(h2, key)
				descriptions = append(descriptions, internalDescribe(e1.Key(), e1.Key(), e2.Key(), pathWith(path, &pathElement{key, entryKey}))...)
				descriptions = append(descriptions, internalDescribe(e1.Value(), e1.Value(), e2.Value(), pathWith(path, &pathElement{key, entry}))...)
			} else {
				descriptions = append(descriptions, newMissingKey(path, e1.Name()))
			}
		}
		for key := range h2 {
			descriptions = append(descriptions, newExtraneousKey(path, key))
		}
	} else if ha, ok := actual.(*HashType); ok {
		if IsAssignable(expected.Size(), ha.Size()) {
			descriptions = append(descriptions, newTypeMismatch(path, original, NewHashType(ha.KeyType(), ha.ValueType(), nil)))
		} else {
			descriptions = append(descriptions, newSizeMismatch(path, expected.Size(), ha.Size()))
		}
	} else {
		descriptions = append(descriptions, newTypeMismatch(path, original, actual))
	}
	return descriptions
}

func describeTupleType(expected *TupleType, original, actual PType, path []*pathElement) []mismatch {
	return describeTuple(expected, original, actual, path, newCountMismatch)
}

func describeArgumentTuple(expected *TupleType, actual PType, path []*pathElement) []mismatch {
	return describeTuple(expected, expected, actual, path, newCountMismatch)
}

func describeTuple(expected *TupleType, original, actual PType, path []*pathElement, sm sizeMismatchFunc) []mismatch {
	if aa, ok := actual.(*ArrayType); ok {
		if len(expected.Types()) == 0 {
			return NO_MISMATCH
		}
		t2Entry := aa.ElementType()
		if t2Entry == DefaultAnyType() {
			// Array of anything can not be assigned (unless tuple is tuple of anything) - this case
			// was handled at the top of this method.
			return []mismatch{newTypeMismatch(path, original, actual)}
		}

		if !IsAssignable(expected.Size(), aa.Size()) {
			return []mismatch{sm(path, expected.Size(), aa.Size())}
		}

		descriptions := make([]mismatch, 0, 4)
		for ex, et := range expected.Types() {
			descriptions = append(descriptions, internalDescribe(et, et, aa.ElementType(),
				pathWith(path, &pathElement{strconv.Itoa(ex), index}))...)
		}
		return descriptions
	}

	if at, ok := actual.(*TupleType); ok {
		if expected.Equals(actual, nil)  {
			return NO_MISMATCH
		}

		if !IsAssignable(expected.Size(), at.Size()) {
			return []mismatch{sm(path, expected.Size(), at.Size())}
		}

		exl := len(expected.Types())
		if exl == 0 {
			return NO_MISMATCH
		}

		descriptions := make([]mismatch, 0, 4)
		for ax, at := range at.Types() {
			ex := ax
			if ax >= exl {
				ex = exl - 1
				ext := expected.Types()[ex]
				descriptions = append(descriptions, internalDescribe(ext, ext, at,
					pathWith(path, &pathElement{strconv.Itoa(ax), index }))...)
			}
		}
		return descriptions
	}

	return []mismatch{newTypeMismatch(path, original, actual)}
}

func pathEquals(a, b []*pathElement) bool {
	n := len(a)
	if n != len(b) {
		return false
	}
	for i := 0; i < n; i++ {
		if *(a[i]) != *(b[i]) {
			return false
		}
	}
	return true
}

func pathWith(path []*pathElement, elem *pathElement) []*pathElement {
	top := len(path)
	pc := make([]*pathElement, top + 1)
	copy(pc, path)
	pc[top] = elem
	return pc
}

func describeCallableType(expected *CallableType, original, actual PType, path []*pathElement) []mismatch {
	if ca, ok := actual.(*CallableType); ok {
		ep := expected.ParametersType()
		paramErrors := NO_MISMATCH
		if ep != nil {
			ap := ca.ParametersType()
			paramErrors = describeArgumentTuple(expected.ParametersType().(*TupleType), NilAs(DefaultTupleType(), ap), path)
		}
		if len(paramErrors) == 0 {
			er := expected.ReturnType()
			ar :=  NilAs(DefaultAnyType(), ca.ReturnType())
			if er == nil || IsAssignable(er, ar) {
				eb := expected.BlockType()
				ab := ca.BlockType()
				if eb == nil || IsAssignable(eb, NilAs(DefaultUndefType(), ab)) {
					return NO_MISMATCH
				}
				if ab == nil {
					return []mismatch{newMissingRequiredBlock(path)}
				}
				return []mismatch{newTypeMismatch(pathWith(path, &pathElement{``, block}), eb, ab)}
			}
			return []mismatch{newTypeMismatch(pathWith(path, &pathElement{``, _return}), er, ar)}
		}
		return paramErrors
	}
	return []mismatch{newTypeMismatch(path, original, actual)}
}

func describeAnyType(expected PType, original, actual PType, path []*pathElement) []mismatch {
	if IsAssignable(expected, actual) {
		return NO_MISMATCH
	}
	return []mismatch{newTypeMismatch(path, original, actual)}
}

func describe(expected PType, actual PType, path []*pathElement) []mismatch {
	var unresolved *TypeReferenceType
	expected.Accept(func(t PType) {
		if unresolved == nil {
			if ur, ok := t.(*TypeReferenceType); ok {
				unresolved = ur
			}
		}
	}, nil)

	if unresolved != nil {
		return []mismatch{newUnresolvedTypeReference(path, unresolved.TypeString())}
	}
	return internalDescribe(Normalize(expected), expected, actual, path)
}

func internalDescribe(expected PType, original, actual PType, path []*pathElement) []mismatch {
	switch expected.(type) {
	case *VariantType:
		return describeVariantType(expected.(*VariantType), original, actual, path)
	case *StructType:
		return describeStructType(expected.(*StructType), original, actual, path)
	case *HashType:
		return describeHashType(expected.(*HashType), original, actual, path)
	case *TupleType:
		return describeTupleType(expected.(*TupleType), original, actual, path)
	case *ArrayType:
		return describeArrayType(expected.(*ArrayType), original, actual, path)
	case *CallableType:
		return describeCallableType(expected.(*CallableType), original, actual, path)
	case *OptionalType:
		return describeOptionalType(expected.(*OptionalType), original, actual, path)
	case *PatternType:
		return describePatternType(expected.(*PatternType), original, actual, path)
	case *EnumType:
		return describeEnumType(expected.(*EnumType), original, actual, path)
	case *TypeAliasType:
		return describeTypeAliasType(expected.(*TypeAliasType), original, actual, path)
	default:
		return describeAnyType(expected, original, actual, path)
	}
}

func describeVariantType(expected *VariantType, original, actual PType, path []*pathElement) []mismatch {
	variantDescs := make([]mismatch, 0, len(expected.Types()))
	types := expected.Types()
	if _, ok := original.(*OptionalType); ok {
		types = CopyAppend(types, DefaultUndefType())
	}

	for ex, vt := range types {
		d := internalDescribe(vt, vt, actual, pathWith(path, &pathElement{strconv.Itoa(ex), variant}))
		if len(d) == 0 {
			return NO_MISMATCH
		}
		variantDescs = append(variantDescs, d...)
	}

	descs := mergeDescriptions(len(path), sizeMismatchClass, variantDescs)
	if _, ok := original.(*TypeAliasType); ok && len(descs) == 1 {
		// All variants failed in this alias so we report it as a mismatch on the alias
		// rather than reporting individual failures of the variants
		descs = []mismatch{newTypeMismatch(path, original, actual)}
	}
	return descs
}

func mergeDescriptions(varyingPathPosition int, sm mismatchClass, descriptions []mismatch) []mismatch {
	n := len(descriptions)
	if n == 0 {
		return NO_MISMATCH
	}

	for _, mClass := range []mismatchClass{sm, missingRequiredBlockClass, unexpectedBlockClass, typeMismatchClass} {
		mismatches := make([]mismatch, 0, 4)
		for _, desc := range descriptions {
			if desc.class() == mClass {
				mismatches = append(mismatches, desc)
			}
		}
		if len(mismatches) == n {
			// If they all have the same canonical path, then we can compact this into one
			prev := mismatches[0]
			for idx := 1; idx < n; idx++ {
				curr := mismatches[idx]
				if pathEquals(prev.canonicalPath(), curr.canonicalPath()) {
					prev = merge(prev, curr, prev.path())
				} else {
					prev = nil
					break
				}
			}
			if prev != nil {
				// Report the generic mismatch and skip the rest
				descriptions = []mismatch{prev}
				break
			}
		}
	}
	descriptions = unique(descriptions)
	if len(descriptions) == 1 {
		descriptions = []mismatch{chopPath(descriptions[0], varyingPathPosition)}
	}
	return descriptions
}

func unique(v []mismatch) []mismatch {
	u := make([]mismatch, 0, len(v))
	next: for _, m := range v {
		for _, x := range u {
			if m == x {
				break next
			}
		}
		u = append(u, m)
	}
	return u
}

func init() {
	DescribeSignatures = describeSignatures

	DescribeMismatch = func(name string, expected, actual PType) string {
		result := describe(expected, actual, []*pathElement{{name, subject}})
		switch len(result) {
		case 0:
			return ``
		case 1:
			return format(result[0])
		default:
			rs := make([]string, len(result))
			for i, r := range result {
				rs[i] = format(r)
			}
			return strings.Join(rs, "\n")
		}
	}
}

func describeSignatures(signatures []Signature, argsTuple PType, block Lambda) string {
	errorArrays := make([][]mismatch, len(signatures))
	allSet := true
	for ix, sg := range signatures {
		ae := describeSignatureArguments(sg, argsTuple.(*TupleType), []*pathElement{{strconv.Itoa(ix), signature}})
		errorArrays[ix] = ae
		if len(ae) == 0 {
			allSet = false
		}
	}
	// Skip block checks if all signatures have argument errors
	if !allSet {
		blockArrays := make([][]mismatch, len(signatures))
		bcCount := 0
		for ix, sg := range signatures {
			ae := describeSignatureBlock(sg, block, []*pathElement{{strconv.Itoa(ix), signature}})
			blockArrays[ix] = ae
			if len(ae) > 0 {
				bcCount++
			}
		}
		if bcCount == len(blockArrays) {
			// Skip argument errors when all alternatives have block errors
			errorArrays = blockArrays
		} else if bcCount > 0 {
			// Merge errors giving argument errors precedence over block errors
			for ix, ea := range errorArrays {
				if len(ea) == 0 {
					errorArrays[ix] = blockArrays[ix]
				}
			}
		}
	}
	if len(errorArrays) == 0 {
		return ``
	}

	errors := make([]mismatch, 0)
	for _, ea := range errorArrays {
		errors = append(errors, ea...)
	}

	errors = mergeDescriptions(0, countMismatchClass, errors)
	if len(errors) == 1 {
		return format(errors[0])
	}

	var result []string
	if len(signatures) == 1 {
		result = []string{fmt.Sprintf(`expects (%s)`, signatureString(signatures[0]))}
		for _, e := range errorArrays[0] {
			result = append(result, fmt.Sprintf(`  rejected:%s`, format(chopPath(e, 0))))
		}
	} else {
		result = []string{`expects one of:`}
		for ix, sg := range signatures {
			result = append(result, fmt.Sprintf(`  (%s)`, signatureString(sg)))
			for _, e := range errorArrays[ix] {
				result = append(result, fmt.Sprintf(`    rejected:%s`, format(chopPath(e, 0))))
			}
		}
	}
	return strings.Join(result, "\n")
}

func describeSignatureArguments(signature Signature, argsTuple *TupleType, path []*pathElement) []mismatch {
	paramsTuple := signature.ParametersType().(*TupleType)
	eSize := paramsTuple.Size()
	aSize := argsTuple.Size()
	if IsAssignable(eSize, aSize) {
		eTypes := paramsTuple.Types()
		eLast := len(eTypes) - 1
		eNames := signature.ParameterNames()
		for ax, aType := range argsTuple.Types() {
			ex := ax
			if ex > eLast {
				ex = eLast
			}
      eType := eTypes[ex]
      if !IsAssignable(eType, aType) {
      	descriptions := describe(eType, aType, pathWith(path, &pathElement{eNames[ex], _parameter}))
      	if len(descriptions) > 0 {
      		return descriptions
				}
			}
		}
		return NO_MISMATCH
	}
	return []mismatch{newCountMismatch(path, eSize, aSize)}
}

func describeSignatureBlock(signature Signature, aBlock Lambda, path []*pathElement) []mismatch {
	eBlock := signature.BlockType()
	if aBlock == nil {
		if eBlock == nil || IsAssignable(eBlock, DefaultUndefType()) {
			return NO_MISMATCH
		}
		return []mismatch{newMissingRequiredBlock(path)}
	}

	if eBlock == nil {
		return []mismatch{newUnexpectedBlock(path)}
	}
	return describe(eBlock, aBlock.Signature(), pathWith(path, &pathElement{signature.BlockName(), block}))
}

func signatureString(signature Signature) string {
	tuple := signature.ParametersType().(*TupleType)
	size := tuple.Size()
	if size.Min() == 0 && size.Max() == 0 {
		return ``
	}

	names := signature.ParameterNames()
	types := tuple.Types()
	limit := len(types)
	results := make([]string, 0, limit)
	for ix, t := range types {
		indicator := ``
		if size.Max() == math.MaxInt64 && ix == limit - 1 {
			// Last is a repeated_param.
			indicator = `*`
			if size.Min() == int64(len(names)) {
				indicator = `+`
			}
		} else if optional(ix, size.Max()) {
			indicator = `?`
			if ot, ok := t.(*OptionalType); ok {
				t = ot.ContainedType()
			}
		}
		results = append(results, fmt.Sprintf(`%s %s%s`, t.String(), names[ix], indicator))
	}

	block := signature.BlockType()
	if block != nil {
		if ob, ok := block.(*OptionalType); ok {
			block = ob.ContainedType()
		}
		results = append(results, fmt.Sprintf(`%s %s`, block.String(), signature.BlockName()))
	}
	return strings.Join(results, `, `)
}

func optional(index int, requiredCount int64) bool {
	count := int64(index + 1)
	return count > requiredCount
}
