package types

import (
	"io"

	. "github.com/puppetlabs/go-evaluator/eval"
	. "github.com/puppetlabs/go-evaluator/semver"
	. "github.com/puppetlabs/go-parser/parser"
)

type (
	TypeSetReference struct {
		name          string
		nameAuthority URI
		versionRange  *VersionRange
		typeSet       *TypeSetType
		annotations   *HashValue
	}

	TypeSetType struct {
		dcToCcMap          map[string]string
		name               string
		nameAuthority      URI
		pcoreURI           URI
		pcoreVersion       *Version
		version            *Version
		loader             Loader
		annotations        *HashValue
		initHashExpression Expression
	}
)

var typeSetType_DEFAULT = &TypeSetType{
	name:          `DefaultTypeSet`,
	nameAuthority: RUNTIME_NAME_AUTHORITY,
	pcoreURI:      PCORE_URI,
	pcoreVersion:  PCORE_VERSION,
	version:       NewVersion4(0, 0, 0, ``, ``),
}

func DefaultTypeSetType() *TypeSetType {
	return typeSetType_DEFAULT
}

func (t *TypeSetType) Accept(v Visitor, g Guard) {
	v(t)
	// TODO: Visit typeset members
}

func (t *TypeSetType) Default() PType {
	return typeSetType_DEFAULT
}

func (t *TypeSetType) Equals(other interface{}, guard Guard) bool {
	panic("implement me")
}

func (t *TypeSetType) String() string {
	panic("implement me")
}

func (t *TypeSetType) ToString(bld io.Writer, format FormatContext, g RDetect) {
	panic("implement me")
}

func (t *TypeSetType) Type() PType {
	panic("implement me")
}

func (t *TypeSetType) IsInstance(o PValue, g Guard) bool {
	panic("implement me")
}

func (t *TypeSetType) IsAssignable(ot PType, g Guard) bool {
	panic("implement me")
}

func (t *TypeSetType) Annotations() *HashValue {
	return t.annotations
}

func (t *TypeSetType) Name() string {
	return t.name
}

func (t *TypeSetType) Resolve(loader Loader) {
	panic("implement me")
}
