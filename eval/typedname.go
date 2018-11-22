package eval

import (
	"github.com/puppetlabs/go-issues/issue"
)

type Namespace string

// Identifier TypedName namespaces. Used by a service to identify what the type of entity a loader
// will look for.

// NsType denotes a type in the Puppet type system
const  NsType        = Namespace(`type`)

// NsFunction denotes a callable function
const  NsFunction    = Namespace(`function`)

// NsInterface denotes an entity that must have an "interface" property that appoints
// an object type which in turn contains a declaration of the methods that the interface
// implements.
const NsInterface = Namespace(`interface`)

// NsActivity denotes an entity that can participate in a workflow. The entity must
// declare input and output parameters. An activity of type "action" may also be an interface
// in which case it must have an "interface" property
const NsActivity = Namespace(`activity`)

// ServiceId TypedName namespaces. Used by the Loader to determine the right type
// of RPC mechanism to use when communicating with the service.

// NsHandler denotes a handler for a state in a workflow
const NsHandler = Namespace(`handler`)

// NsService denotes a remote service
const NsService = Namespace(`service`)

// Here in case of future Bolt integration with the Evaluator
const  NsPlan        = Namespace(`plan`)
const  NsTask        = Namespace(`task`)

// For internal use only

// NsAllocator returns a function capable of allocating an instance of an object
// without initializing its content
const NsAllocator   = Namespace(`allocator`)

// NsConstructor denotes a function that both allocates an initializes an object based
// on parameter values
const NsConstructor = Namespace(`constructor`)


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

// TypedNameFromMapKey recreates a TypedName from a given MapKey that was produced by a TypedName
var TypedNameFromMapKey func(mapKey string) TypedName

