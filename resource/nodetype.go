package resource

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-parser/issue"
	"io"
	"github.com/puppetlabs/go-evaluator/types"
	"gonum.org/v1/gonum/graph"
	"github.com/puppetlabs/go-parser/parser"
	"sync"
)

var Node_Type eval.ObjectType

func init() {
	Node_Type = eval.NewObjectType(`ResourceNode`, `{
	attributes => {
		id => Integer,
	  value => Any,
		resources => Hash[String,Resource]
	}
}`)
}

type(
	// A Node represents a Future value. It can be evaluated once all edges
	// pointing to it has been evaluated
	Node interface {
		graph.Node
		eval.PValue

		Location() issue.Location

		// Resolved returns true if all promises has been fulfilled for this
		// value. In essence, that means that this node is not appointed by
		// edges from any unresolved nodes and that the contained expression has
		// been evaluated.
		Resolved() bool

		// Resource returns the resource that corresponds to the given ref
		Resource(ref string) (eval.PuppetObject, bool)

		// Resource returns the resources kept by this node that corresponds to the given ref
		Resources(c eval.Context) eval.KeyedValue

		// Value returns the value of this node. This is a potentially blocking operation. In
		// all nodes appointing this node must be resolved and the contained expression must
		// be evaluated before the value can be produced.
		Value(c eval.Context) eval.PValue
	}

	// node represents a PuppetObject in the graph with a unique ID
	node struct {
		id int64
		resolveLock sync.Mutex
		lock sync.RWMutex

		// Broadcast channel. Will be closed when node is resolved
		resolved chan bool

		value eval.PValue
		resources map[string]*handle

		// resource expression or a type reference expression
		expression parser.Expression
	}
)

func (rn *node) Equals(other interface{}, guard eval.Guard) bool {
	on, ok := other.(*node)
	return ok && rn.id == on.id
}

func (rn *node) Get(c eval.Context, key string) (value eval.PValue, ok bool) {
	switch key {
	case `id`:
		return types.WrapInteger(rn.id), true
	case `resources`:
		return rn.Resources(c), true
	case `value`:
		return rn.Value(c), true
	}
	return eval.UNDEF, false
}

func (rn *node) InitHash() eval.KeyedValue {
	<-rn.resolved
	rn.lock.RLock()
	hash := map[string]eval.PValue {
		`id`: types.WrapInteger(rn.id),
	}
	hash[`value`] = rn.value
	rn.lock.RUnlock()
	return types.WrapHash3(hash)
}

func (rn *node) Location() issue.Location {
	return rn.expression
}

func (rn *node) Resolved() bool {
	rn.lock.RLock()
	resolved := rn.value != nil
	rn.lock.RUnlock()
	return resolved
}

func (rn *node) Resources(c eval.Context) eval.KeyedValue {
	<-rn.resolved
	rn.lock.RLock()
	entries := make([]*types.HashEntry, 0, len(rn.resources))
	for k, r := range rn.resources {
		entries = append(entries, types.WrapHashEntry2(k, r))
	}
	rn.lock.RUnlock()
	sortByEntriesLocation(entries)
	return types.WrapHash(entries)
}

func (rn *node) String() string {
	return eval.ToString2(rn, types.NONE)
}

func (rn *node) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	types.ObjectToString(rn, s, b, g)
}

func (rn *node) Type() eval.PType {
	return Node_Type
}

func (rn *node) Value(c eval.Context) eval.PValue {
	<-rn.resolved
	rn.lock.RLock()
	value := rn.value
	rn.lock.RUnlock()
	if edge, ok := value.(Edge); ok {
		value = edge.To().(Node).Value(c)
	}
	return value
}

// Creates an unevaluated node, aware of all resources that uses static titles
func newNode(c eval.Context, expression parser.Expression, value eval.PValue) *node {
	var handles map[string]*handle
	if expression == nil {
		// Root node
		handles = map[string]*handle{}
	} else {
		refs := findResources(c, expression)
		handles = make(map[string]*handle, len(refs))
		for _, r := range refs {
			handles[r] = &handle{}
		}
	}
	g := GetGraph(c).(graph.DirectedBuilder)
	node := g.NewNode().(*node)
	node.expression = expression
	node.resources = handles
	node.value = value
	g.AddNode(node)
	return node
}


