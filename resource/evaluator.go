package resource

import (
	"fmt"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/impl"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-parser/parser"
	"github.com/puppetlabs/go-parser/validator"
	"gonum.org/v1/gonum/graph"
	"log"
)

type resourceEval struct {
	evaluator eval.Evaluator
}

// Evaluator is capable of evaluating resource expressions and resource operators. The
// evaluator builds a graph which can be accessed by functions during evaluation.
type Evaluator interface {
	eval.Evaluator
}

// NewEvaluator creates a new instance of the resource.Evaluator
func NewEvaluator(logger eval.Logger) Evaluator {
	re := &resourceEval{}
	re.evaluator = impl.NewOverriddenEvaluator(logger, re)
	return re
}

func defaultApplyFunc(c eval.Context, resources []eval.PuppetObject) ([]eval.PuppetObject, error) {
	for _, h := range resources {
		text := types.WrapString(fmt.Sprintf("Applying %s", Reference(c, h)))
		log.Println(text)
		c.Logger().Log(eval.NOTICE, text)
	}
	return resources, nil
}

func (re *resourceEval) Evaluate(c eval.Context, expression parser.Expression) (value eval.PValue, err issue.Reported) {
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			if err, ok = r.(issue.Reported); !ok {
				panic(r)
			}
		}
	}()

	// Create the node resolution workers
	nodeJobs := make(chan *nodeJob, 50)
	done := make(chan bool)
	for w := 1; w <= 5; w++ {
		go nodeWorker(w, nodeJobs, done)
	}

	// Add a shared map. Must be considered immutable from this point on as there will
	// be concurrent reads
	var applyFunction ApplyFunction
	if af, ok := c.Get(APPLY_FUNCTION); ok {
		applyFunction = af.(ApplyFunction)
	} else {
		applyFunction = defaultApplyFunc
	}

	c.Set(SHARED_MAP, map[string]interface{}{
		NODE_GRAPH:     NewConcurrentGraph(),
		NODE_JOBS:      nodeJobs,
		JOB_COUNTER:    &jobCounter{0},
		APPLY_FUNCTION: applyFunction,
	})

	topNode := newNode(c, expression)
	scheduleNodes(c, types.SingletonArray(topNode))

	<-done
	errors := []issue.Reported{}
	for _, n := range GetGraph(c).Nodes() {
		if err := n.(Node).Error(); err != nil {
			errors = append(errors, err)
		}
	}
	switch len(errors) {
	case 0:
		return topNode.Value(), nil
	case 1:
		return eval.UNDEF, errors[0]
	default:
		return eval.UNDEF, eval.Error(c, eval.EVAL_MULTI_ERROR, issue.H{`errors`: errors})
	}
}

func (re *resourceEval) evaluateNodeExpression(c eval.Context, rn *node) (eval.PValue, issue.Reported) {
	setCurrentNode(c, rn)
	g := GetGraph(c)
	extEdges := g.From(rn.ID())
	setExternalEdgesTo(c, extEdges)
	setResources(c, map[string]*handle{})
	value, err := re.evaluator.Evaluate(c, rn.expression)
	if err != nil {
		return nil, err
	}

	if len(extEdges) < len(g.From(rn.ID())) {
		// Original externa edges are no longer needed since they now describe paths
		// that are reached using children
		r := g.(graph.EdgeRemover)
		for _, en := range extEdges {
			r.RemoveEdge(rn.ID(), en.ID())
		}
	}
	return value, nil
}

func (re *resourceEval) Logger() eval.Logger {
	return re.evaluator.Logger()
}

func (re *resourceEval) Eval(expr parser.Expression, c eval.Context) eval.PValue {
	switch expr.(type) {
	case *parser.RelationshipExpression:
		return re.eval_RelationshipExpression(expr.(*parser.RelationshipExpression), c)
	case *parser.ResourceExpression:
		return re.eval_ResourceExpression(expr.(*parser.ResourceExpression), c)
	default:
		return re.evaluator.Eval(expr, c)
	}
}

