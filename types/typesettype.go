package types

import (
	"io"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/semver"
	"github.com/puppetlabs/go-parser/parser"
)

type (
	TypeSetReference struct {
		name          string
		nameAuthority eval.URI
		versionRange  *semver.VersionRange
		typeSet       *TypeSetType
		annotations   *HashValue
	}

	TypeSetType struct {
		dcToCcMap          map[string]string
		name               string
		nameAuthority      eval.URI
		pcoreURI           eval.URI
		pcoreVersion       *semver.Version
		version            *semver.Version
		loader             eval.Loader
		annotations        *HashValue
		initHashExpression parser.Expression
	}
)

var typeSetType_DEFAULT = &TypeSetType{
	name:          `DefaultTypeSet`,
	nameAuthority: eval.RUNTIME_NAME_AUTHORITY,
	pcoreURI:      eval.PCORE_URI,
	pcoreVersion:  eval.PCORE_VERSION,
	version:       semver.NewVersion4(0, 0, 0, ``, ``),
}

func DefaultTypeSetType() *TypeSetType {
	return typeSetType_DEFAULT
}

func (t *TypeSetType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
	// TODO: Visit typeset members
}

func (t *TypeSetType) Default() eval.PType {
	return typeSetType_DEFAULT
}

func (t *TypeSetType) Equals(other interface{}, guard eval.Guard) bool {
	panic("implement me")
}

func (t *TypeSetType) String() string {
	panic("implement me")
}

func (t *TypeSetType) ToString(bld io.Writer, format eval.FormatContext, g eval.RDetect) {
	panic("implement me")
}

func (t *TypeSetType) Type() eval.PType {
	panic("implement me")
}

func (t *TypeSetType) IsInstance(o eval.PValue, g eval.Guard) bool {
	panic("implement me")
}

func (t *TypeSetType) IsAssignable(ot eval.PType, g eval.Guard) bool {
	panic("implement me")
}

func (t *TypeSetType) Annotations() *HashValue {
	return t.annotations
}

func (t *TypeSetType) Name() string {
	return t.name
}

func (t *TypeSetType) Resolve(loader eval.Loader) {
	panic("implement me")
}
