package js2ast

import (
	"github.com/puppetlabs/go-parser/parser"
	"github.com/puppetlabs/go-parser/pn"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-parser/validator"
	"github.com/puppetlabs/go-evaluator/errors"
	"github.com/puppetlabs/go-evaluator/impl"
)

type forExpression struct {
	parser.Positioned

	init parser.Expression
	test parser.Expression
	update parser.Expression
	body parser.Expression
}

func NewForExpression(init, test, update, body parser.Expression, locator *parser.Locator, pos, len int) parser.Expression {
	fe := &forExpression{init: init, test: test, update: update, body: body}
	fe.Init(locator, pos, len)
	return fe
}

func (fe *forExpression) AllContents(path []parser.Expression, visitor parser.PathVisitor) {
	parser.DeepVisit(fe, path, visitor, fe.init, fe.test, fe.update, fe.body)
}

func (fe *forExpression) Contents(path []parser.Expression, visitor parser.PathVisitor) {
	parser.ShallowVisit(fe, path, visitor, fe.init, fe.test, fe.update, fe.body)
}

func (fe *forExpression) ToPN() pn.PN {
	entries := make([]pn.Entry, 0, 4)
	if !fe.init.IsNop() {
		entries = append(entries, fe.init.ToPN().WithName(`init`))
	}
	if !fe.test.IsNop() {
		entries = append(entries, fe.test.ToPN().WithName(`test`))
	}
	if !fe.update.IsNop() {
		entries = append(entries, fe.update.ToPN().WithName(`update`))
	}
	if !fe.body.IsNop() {
		entries = append(entries, fe.body.ToPN().WithName(`body`))
	}
	return pn.Map(entries).AsCall(`for`)
}

func (fe *forExpression) Evaluate(e eval.Evaluator, c eval.Context) eval.PValue {
	return c.Scope().WithLocalScope(func() eval.PValue {
		if !fe.init.IsNop() {
			e.Eval(fe.init, c)
		}

		oneIteration := func() (cont bool) {
			defer func() {
				cont = true
				if err := recover(); err != nil {
					switch err.(type) {
					case *errors.StopIteration:
						cont = false
					case *errors.NextIteration:
						// No action.
					}
				}
			}()

			if !fe.body.IsNop() {
				e.Eval(fe.body, c)
			}
			return
		}

		for {
			if !fe.test.IsNop() {
				r := e.Eval(fe.test, c)
				if eval.Equals(r, eval.UNDEF) || eval.Equals(r, types.Boolean_FALSE) {
					break
				}
			}
			if !oneIteration() {
				break
			}
			if !fe.update.IsNop() {
				e.Eval(fe.update, c)
			}
		}
		return eval.UNDEF
	})
}

type thisExpression struct {
	parser.Positioned
}

func NewThisExpression(locator *parser.Locator, pos, len int) parser.Expression {
	te := &thisExpression{}
	te.Init(locator, pos, len)
	return te
}

func (te *thisExpression) AllContents(path []parser.Expression, visitor parser.PathVisitor) {
}

func (te *thisExpression) Contents(path []parser.Expression, visitor parser.PathVisitor) {
}

func (te *thisExpression) ToPN() pn.PN {
	return pn.Call(`this`)
}

type caseExpression struct {
	parser.Positioned

	test parser.Expression
	body parser.Expression
}

type switchExpression struct {
	parser.Positioned

	discriminant parser.Expression
	cases []parser.Expression
}

func NewCaseExpression(test, body parser.Expression, locator *parser.Locator, pos, len int) parser.Expression {
	ce := &caseExpression{test: test, body: body}
	ce.Init(locator, pos, len)
	return ce
}

func (ce *caseExpression) AllContents(path []parser.Expression, visitor parser.PathVisitor) {
	if ce.test == nil {
		parser.DeepVisit(ce, path, visitor, ce.body)
	} else {
		parser.DeepVisit(ce, path, visitor, ce.test, ce.body)
	}
}

func (ce *caseExpression) Contents(path []parser.Expression, visitor parser.PathVisitor) {
	if ce.test == nil {
		parser.ShallowVisit(ce, path, visitor, ce.body)
	} else {
		parser.ShallowVisit(ce, path, visitor, ce.test, ce.body)
	}
}

func (ce *caseExpression) ToPN() pn.PN {
	entries := make([]pn.Entry, 0, 3)
	if ce.test != nil {
		entries = append(entries, ce.test.ToPN().WithName(`test`))
	}
	entries = append(entries, ce.body.ToPN().WithName(`body`))
	return pn.Map(entries).AsCall(`js::case`)
}

func NewSwitchExpression(discriminant parser.Expression, cases []parser.Expression, locator *parser.Locator, pos, len int) parser.Expression {
	se := &switchExpression{discriminant: discriminant, cases: cases}
	se.Init(locator, pos, len)
	return se
}