func (rn* node) Resource(ref string) (eval.PuppetObject, bool) {
	rn.lock.RLock()
	if h, ok := rn.resources[ref]; ok {
		rn.lock.RUnlock()
		return h, true
	}
	rn.lock.RUnlock()
	return nil, false
}

func (rn *node) ID() int64 {
	return rn.id
}

func (rn *node) evaluate(c eval.Context) (errs []*issue.Reported) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(*issue.Reported); ok {
				errs = []*issue.Reported{err}
			} else {
				panic(r)
			}
		}
	}()

	error := func() *issue.Reported {
		rn.resolveLock.Lock()
		defer func() {
			// Closing the resolved channel will notify everyone that awaits its value, i.e.
			// everyone that waited for this node to be resolved.
			close(rn.resolved)
			rn.resolveLock.Unlock()
		}()

		if rn.value != nil {
			return nil
		}

		// Ensure that all nodes that has an edge to this node have been
		// fully resolved.
		rn.waitForEdgesTo(c)

		value, err := c.Evaluator().(*resourceEval).evaluateNodeExpression(c, rn)
		if err == nil {
			rn.update(c, value, getResources(c))
			return nil
		}

		rn.value = types.NewError(c, types.WrapString(err.String()), eval.UNDEF, types.WrapString(string(err.Code())))
		return err
	}()

	if error != nil {
		errs = []*issue.Reported{error}
		return
	}

	resources := rn.Resources(c)
	if resources.Len() > 0 {
		// TODO: Block on apply of all resources here.
	}

	scheduleNodes(c, GetGraph(c).FromNode(rn))
	return
}

func (rn *node) update(c eval.Context, value eval.PValue, resources map[string]*handle) {
	rn.value = value
	for ref, rh := range resources {
		if h, ok := rn.resources[ref]; ok {
			if h.value != nil {
				panic(eval.Error(c, EVAL_DUPLICATE_RESOURCE, issue.H{`ref`: ref, `previous_location`: issue.LocationString(h.location)}))
			}
			h.location = rh.location
			h.value = rh.value
		} else {
			// New declaration of resource that wasn't previously known. This may happen when the resource title
			// could not be determined statically. It's not a problem, it just means that this resource could not
			// be referenced until at this point.
			rn.resources[ref] = rh
		}
	}
}

// Search this node and all parent nodes for the given resource
func (rn *node) findResource(c eval.Context, ref string) (*node, bool) {
	if _, ok := rn.Resource(ref); ok {
		return rn, true
	}
	for _, n := range GetGraph(c).To(rn.ID()) {
		if found, ok := n.(*node).findResource(c, ref); ok {
			return found, true
		}
	}
	return nil, false
}

// Search node children for the given resource. The receiver is not included in
// this search.
func (rn *node) findChildWithResource(c eval.Context, ref string) (*node, bool) {
	for _, n := range GetGraph(c).From(rn.ID()) {
		cn := n.(*node)
		if _, ok := cn.Resource(ref); ok {
			return rn, true
		}
		if found, ok := cn.findChildWithResource(c, ref); ok {
			return found, true
		}
	}
	return nil, false
}

// Ensure that all nodes that has an edge to this node have been
// fully resolved.
func (rn *node) waitForEdgesTo(c eval.Context) {
	g := GetGraph(c)
	parents := g.To(rn.ID())
	for {
		for _, before := range parents {
			<-before.(*node).resolved
		}

		// A new chech must be made if the list of nodes have changed
		parentsNow := g.To(rn.ID())
		if sameNodes(parents, parentsNow) {
			return
		}
		parents = parentsNow
	}
}

func sameNodes(a, b []graph.Node) bool {
	if len(a) != len(b) {
		return false
	}
	for _, ap := range a {
		found := false
		for _, bp := range b {
			if ap.ID() == bp.ID() {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
