package eval

import (
	"strings"
	"bytes"
	"github.com/puppetlabs/go-issues/issue"
)

type (
	Namespace string

	TypedName interface {
		issue.Named

		IsQualified() bool

		MapKey() string

		String() string

		NameAuthority() URI

		Namespace() Namespace

		NameParts() []string

		// Child returns the typed name with its leading segment stripped off, e.g.
		// A::B::C returns B::C
		Child() TypedName

		// Parent returns the typed name with its final segment stripped off, e.g.
		// A::B::C returns A::B
		Parent() TypedName
	}

	typedName struct {
		namespace     Namespace
		nameAuthority URI
		compoundName  string
		canonicalName string
		nameParts     []string
	}
)

const (
	TYPE        = Namespace(`type`)
	FUNCTION    = Namespace(`function`)
	PLAN        = Namespace(`plan`)
	ALLOCATOR   = Namespace(`allocator`)
	CONSTRUCTOR = Namespace(`constructor`)
	TASK        = Namespace(`task`)
)

func NewTypedName(namespace Namespace, name string) TypedName {
	return NewTypedName2(namespace, name, RUNTIME_NAME_AUTHORITY)
}

func NewTypedName2(namespace Namespace, name string, nameAuthority URI) TypedName {
	tn := typedName{}

	parts := strings.Split(strings.ToLower(name), `::`)
	if len(parts) > 0 && parts[0] == `` {
		parts = parts[1:]
		name = name[2:]
	}
	tn.nameParts = parts
	tn.namespace = namespace
	tn.nameAuthority = nameAuthority
	tn.compoundName = string(nameAuthority) + `/` + string(namespace) + `/` + name
	tn.canonicalName = strings.ToLower(tn.compoundName)
	return &tn
}

func (t *typedName) Child() TypedName {
	if !t.IsQualified() {
		return nil
	}

	stripLen := len(t.nameParts[0]) + 2
	parts := t.nameParts[1:]
	b := bytes.NewBufferString(``)
	b.Grow(len(t.compoundName) - stripLen)
	b.WriteString(string(t.nameAuthority))
	b.WriteByte('/')
	b.WriteString(string(t.namespace))
	b.WriteByte('/')
	b.WriteString(parts[0])
	for idx := 1; idx < len(parts); idx++ {
		b.WriteString(`::`)
		b.WriteString(parts[idx])
	}
	compoundName := b.String()

	return &typedName{
		nameParts:     parts,
		namespace:     t.namespace,
		nameAuthority: t.nameAuthority,
		compoundName:  compoundName,
		canonicalName: strings.ToLower(compoundName)}
}

func (t *typedName) Parent() TypedName {
	if !t.IsQualified() {
		return nil
	}
	lx := strings.LastIndex(t.compoundName, `::`)
	return &typedName{
		nameParts:     t.nameParts[:len(t.nameParts)-1],
		namespace:     t.namespace,
		nameAuthority: t.nameAuthority,
		compoundName:  t.compoundName[:lx],
		canonicalName: t.canonicalName[:lx]}
}

func (t *typedName) Equals(other interface{}, g Guard) bool {
	if tn, ok := other.(TypedName); ok {
		return t.canonicalName == tn.MapKey()
	}
	return false
}

func (t *typedName) Name() string {
	cn := t.compoundName
	return cn[strings.LastIndex(cn, `/`)+1:]
}

func (t *typedName) IsQualified() bool {
	return len(t.nameParts) > 1
}

func (t *typedName) MapKey() string {
	return t.canonicalName
}

func (t *typedName) NameParts() []string {
	return t.nameParts
}

func (t *typedName) String() string {
	return t.compoundName
}

func (t *typedName) Namespace() Namespace {
	return t.namespace
}

func (t *typedName) NameAuthority() URI {
	return t.nameAuthority
}
