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
		text := types.WrapString(fmt.Sprintf("Applying %s", Reference(h)))
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
		return eval.UNDEF, eval.Error(eval.EVAL_MULTI_ERROR, issue.H{`errors`: errors})
	}
}

func (re *resourceEval) evaluateNodeExpression(c eval.Context, rn *node) (eval.PValue, issue.Reported) {
	setCurrentNode(c, rn)
	g := GetGraph(c)
	extEdges := g.From(rn.ID())
	setExternalEdgesFrom(c, extEdges)
	setResources(c, map[string]*handle{})

	var lambda eval.InvocableValue
	var err issue.Reported

	value := eval.UNDEF

	if rn.lambda == nil {
		value, err = re.evaluator.Evaluate(c, rn.expression)
		if err != nil {
			return nil, err
		}

		if l, ok := value.(eval.InvocableValue); ok {
			lambda = l
		}
	} else {
		lambda = rn.lambda
	}

	if lambda != nil {
		value = lambda.Call(c, nil, rn.parameters...)
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
	if c.Static() {
		return re.evaluator.Eval(expr, c)
	}
	switch expr.(type) {
	case *parser.AccessExpression:
		return re.eval_AccessExpression(expr.(*parser.AccessExpression), c)
	case *parser.RelationshipExpression:
		return re.eval_RelationshipExpression(expr.(*parser.RelationshipExpression), c)
	case *parser.ResourceExpression:
		return re.eval_ResourceExpression(expr.(*parser.ResourceExpression), c)
	default:
		return re.evaluator.Eval(expr, c)
	}
}

const parallel = `parallel`
const sequential = `sequential`

func (re *resourceEval) CallFunction(name string, args []eval.PValue, call parser.CallExpression, c eval.Context) eval.PValue {
	switch name {
	case parallel, sequential:
		// This is not a real function call but rather a node scheduling instruction
		return re.createEdgesWithArgument(c, name, args, call)
	default:
		return re.evaluator.CallFunction(name, args, call, c)
	}
}

func (re *resourceEval) createEdgesWithArgument(c eval.Context, name string, args []eval.PValue, call parser.CallExpression) eval.PValue {
	allInvocable := true
	for _, arg := range args {
		if _, ok := arg.(eval.InvocableValue); !ok {
			allInvocable = false
			break
		}
	}

	if allInvocable {
		// Create one node per function. The node calls the function without arguments
		nodes := make([]Node, len(args))
		for i, arg := range args {
			nodes[i] = newNode2(c, arg.(eval.InvocableValue))
		}
		return createEdgesForNodes(c, nodes, name)
	}

	nodeExpr, ok := call.Lambda().(*parser.LambdaExpression)
	if !ok {
		panic(eval.Error2(call, EVAL_DECLARATION_MUST_HAVE_LAMBDA, issue.H{`declaration`: name}))
	}


	ac := len(args)
	if ac != 1 {
		panic(eval.Error2(call, eval.EVAL_ILLEGAL_ARGUMENT_COUNT, issue.H{`expression`: call, `expected`: 1, `actual`: ac}))
	}
	arg := args[0]

	params := nodeExpr.Parameters()
	np := len(params)
	if np != 1 && np != 2 {
		panic(eval.Error2(call, eval.EVAL_ILLEGAL_ARGUMENT_COUNT, issue.H{`expression`: nodeExpr, `expected`: `1-2`, `actual`: np}))
	}

	// Emulate "each" style iteration with either one or two arguments
	nodes := make([]Node, 0, 8)
	eval.AssertInstance(name, types.DefaultIterableType(), arg)
	if hash, ok := arg.(*types.HashValue); ok {
		if np == 2 {
			hash.EachPair(func(k, v eval.PValue) {
				nodes = append(nodes, newNode(c, nodeExpr, k, v))
			})
		} else {
			hash.Each(func(v eval.PValue) {
				nodes = append(nodes, newNode(c, nodeExpr, v))
			})
		}
	} else {
		iter := arg.(eval.IterableValue)
		if np == 2 {
			if iter.IsHashStyle() {
				iter.Iterator().Each(func(v eval.PValue) {
					vi := v.(eval.IndexedValue)
					nodes = append(nodes, newNode(c, nodeExpr, vi.At(0), vi.At(1)))
				})
			} else {
				iter.Iterator().EachWithIndex(func(idx eval.PValue, v eval.PValue) {
					nodes = append(nodes, newNode(c, nodeExpr, idx, v))
				})
			}
		} else {
			iter.Iterator().Each(func(v eval.PValue) {
				nodes = append(nodes, newNode(c, nodeExpr, v))
			})
		}
	}

	return createEdgesForNodes(c, nodes, name)
}

func createEdgesForNodes(c eval.Context, nodes []Node, name string) eval.PValue {
	cn := getCurrentNode(c)
	g := GetGraph(c).(graph.DirectedBuilder)
	extEdges := getExternalEdgesFrom(c)

	if name == parallel {
		edges := make([]eval.PValue, len(nodes))
		for i, n := range nodes {
			// Create edge from current to node
			edge := newEdge(cn, n, false)
			g.SetEdge(edge)
			edges[i] = edge

			// node must evaluate before any edges external to the current node evaluates
			for _, cnTo := range extEdges {
				exEdge := g.Edge(cn.ID(), cnTo.ID()).(Edge)
				g.SetEdge(newEdge(n, cnTo.(Node), exEdge.Subscribe()))
			}
		}
		return types.WrapArray(edges)
	} else {
		// Create edge from current to first node in the list
		prev := nodes[0]
		// Create edge from current to node
		g.SetEdge(newEdge(cn, prev, false))

		// Create the edge chain that enforces sequential execution
		var lastEdge Edge
		nc := len(nodes)
		for i := 1; i < nc; i++ {
			nxt := nodes[i]
			lastEdge = newEdge(prev, nxt, false)
			g.SetEdge(lastEdge)
			prev = nxt
		}
		last := nodes[nc-1]
		// last node must evaluate before any edges external to the current node evaluates
		for _, cnTo := range extEdges {
			exEdge := g.Edge(cn.ID(), cnTo.ID()).(Edge)
			g.SetEdge(newEdge(last, cnTo.(Node), exEdge.Subscribe()))
		}
		return lastEdge
	}
}

func (re *resourceEval) eval_AccessExpression(expr *parser.AccessExpression, c eval.Context) eval.PValue {
	if qn, ok := expr.Operand().(*parser.QualifiedName); ok && (qn.Name() == parallel || qn.Name() == sequential) {
		// Deal with parallel[expr, ...] and sequential[expr, ...]
		nes := expr.Keys()
		nodes := make([]Node, len(nes))
		for i, ne := range nes {
			nodes[i] = newNode(c, ne)
		}
		return createEdgesForNodes(c, nodes, qn.Name())
	}
	return re.evaluator.Eval(expr, c)
}

func (re *resourceEval) eval_RelationshipExpression(expr *parser.RelationshipExpression, c eval.Context) eval.PValue {
	switch expr.Operator() {
	case `->`:
		return re.createEdges(c, expr.Lhs(), expr.Rhs(), false)
	case `~>`:
		return re.createEdges(c, expr.Lhs(), expr.Rhs(), true)
	case `<-`:
		return re.createEdges(c, expr.Rhs(), expr.Lhs(), false)
	default:
		return re.createEdges(c, expr.Rhs(), expr.Lhs(), true)
	}
}

func (re *resourceEval) createEdges(c eval.Context, first, second parser.Expression, subscribe bool) eval.PValue {
	// Evaluate first as part of this evaluation. We don't care about the actual evaluation result but we do
	// care about side effects.
	if r, ok := first.(*parser.RelationshipExpression); ok {
		// Only one side of this can be evaluated so we need to reorder the actual expression
		f := parser.DefaultFactory()
		switch r.Operator() {
		case `->`:
			return re.createEdges(c, r.Lhs(), f.RelOp(`->`, r.Rhs(), second, r.Locator(), r.ByteOffset(), r.ByteLength()), false)
		case `~>`:
			return re.createEdges(c, r.Lhs(), f.RelOp(`~>`, r.Rhs(), second, r.Locator(), r.ByteOffset(), r.ByteLength()), true)
		case `<-`:
			return re.createEdges(c, r.Rhs(), f.RelOp(`->`, r.Lhs(), second, r.Locator(), r.ByteOffset(), r.ByteLength()), false)
		default:
			return re.createEdges(c, r.Rhs(), f.RelOp(`~>`, r.Lhs(), second, r.Locator(), r.ByteOffset(), r.ByteLength()), true)
		}
	}


	var er eval.PValue = nil

	edges := make([]eval.PValue, 0)
	if ae, ok := first.(*parser.AccessExpression); ok {
		if qn, ok := ae.Operand().(*parser.QualifiedName); ok && (qn.Name() == parallel || qn.Name() == sequential) {
			// Deal with parallel[expr, ...] and sequential[expr, ...]
			nes := ae.Keys()
			nodes := make([]Node, len(nes))
			for i, ne := range nes {
				nodes[i] = newNode(c, ne)
			}
			efn := createEdgesForNodes(c, nodes, qn.Name())
			if arr, ok := efn.(*types.ArrayValue); ok {
				arr.AppendTo(edges)
			} else {
				edges = append(edges, efn)
			}
		} else {
			er = re.Eval(first, c)
		}
	} else {
		er = re.Eval(first, c)
	}

	cn := getCurrentNode(c)
	g := GetGraph(c).(graph.DirectedBuilder)
	leafs := findLeafs(g, cn, []Node{})
	for _, rn := range createNodes(c, second, []Node{}) {
		// Create edge leafs of from current
		for _, leaf := range leafs {
			edge := newEdge(leaf.(Node), rn, subscribe)
			g.SetEdge(edge)
			edges = append(edges, edge)
		}
	}

	if er == nil {
		if len(edges) == 1 {
			er = edges[0]
		} else {
			er = types.WrapArray(edges)
		}
	}
	return er
}

func findLeafs(g graph.Directed, n graph.Node, leafs []Node) []Node {
	depNodes := g.From(n.ID())
	if len(depNodes) == 0 {
		leafs = append(leafs, n.(Node))
	} else {
		for _, l := range depNodes {
			leafs = findLeafs(g, l, leafs)
		}
	}
	return leafs
}

func createNodes(c eval.Context, expr parser.Expression, nodes []Node) []Node {
	if ae, ok := expr.(*parser.AccessExpression); ok {
		if qn, ok := ae.Operand().(*parser.QualifiedName); ok && qn.Name() == parallel {
			for _, el := range ae.Keys() {
				nodes = createNodes(c, el, nodes)
			}
			return nodes
		}
	}
	return append(nodes, newNode(c, expr))
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
