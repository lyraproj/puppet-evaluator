package impl

import (
	"fmt"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-parser/issue"
	"github.com/puppetlabs/go-parser/parser"
	"github.com/puppetlabs/go-parser/validator"
)

type (
	context struct {
		evaluator   eval.Evaluator
		loader      eval.Loader
		stack       []issue.Location
		scope       eval.Scope
		definitions []interface{}
		vars        map[string]eval.PValue
	}
)

func (c *context) DefiningLoader() eval.DefiningLoader {
	l := c.loader
	for {
		if dl, ok := l.(eval.DefiningLoader); ok {
			return dl
		}
		if pl, ok := l.(eval.ParentedLoader); ok {
			l = pl.Parent()
			continue
		}
		panic(`No defining loader found in context`)
	}
}

func NewContext(evaluator eval.Evaluator, loader eval.Loader) eval.Context {
	return &context{evaluator, loader, make([]issue.Location, 0, 8), nil, nil, nil}
}

func NewContext2(evaluator eval.Evaluator, loader eval.Loader, scope eval.Scope) eval.Context {
	return &context{evaluator, loader, make([]issue.Location, 0, 8), scope, nil, nil}
}

func (c *context) AddDefinitions(expr parser.Expression) {
	if prog, ok := expr.(*parser.Program); ok {
		loader := c.DefiningLoader()
		for _, d := range prog.Definitions() {
			c.define(loader, d)
		}
	}
}

func (c *context) Call(name string, args []eval.PValue, block eval.Lambda) eval.PValue {
	tn := eval.NewTypedName2(`function`, name, c.Loader().NameAuthority())
	if f, ok := eval.Load(c, tn); ok {
		return f.(eval.Function).Call(c, block, args...)
	}
	panic(issue.NewReported(eval.EVAL_UNKNOWN_FUNCTION, issue.SEVERITY_ERROR, issue.H{`name`: tn.String()}, c.StackTop()))
}

func (c *context) Error(location issue.Location, issueCode issue.Code, args issue.H) *issue.Reported {
	if location == nil {
		location = c.StackTop()
	}
	return issue.NewReported(issueCode, issue.SEVERITY_ERROR, args, location)
}

func (c *context) Evaluate(expr parser.Expression) eval.PValue {
	return c.evaluator.Eval(expr, c)
}

func (c *context) EvaluateIn(expr parser.Expression, scope eval.Scope) eval.PValue {
	return c.evaluator.Eval(expr, c.WithScope(scope))
}

func (c *context) Evaluator() eval.Evaluator {
	return c.evaluator
}

func (c *context) Fork() eval.Context {
	s := make([]issue.Location, len(c.stack))
	copy(s, c.stack)
	clone := c.clone()
	clone.stack = s
	return clone
}

func (c *context) Fail(message string) *issue.Reported {
	return c.Error(nil, eval.EVAL_FAILURE, issue.H{`message`: message})
}

func (c *context) Loader() eval.Loader {
	return c.loader
}

func (c *context) Logger() eval.Logger {
	return c.evaluator.Logger()
}

func (c *context) ParseAndValidate(filename, str string, singleExpression bool) parser.Expression {
	var parserOptions []parser.Option
	if eval.GetSetting(`tasks`, types.Boolean_FALSE).(*types.BooleanValue).Bool() {
		parserOptions = append(parserOptions, parser.PARSER_TASKS_ENABLED)
	}
	expr, err := parser.CreateParser(parserOptions...).Parse(filename, str, singleExpression)
	if err != nil {
		panic(err)
	}
	checker := validator.NewChecker(validator.STRICT_ERROR)
	checker.Validate(expr)
	issues := checker.Issues()
	if len(issues) > 0 {
		severity := issue.SEVERITY_IGNORE
		for _, issue := range issues {
			c.Logger().Log(eval.LogLevel(issue.Severity()), types.WrapString(issue.String()))
			if issue.Severity() > severity {
				severity = issue.Severity()
			}
		}
		if severity == issue.SEVERITY_ERROR {
			c.Fail(fmt.Sprintf(`Error validating %s`, filename))
		}
	}
	return expr
}

func (c *context) ParseType(typeString eval.PValue) eval.PType {
	if sv, ok := typeString.(*types.StringValue); ok {
		return c.ParseType2(sv.String())
	}
	panic(types.NewIllegalArgumentType2(`ParseType`, 0, `String`, typeString))
}

func (c *context) ParseType2(str string) eval.PType {
	return c.ResolveType(c.ParseAndValidate(``, str, true))
}

func (c *context) ResolveDefinitions() {
	if c.definitions == nil {
		return
	}

	for len(c.definitions) > 0 {
		defs := c.definitions
		c.definitions = nil
		for _, d := range defs {
			switch d.(type) {
			case *puppetFunction:
				d.(*puppetFunction).Resolve(c)
			case eval.ResolvableType:
				d.(eval.ResolvableType).Resolve(c)
			}
		}
	}
}

func (c *context) ResolveResolvables() {
	c.Loader().(eval.DefiningLoader).ResolveResolvables(c)
}

func (c *context) ResolveType(expr parser.Expression) eval.PType {
	resolved := c.Evaluate(expr)
	if pt, ok := resolved.(eval.PType); ok {
		return pt
	}
	panic(fmt.Sprintf(`Expression "%s" does no resolve to a Type`, expr.String()))
}

func (c *context) Scope() eval.Scope {
	if c.scope == nil {
		c.scope = NewScope()
	}
	return c.scope
}

func (c *context) Stack() []issue.Location {
	return c.stack
}

func (c *context) StackPop() {
	c.stack = c.stack[:len(c.stack)-1]
}

func (c *context) StackPush(location issue.Location) {
	c.stack = append(c.stack, location)
}

func (c *context) StackTop() issue.Location {
	s := len(c.stack)
	if s == 0 {
		return &systemLocation{}
	}
	return c.stack[s-1]
}

func (c *context) WithLoader(loader eval.Loader) eval.Context {
	clone := c.clone()
	clone.loader = loader
	return clone
}

func (c *context) WithScope(scope eval.Scope) eval.Context {
	clone := c.clone()
	clone.scope = scope
	return clone
}

// clone a new context from this context which is an exact copy. It is used
// internally by methods like Fork, WithLoader, and WithScope to prevent them
// from creating specific implementations.
func (c *context) clone() *context {
	return &context{c.evaluator, c.loader, c.stack, c.scope, c.definitions, c.vars}
}

func (c *context) define(loader eval.DefiningLoader, d parser.Definition) {
	var ta interface{}
	var tn eval.TypedName
	switch d.(type) {
	case *parser.PlanDefinition:
		pe := d.(*parser.PlanDefinition)
		tn = eval.NewTypedName2(eval.PLAN, pe.Name(), loader.NameAuthority())
		ta = NewPuppetPlan(pe)
	case *parser.FunctionDefinition:
		fe := d.(*parser.FunctionDefinition)
		tn = eval.NewTypedName2(eval.FUNCTION, fe.Name(), loader.NameAuthority())
		ta = NewPuppetFunction(fe)
	default:
		ta, tn = types.CreateTypeDefinition(d, loader.NameAuthority())
	}
	loader.SetEntry(tn, eval.NewLoaderEntry(ta, d))
	if c.definitions == nil {
		c.definitions = []interface{}{ta}
	} else {
		c.definitions = append(c.definitions, ta)
	}
}
