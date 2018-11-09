package resource

import (
	"io"
	"sync"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-parser/parser"
	"gonum.org/v1/gonum/graph"
)

var Node_Type eval.ObjectType

func init() {
	Node_Type = eval.NewObjectType(`ResourceNode`, `{
	attributes => {
		id => Integer,
	  value => RichData,
		resources => Hash[String,Resource]
	}
}`)
}

type (
	// A Node represents a Future value. It can be evaluated once all edges
	// pointing to it has been evaluated
	Node interface {
		graph.Node
		eval.Value

		// Error returns an error if the evaluation of the node was unsuccessful, otherwise
		// nil is returned
		Error() issue.Reported

		Location() issue.Location

		// Context used when this node was created (defines scope)
		Context() eval.Context

		// Resolved returns true if all promises has been fulfilled for this
		// value. In essence, that means that this node is not appointed by
		// edges from any unresolved nodes and that the contained expression has
		// been evaluated.
		Resolved() bool

		// Resource returns the resource that corresponds to the given ref
		Resource(ref string) (eval.PuppetObject, bool)

		// Resource returns the resources kept by this node that corresponds to the given ref
		Resources() eval.OrderedMap

		// Value returns the result of evaluating the node expression. This is a potentially blocking
		// operation. In all nodes appointing this node must be resolved and the contained expression must
		// be evaluated before the result can be produced.
		Value() eval.Value
	}

	// node represents a PuppetObject in the graph with a unique ID
	node struct {
		id          int64
		resolveLock sync.Mutex
		lock        sync.RWMutex

		// Broadcast channel. Will be closed when node is resolved
		resolved chan bool

		value     eval.Value

		resources map[string]*handle

		// invocable value (mutualy exclusive to the expression
		lambda eval.InvocableValue

		// resource expression or a type reference expression
		expression parser.Expression

		// parameters to lambda (only valid when expression is a lambda)
		parameters []eval.Value

		// Set in case node evaluation ended in error
		error issue.Reported

		results []eval.Value

		context eval.Context
	}
)

func (rn *node) Context() eval.Context {
	return rn.context
}

func (rn *node) Equals(other interface{}, guard eval.Guard) bool {
	on, ok := other.(*node)
	return ok && rn.id == on.id
}

func (rn *node) Get(key string) (value eval.Value, ok bool) {
	switch key {
	case `id`:
		return types.WrapInteger(rn.id), true
	case `resources`:
		return rn.Resources(), true
	case `value`:
		return rn.Value(), true
	}
	return eval.UNDEF, false
}

func (rn *node) InitHash() eval.OrderedMap {
	<-rn.resolved
	rn.lock.RLock()
	hash := map[string]eval.Value{
		`id`: types.WrapInteger(rn.id),
	}
	hash[`value`] = rn.value
	rn.lock.RUnlock()
	return types.WrapHashSorted(hash)
}

