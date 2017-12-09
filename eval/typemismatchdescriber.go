package eval

import (
	"fmt"
	. "github.com/puppetlabs/go-evaluator/evaluator"
	"strings"
	"reflect"
)

type (
	pathElement struct {
		key string
		pathType pathType
	}

	pathType string

	mismatch interface {
		canonicalPath() []*pathElement
		path() []*pathElement
		pathString() string
		setPath(path []*pathElement)
		text() string
	}

	expectedActualMismatch interface {
		mismatch
		actual() PType
		expected() PType
		setExpected(expected PType)
	}

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
)

const (
	subject = pathType(``)
	entry  = pathType(`entry`)
	entryKey = pathType(`entryKey`)
	_parameter = pathType(`parameter`)
	_return = pathType(`return`)
	block = pathType(`block`)
	index = pathType(`index`)
	variant = pathType(`variant`)
	signature = pathType(`singature`)
)

func (p pathType) format(key string) string {
	if p == subject || p == signature {
		return key
	}
	return fmt.Sprintf("%s '%s'", p, key)
}

func (pe *pathElement) String() string {
	return pe.pathType.format(pe.key)
}

func copyMismatch(m mismatch) mismatch {
	orig := reflect.Indirect(reflect.ValueOf(m))
	copy := reflect.New(orig.Type())
	copy.Set(orig)
	return copy.Interface().(mismatch)
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
	if eam, ok := m.(expectedActualMismatch); ok {
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
		if p.pathType != variant {
			result = append(result, p)
		}
	}
	return result
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

func (*unexpectedBlock) text() string {
	return `does not expect a block`
}

func (*missingRequiredBlock) text() string {
	return `expects a block`
}

func (m *missingKey) text() string {
	return fmt.Sprintf(`expects a value for key '%s'`, m.key)
}

func (m *missingParameter) text() string {
	return fmt.Sprintf(`expects a value for parameter '%s'`, m.key)
}

func (m *extraneousKey) text() string {
	return fmt.Sprintf(`unrecognized key '%s'`, m.key)
}

func (m *invalidParameter) text() string {
	return fmt.Sprintf(`has no parameter named '%s'`, m.key)
}

func (m *unresolvedTypeReference) text() string {
	return fmt.Sprintf(`references an unresolved type '%s'`, m.key)
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
/*
func (tm *typeMismatch) text() string {
	e := tm.expectedType
	a := tm.actualType
	multi := false
	as := ``
	es := ``
	if vt, ok := e.(*types.VariantType); ok {
		el := vt.Types()
		var els []string
		if reportDetailed(el, a) {
			as = detailedToActualToS(el, a)
			els = MapTypes(el, func(t PType) PValue { return t.String() })
		}
	}
}
*/