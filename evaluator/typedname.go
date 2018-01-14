package evaluator

import (
	"strings"
)

type (
	Namespace string

	TypedName interface {
		Equals(tn TypedName) bool

		IsQualified() bool

		MapKey() string

		String() string

		NameAuthority() URI

		Namespace() Namespace

		NameParts() []string

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
	CONSTRUCTOR = Namespace(`constructor`)
	TASK        = Namespace(`task`)
)

func NewTypedName(namespace Namespace, name string) TypedName {
	return NewTypedName2(namespace, name, RUNTIME_NAME_AUTHORITY)
}

func NewTypedName2(namespace Namespace, name string, nameAuthority URI) TypedName {
	tn := typedName{}

	parts := strings.Split(name, `::`)
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

func (t *typedName) Parent() TypedName {
	if !t.IsQualified() {
		return nil
	}
	parts := t.nameParts[1:]
	name := strings.Join(parts, `::`)
	compoundName := string(t.nameAuthority) + `/` + string(t.namespace) + `/` + name

	return &typedName{
		nameParts:     parts,
		namespace:     t.namespace,
		nameAuthority: t.nameAuthority,
		compoundName:  compoundName,
		canonicalName: strings.ToLower(compoundName)}
}

func (t *typedName) Equals(tn TypedName) bool {
	return t.canonicalName == tn.MapKey()
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