func (se *switchExpression) AllContents(path []parser.Expression, visitor parser.PathVisitor) {
	parser.DeepVisit(se, path, visitor, se.discriminant, se.cases)
}

func (se *switchExpression) Contents(path []parser.Expression, visitor parser.PathVisitor) {
	parser.ShallowVisit(se, path, visitor, se.discriminant, se.cases)
}

func (se *switchExpression) ToPN() pn.PN {
	entries := make([]pn.Entry, 0, 3)
	entries = append(entries, se.discriminant.ToPN().WithName(`discriminant`))
	pnl := make([]pn.PN, len(se.cases))
	for i, c := range se.cases {
		pnl[i] = c.ToPN()
	}
	entries = append(entries, pn.List(pnl).WithName(`cases`))
	return pn.Map(entries).AsCall(`js::switch`)
}

func (fe *switchExpression) Evaluate(e eval.Evaluator, c eval.Context) eval.PValue {
	return c.Scope().WithLocalScope(func() eval.PValue {
		discr := e.Eval(fe.discriminant, c)
		entry := -1
		dfltCase := -1
		for i, cn := range fe.cases {
			cs := cn.(*caseExpression)
			if cs.test == nil {
				dfltCase = i
				continue
			}

			// Perform a strict === equality check
			if impl.JsEquals(discr, e.Eval(cs.test, c), true) {
				entry = i
				break
			}
		}

		if entry == -1 {
			// No case test was equal to the discriminant. Use default
			if dfltCase < 0 {
				// There was no default
				return eval.UNDEF
			}
			entry = dfltCase
		}

		// Evaluate from default and onwards. Cases that do not break out
		// will fallthrough to next
		evalCase := func(caseBody parser.Expression) (cont bool) {
			cont = true
			defer func() {
				if err := recover(); err != nil {
					if _, ok := err.(*errors.StopIteration); ok {
						// Normal break issued from an evaluated case
						cont = false
					} else {
						panic(err)
					}
				}
			}()
			e.Eval(caseBody, c)
			return
		}
		t := len(fe.cases)
		for i := entry; i < t; i++ {
			if !evalCase(fe.cases[i].(*caseExpression).body) {
				break
			}
		}
		return eval.UNDEF
	})
}

type unaryNumericExpression struct {
	parser.Positioned

	// Only variables allowed. Objects are considered immutable
	variable *parser.VariableExpression

	// --x or x--
	postFix bool

	// -- or ++
	increment bool
}

func NewUnaryNumericExpression(v *parser.VariableExpression, postFix, increment bool, locator *parser.Locator, pos, len int) parser.Expression {
	une := &unaryNumericExpression{variable: v, postFix: postFix, increment: increment}
	une.Init(locator, pos, len)
	return une
}

func (ie *unaryNumericExpression) Expr() parser.Expression {
	return ie.variable
}

func (ie *unaryNumericExpression) Contents(path []parser.Expression, visitor parser.PathVisitor) {
	parser.ShallowVisit(ie, path, visitor, ie.variable)
}

func (ie *unaryNumericExpression) AllContents(path []parser.Expression, visitor parser.PathVisitor) {
	parser.DeepVisit(ie, path, visitor, ie.variable)
}

func (ie *unaryNumericExpression) Postfix() bool {
	return ie.postFix
}

func (ie *unaryNumericExpression) Increment() bool {
	return ie.increment
}

func (ie *unaryNumericExpression) ToUnaryExpression() parser.UnaryExpression {
	return ie
}

func (ie *unaryNumericExpression)  ToPN() pn.PN {
	sf := `pre`
	op := `-decr`
	if ie.postFix {
		sf = `post`
	}
	if ie.increment {
		op = `-incr`
	}
	return pn.Call(sf + op, ie.Expr().ToPN())
}

func (ie *unaryNumericExpression)  Op() string {
	if ie.increment {
		return `++`
	}
	return `--`
}

func (ie *unaryNumericExpression) Evaluate(e eval.Evaluator, c eval.Context) eval.PValue {
	num := e.Eval(ie.variable, c)
	if ov, ok := num.(*types.IntegerValue); ok {
		iv := ov.Int()
		if ie.increment {
			iv++
		} else {
			iv--
		}

		nv := types.WrapInteger(iv)
		name, _ := ie.variable.Name()
		if !c.Scope().Set(name, nv) {
			panic(eval.Error2(ie, eval.EVAL_ILLEGAL_REASSIGNMENT, issue.H{`var`: name}))
		}

		if ie.postFix {
			return ov
		}
		return nv
	}
	panic(eval.Error2(ie, validator.VALIDATE_UNSUPPORTED_OPERATOR_IN_CONTEXT, issue.H{`operator`: ie.Op(), `value`: num.Type()}))
}