func (rn *node) Error() issue.Reported {
	return rn.error
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

func (rn *node) Resources() eval.OrderedMap {
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

func (rn *node) PType() eval.Type {
	return Node_Type
}

func (rn *node) Value() eval.Value {
	<-rn.resolved
	rn.lock.RLock()
	value := rn.value
	rn.lock.RUnlock()
	if edge, ok := value.(Edge); ok {
		value = edge.To().(Node).Value()
	} else if maybeEdges, ok := value.(*types.ArrayValue); ok && maybeEdges.All(func(e eval.Value) bool {
		_, ok := e.(Edge)
		return ok
	}) {
		toUnique := make(map[int64]bool, maybeEdges.Len())
		toValues := make([]eval.Value, 0)
		maybeEdges.Each(func(e eval.Value) {
			n := e.(Edge).To()
			if !toUnique[n.ID()] {
				toUnique[n.ID()] = true
				toValues = append(toValues, n.(Node).Value())
			}
		})
		if len(toValues) == 1 {
			value = toValues[0]
		} else {
			value = types.WrapArray(toValues)
		}
	}
	return value
}

func appendResults(results []eval.Value, nv eval.Value) []eval.Value {
	switch nv.(type) {
	case ResultSet:
		results = nv.(ResultSet).Results().AppendTo(results)
	case Result:
		results = append(results, nv)
	case *types.ArrayValue:
		nv.(eval.List).Each(func(e eval.Value) {
			results = appendResults(results, e)
		})
	}
	return results
}

// Creates an unevaluated node, aware of all resources that uses static titles
func newNode(c eval.Context, expression parser.Expression, parameters ...eval.Value) *node {
	g := GetGraph(c).(graph.DirectedBuilder)
	node := g.NewNode().(*node)
	node.lambda = nil
	node.expression = expression
	node.parameters = parameters
	node.resources = map[string]*handle{}
	node.value = nil
	node.context = c.Fork()
	g.AddNode(node)
	return node
}

// Creates an unevaluated node, aware of all resources that uses static titles
func newNode2(c eval.Context, lambda eval.InvocableValue) *node {
	g := GetGraph(c).(graph.DirectedBuilder)
	node := g.NewNode().(*node)
	node.lambda = lambda
	node.expression = &parser.Nop{}
	node.parameters = []eval.Value{}
	node.resources = map[string]*handle{}
	node.value = nil
	node.context = c.Fork()
	g.AddNode(node)
	return node
}

func (rn *node) Resource(ref string) (eval.PuppetObject, bool) {
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

func (rn *node) evaluate() {
	c := rn.context
	done := func() bool {
		rn.resolveLock.Lock()
		if rn.value != nil {
			rn.resolveLock.Unlock()
			return true
		}

		defer func() {
			// Closing the resolved channel will notify everyone that awaits its value, i.e.
			// everyone that waited for this node to be resolved.
			close(rn.resolved)
			rn.resolveLock.Unlock()
		}()

		// Ensure that all nodes that has an edge to this node have been
		// fully resolved.
		c.Scope().Set(`pnr`, rn.waitForEdgesTo(c))
		value, err := c.Evaluator().(*resourceEval).evaluateNodeExpression(c, rn)
		if err == nil {
			rn.update(c, value, getResources(c))
			return false
		}

		rn.value = NewErrorResult(types.WrapInteger(rn.ID()), eval.ErrorFromReported(c, err))
		rn.error = err
		return true
	}()

	if !done {
		rn.apply(c)
	}
}

func (rn *node) apply(c eval.Context) {
	resources := rn.Resources()
	rcount := resources.Len()
	results := make([]eval.Value, 0, rcount+1)
	if _, ok := rn.expression.(*parser.ResourceExpression); !ok {
		results = appendResults(results, rn.value)
	}
	if rcount > 0 {
		rs := make([]eval.PuppetObject, 0, rcount)
		resources.EachValue(func(r eval.Value) {
			h := r.(*handle)
			if h.value != nil {
				rs = append(rs, h.value)
			}
		})

		applyResults, err := getApplyFunction(c)(c, rs)
		if err != nil {
			ir, ok := err.(issue.Reported)
			if !ok {
				ir = eval.Error(eval.EVAL_FAILURE, issue.H{`message`: err.Error()})
			}
			results = append(results, NewErrorResult(types.WrapInteger(rn.ID()), eval.ErrorFromReported(c, ir)))
			rn.error = ir
		} else {
			if len(applyResults) != len(rs) {
				panic(eval.Error(EVAL_APPLY_FUNCTION_SIZE_MISMATCH, issue.H{`expected`: len(rs), `actual`: len(applyResults)}))
			}
			for ix, ar := range applyResults {
				r := rs[ix]
				if err, ok := ar.(eval.ErrorObject); ok {
					results = append(results, NewErrorResult(types.WrapString(Reference(r)), err))
				} else {
					if ar == nil {
						panic(eval.Error(EVAL_APPLY_FUNCTION_NIL_RETURN, issue.NO_ARGS))
					}
					rh, _ := resources.Get4(Reference(r))
					if arp, ok := ar.(eval.PuppetObject); ok {
						// Update handle
						rh.(*handle).Replace(arp)
						results = append(results, NewResult(types.WrapString(Reference(r)), arp, ``))
					} else {
						panic(eval.Error(EVAL_APPLY_FUNCTION_INVALID_RETURN, issue.H{`value`: ar}))
					}
				}
			}
		}
	}
	rn.results = results

	scheduleNodes(c, GetGraph(c).FromNode(rn))
}

func (rn *node) update(c eval.Context, value eval.Value, resources map[string]*handle) {
	rn.value = value
	for ref, rh := range resources {
		if h, ok := rn.resources[ref]; ok {
			if h.value != nil {
				panic(eval.Error(EVAL_DUPLICATE_RESOURCE, issue.H{`ref`: ref, `previous_location`: issue.LocationString(h.location)}))
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
	ti := GetGraph(c).To(rn.ID())
	for ti.Next() {
		if found, ok := ti.Node().(*node).findResource(c, ref); ok {
			return found, true
		}
	}
	return nil, false
}

// Search node children for the given resource. The receiver is not included in
// this search.
func (rn *node) findChildWithResource(c eval.Context, ref string) (*node, bool) {
	fi := GetGraph(c).From(rn.ID())
	for fi.Next() {
		cn := fi.Node().(*node)
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
func (rn *node) waitForEdgesTo(c eval.Context) ResultSet {
	g := GetGraph(c)
	parents := nodeList(g.To(rn.ID()))
	for {
		parents.Each(func(n eval.Value) { <-n.(*node).resolved })

		// A new chech must be made if the list of nodes have changed
		parentsNow := nodeList(g.To(rn.ID()))
		if sameNodes(parents, parentsNow) {
			results := make([]eval.Value, 0)
			parents.Each(func(n eval.Value) { results = append(results, n.(*node).results...) })
			return NewResultSet(types.WrapArray(results))
		}
		parents = parentsNow
	}
}

func sameNodes(a, b eval.List) bool {
	if a.Len() != b.Len() {
		return false
	}
	return a.All(func(e eval.Value) bool {
		ap := e.(graph.Node)
		return b.Any(func(e eval.Value) bool { return ap.ID() == e.(graph.Node).ID() })
	})
}
