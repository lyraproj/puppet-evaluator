package resource

import (
	"fmt"
	"strings"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/impl"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-parser/issue"
	"github.com/puppetlabs/go-parser/parser"
	"github.com/puppetlabs/go-parser/validator"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

type (
	resourceEval struct {
		evaluator eval.Evaluator

		maxNode int64

		graph *simple.DirectedGraph

		nodes map[string]*node
	}

	// Evaluator is capable of evaluating resource expressions and resource operators. The
	// evaluator builds a graph which can be accessed by functions during evaluation.
	Evaluator interface {
		eval.Evaluator

		Edges(from Node) eval.IndexedValue

		Graph() graph.Graph

		HasNode(value eval.PValue) bool

		Node(value eval.PValue) (Node, bool)

		Nodes() eval.IndexedValue

		From(node Node) eval.IndexedValue
	}
)

// NodeName returns the string T[<title>] where T is the lower case name of a resourc type
// and <title> is the unique title of the instance that is referenced
func NodeName(value eval.PValue) (string, *issue.Reported) {
	switch value.(type) {
	case eval.PuppetObject:
		resource := value.(eval.PuppetObject)
		if title, ok := resource.Get(`title`); ok {
			return fmt.Sprintf(`%s[%s]`, strings.ToLower(resource.Type().Name()), title.String()), nil
		}
		return ``, eval.Error(EVAL_ILLEGAL_RESOURCE, issue.H{`value_type`: resource.Type().String()})
	case eval.ParameterizedType:
		pt := value.(eval.ParameterizedType)
		params := pt.Parameters()
		if len(params) == 1 {
			if p0, ok := params[0].(*types.StringValue); ok {
				return fmt.Sprintf(`%s[%s]`, strings.ToLower(pt.Name()), p0.String()), nil
			}
		}
	case *types.StringValue:
		if name, title, ok := SplitRef(value.String()); ok {
			return fmt.Sprintf(`%s[%s]`, strings.ToLower(name), title), nil
		}
		return ``, eval.Error(EVAL_ILLEGAL_RESOURCE_REFERENCE, issue.H{`str`: value.String()})
	}
	return ``, eval.Error(EVAL_ILLEGAL_RESOURCE_OR_REFERENCE, issue.H{`value_type`: value.Type().String()})
}

// SplitRef splits a reference in the form `<name> '[' <title> ']'` into a name and
// a title string and returns them.
// The method returns two empty strings and boolean false if the string cannot be
// parsed into a name and a title.
func SplitRef(ref string) (typeName, title string, ok bool) {
	end := len(ref) - 1
	if end >= 3 && ref[end] == ']' {
		titleStart := strings.IndexByte(ref, '[')
		if titleStart > 0 && titleStart+1 < end {
			return ref[:titleStart], ref[titleStart+1 : end], true
		}
	}
	return ``, ``, false
}

// NewEvaluator creates a new instance of the resource.Evaluator
func NewEvaluator(loader eval.DefiningLoader, logger eval.Logger) Evaluator {
	re := &resourceEval{}
	re.graph = simple.NewDirectedGraph()
	re.nodes = make(map[string]*node, 17)
	re.evaluator = impl.NewOverriddenEvaluator(loader, logger, re)
	return re
}

func (re *resourceEval) AddDefinitions(expression parser.Expression) {
	re.evaluator.AddDefinitions(expression)
}

func (re *resourceEval) ResolveDefinitions(c eval.EvalContext) {
	re.evaluator.ResolveDefinitions(c)
}

func (re *resourceEval) Evaluate(expression parser.Expression, scope eval.Scope, loader eval.Loader) (eval.PValue, *issue.Reported) {
	return re.evaluator.Evaluate(expression, scope, loader)
}

func (re *resourceEval) Logger() eval.Logger {
	return re.evaluator.Logger()
}

func (re *resourceEval) Graph() graph.Graph {
	return re.graph
}

func (re *resourceEval) HasNode(value eval.PValue) bool {
	ok := false
	if ref, err := NodeName(value); err == nil {
		_, ok = re.nodes[ref]
	}
	return ok
}

func (re *resourceEval) Eval(expr parser.Expression, c eval.EvalContext) eval.PValue {
	switch expr.(type) {
	case *parser.RelationshipExpression:
		return re.eval_RelationshipExpression(expr.(*parser.RelationshipExpression), c)
	case *parser.ResourceExpression:
		return re.eval_ResourceExpression(expr.(*parser.ResourceExpression), c)
	default:
		return re.evaluator.Eval(expr, c)
	}
}

func (re *resourceEval) eval_RelationshipExpression(expr *parser.RelationshipExpression, c eval.EvalContext) eval.PValue {
	lhs := re.Eval(expr.Lhs(), c)
	rhs := re.Eval(expr.Rhs(), c)
	switch expr.Operator() {
	case `->`:
		re.addEdge(&edge{re.node(lhs, expr.Lhs(), true), re.node(rhs, expr.Rhs(), true), false})
	case `~>`:
		re.addEdge(&edge{re.node(lhs, expr.Lhs(), true), re.node(rhs, expr.Rhs(), true), true})
	case `<-`:
		re.addEdge(&edge{re.node(rhs, expr.Rhs(), true), re.node(lhs, expr.Lhs(), true), false})
	default:
		re.addEdge(&edge{re.node(rhs, expr.Rhs(), true), re.node(lhs, expr.Lhs(), true), true})
	}
	return lhs
}

func (re *resourceEval) eval_ResourceExpression(expr *parser.ResourceExpression, c eval.EvalContext) eval.PValue {
	switch expr.Form() {
	case parser.REGULAR:
	default:
		panic(eval.Error2(expr, validator.VALIDATE_UNSUPPORTED_EXPRESSION, issue.H{`expression`: expr}))
	}
	result := make([]eval.PValue, len(expr.Bodies()))
	typeName := re.Eval(expr.TypeName(), c)

	// Load the actual resource type
	if ctor, ok := eval.Load(c.Loader(), eval.NewTypedName(eval.CONSTRUCTOR, typeName.String())); ok {
		for i, body := range expr.Bodies() {
			result[i] = re.newResources(ctor.(eval.Function), body.(*parser.ResourceBody), c)
		}
		return types.WrapArray(result).Flatten()
	}
	return eval.UNDEF
}

// Node returns the node for the given value
func (re *resourceEval) Node(value eval.PValue) (Node, bool) {
	n := re.node(value, nil, false)
	if n == nil {
		return nil, false
	}
	return n, true
}

// Nodes returns all resource nodes, sorted by file, line, pos
func (re *resourceEval) Nodes() eval.IndexedValue {
	return nodeList(re.graph.Nodes())
}

// FromNode returns all resource nodes extending from the given node, sorted by file, line, pos
func (re *resourceEval) From(node Node) eval.IndexedValue {
	return nodeList(re.graph.From(node.ID()))
}

// Edges returns all edges extending from the given node, sorted by file, line, pos of the
// node appointed by the edge
func (re *resourceEval) Edges(from Node) eval.IndexedValue {
	g := re.graph
	return re.From(from).Map(func(to eval.PValue) eval.PValue {
		return g.Edge(from.ID(), to.(Node).ID()).(Edge)
	})
}

func nodeList(nodes []graph.Node) eval.IndexedValue {
	rs := make([]eval.PValue, len(nodes))
	for i, n := range nodes {
		rs[i] = n.(eval.PValue)
	}
	return types.WrapArray(rs).Sort(func(a, b eval.PValue) bool {
		l1 := a.(Node).Location()
		l2 := b.(Node).Location()
		if l1.File() == l2.File() {
			ld := l1.Line() - l2.Line()
			if ld == 0 {
				return l1.Pos() < l2.Pos()
			}
			return ld < 0
		}
		return l1.File() < l2.File()
	})
}

func (re *resourceEval) node(value eval.PValue, location issue.Location, create bool) *node {
	var resource eval.PuppetObject
	if po, ok := value.(eval.PuppetObject); ok {
		resource = po
	}

	ref, err := NodeName(value)
	if err != nil {
		panic(err)
	}

	if node, ok := re.nodes[ref]; ok {
		if node.value == nil {
			node.value = resource // Resolves reference
			node.location = location
		} else if resource != nil && node.value != resource {
			panic(eval.Error(EVAL_DUPLICATE_RESOURCE, issue.H{`ref`: ref, `previous_location`: issue.LocationString(node.location)}))
		}
		return node
	}
	if create {
		node := &node{int64(len(re.nodes)), ref, resource, location}
		re.nodes[ref] = node
		re.graph.AddNode(node)
		return node
	}
	return nil
}

func (re *resourceEval) addEdge(edge *edge) {
	re.graph.SetEdge(edge)
}

func (re *resourceEval) newResources(ctor eval.Function, body *parser.ResourceBody, c eval.EvalContext) eval.PValue {
	// Turn the resource expression into an instantiation of an object
	bt := body.Title()
	if bta, ok := bt.(*parser.LiteralList); ok {
		rs := make([]eval.PValue, len(bta.Elements()))
		for i, e := range bta.Elements() {
			rs[i] = re.newResources2(ctor, e, body, c)
		}
		return types.WrapArray(rs)
	}
	return re.newResources2(ctor, bt, body, c)
}

func (re *resourceEval) newResources2(ctor eval.Function, title parser.Expression, body *parser.ResourceBody, c eval.EvalContext) eval.PValue {
	tes := re.Eval(title, c)
	if ta, ok := tes.(*types.ArrayValue); ok {
		return ta.Map(func(te eval.PValue) eval.PValue {
			return re.newResource(ctor, title, te, body, c)
		})
	}
	return re.newResource(ctor, title, tes, body, c)
}

func (re *resourceEval) newResource(ctor eval.Function, titleExpr parser.Expression, title eval.PValue, body *parser.ResourceBody, c eval.EvalContext) eval.PValue {
	entries := make([]*types.HashEntry, 0)
	entries = append(entries, types.WrapHashEntry2(`title`, title))
	fromHere := make([]*edge, 0)
	toHere := make([]*edge, 0)
	for _, op := range body.Operations() {
		if attr, ok := op.(*parser.AttributeOperation); ok {
			vexpr := attr.Value()
			v := re.Eval(vexpr, c)
			switch attr.Name() {
			case `before`:
				fromHere = append(fromHere, &edge{nil, re.node(v, vexpr, true), false})
			case `notify`:
				fromHere = append(fromHere, &edge{nil, re.node(v, vexpr, true), true})
			case `after`:
				toHere = append(toHere, &edge{re.node(v, vexpr, true), nil, false})
			case `subscribe`:
				toHere = append(toHere, &edge{re.node(v, vexpr, true), nil, true})
			default:
				entries = append(entries, types.WrapHashEntry2(attr.Name(), v))
			}
		} else {
			ops := op.(*parser.AttributesOperation)
			attrOps := re.Eval(ops.Expr(), c)
			if hash, hok := attrOps.(*types.HashValue); hok {
				entries = hash.AppendEntriesTo(entries)
			}
		}
	}
	obj := re.node(ctor.Call(c, nil, types.WrapHash(entries)), titleExpr, true)
	for _, edge := range fromHere {
		edge.from = obj
		re.addEdge(edge)
	}
	for _, edge := range toHere {
		edge.to = obj
		re.addEdge(edge)
	}
	return obj.value
}