func (re *resourceEval) eval_RelationshipExpression(expr *parser.RelationshipExpression, c eval.Context) eval.PValue {
	edges := []Edge{}
	switch expr.Operator() {
	case `->`:
		edges = createEdges(c, expr.Lhs(), expr.Rhs(), false, edges)
	case `~>`:
		edges = createEdges(c, expr.Lhs(), expr.Rhs(), true, edges)
	case `<-`:
		edges = createEdges(c, expr.Rhs(), expr.Lhs(), false, edges)
	default:
		edges = createEdges(c, expr.Rhs(), expr.Lhs(), true, edges)
	}
	cn := getCurrentNode(c)
	g := GetGraph(c).(graph.DirectedBuilder)

	extEdges := getExternalEdgesTo(c)
	for _, e := range edges {
		for _, cnTo := range extEdges {
			// RHS of edge must evaluate before any edges external to the current node evaluates
			exEdge := g.Edge(cn.ID(), cnTo.ID()).(Edge)
			g.SetEdge(newEdge(e.To().(Node), cnTo.(Node), exEdge.Subscribe()))
		}

		// Create edge from current to LHS of edge
		g.SetEdge(newEdge(cn, e.From().(Node), false))
		g.SetEdge(e)
	}
	if len(edges) == 1 {
		return edges[0]
	}
	evs := make([]eval.PValue, len(edges))
	for i, e := range edges {
		evs[i] = e
	}
	return types.WrapArray(evs)
}

func createEdges(c eval.Context, lhs, rhs parser.Expression, subscribe bool, edges []Edge) []Edge {
	rhsNodes := createNodes(c, rhs, []Node{})
	for _, lhsNode := range createNodes(c, lhs, []Node{}) {
		for _, rhsNode := range rhsNodes {
			edges = append(edges, newEdge(lhsNode, rhsNode, subscribe))
		}
	}
	return edges
}

func createNodes(c eval.Context, expr parser.Expression, nodes []Node) []Node {
	if exprArr, ok := expr.(*parser.LiteralList); ok {
		for _, el := range exprArr.Elements() {
			nodes = createNodes(c, el, nodes)
		}
	} else {
		nodes = append(nodes, newNode(c, expr))
	}
	return nodes
}

func (re *resourceEval) eval_ResourceExpression(expr *parser.ResourceExpression, c eval.Context) eval.PValue {
	switch expr.Form() {
	case parser.REGULAR:
	default:
		panic(eval.Error2(expr, validator.VALIDATE_UNSUPPORTED_EXPRESSION, issue.H{`expression`: expr}))
	}
	result := make([]eval.PValue, len(expr.Bodies()))
	typeName := re.Eval(expr.TypeName(), c)

	// Load the actual resource type
	if ctor, ok := eval.Load(c, eval.NewTypedName(eval.CONSTRUCTOR, typeName.String())); ok {
		for i, body := range expr.Bodies() {
			result[i] = re.newResources(ctor.(eval.Function), body.(*parser.ResourceBody), c)
		}
		return types.WrapArray(result).Flatten()
	}

	// The resource type is unknown when no constructor for it is found
	panic(eval.Error2(expr, EVAL_UNKNOWN_RESOURCE_TYPE, issue.H{`res_type`: typeName.String()}))
}

func (re *resourceEval) newResources(ctor eval.Function, body *parser.ResourceBody, c eval.Context) eval.PValue {
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

func (re *resourceEval) newResources2(ctor eval.Function, title parser.Expression, body *parser.ResourceBody, c eval.Context) eval.PValue {
	tes := re.Eval(title, c)
	if ta, ok := tes.(*types.ArrayValue); ok {
		return ta.Map(func(te eval.PValue) eval.PValue {
			return re.newResource(ctor, title, te, body, c)
		})
	}
	return re.newResource(ctor, title, tes, body, c)
}

func (re *resourceEval) newResource(ctor eval.Function, titleExpr parser.Expression, title eval.PValue, body *parser.ResourceBody, c eval.Context) eval.PValue {
	entries := make([]*types.HashEntry, 0)
	entries = append(entries, types.WrapHashEntry2(`title`, title))
	for _, op := range body.Operations() {
		if attr, ok := op.(*parser.AttributeOperation); ok {
			entries = append(entries, types.WrapHashEntry2(attr.Name(), re.Eval(attr.Value(), c)))
		} else {
			ops := op.(*parser.AttributesOperation)
			attrOps := re.Eval(ops.Expr(), c)
			if hash, hok := attrOps.(*types.HashValue); hok {
				entries = hash.AppendEntriesTo(entries)
			}
		}
	}
	obj := ctor.Call(c, nil, types.WrapHash(entries)).(eval.PuppetObject)
	defineResource(c, obj, titleExpr)
	return obj
}
