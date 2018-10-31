package eval

import (
	"github.com/puppetlabs/go-issues/issue"
)

type Namespace string

const (
	ACTIVITY    = Namespace(`activity`)
	TYPE        = Namespace(`type`)
	FUNCTION    = Namespace(`function`)
	PLAN        = Namespace(`plan`)
	ALLOCATOR   = Namespace(`allocator`)
	CONSTRUCTOR = Namespace(`constructor`)
	TASK        = Namespace(`task`)
)

type TypedName interface {
	PuppetObject
	issue.Named

	IsParent(n TypedName) bool

	IsQualified() bool

	MapKey() string

	Authority() URI

	Namespace() Namespace

	Parts() []string

	// PartsList returns the parts as a List
	PartsList() List

	// Child returns the typed name with its leading segment stripped off, e.g.
	// A::B::C returns B::C
	Child() TypedName

	// Parent returns the typed name with its final segment stripped off, e.g.
	// A::B::C returns A::B
	Parent() TypedName

	RelativeTo(parent TypedName) (TypedName, bool)
}

var NewTypedName func(namespace Namespace, name string) TypedName
var NewTypedName2 func(namespace Namespace, name string, name_authority URI) TypedName
