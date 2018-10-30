package js2ast

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-parser/parser"

	"fmt"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-parser/validator"
	"github.com/robertkrimen/otto/ast"
	jsparser "github.com/robertkrimen/otto/parser"
	"github.com/robertkrimen/otto/token"
	"log"
)

type transformer struct {
	c       eval.Context
	l       *parser.Locator
	f       parser.ExpressionFactory
	functor bool
	lambda  bool
}

// JavaScriptToAST parses and transforms the given JavaScript content into a Puppet AST. It will
// panic with an issue.Reported unless the parsing and transformation was succesful.
func JavaScriptToAST(c eval.Context, filename string, content []byte) parser.Expression {
	InitJavaScript(c)
	program, err := jsparser.ParseFile(nil, filename, content, 0)
	if err != nil {
		panic(eval.Error(eval.EVAL_PARSE_ERROR, issue.H{`language`: `JavaScript`, `detail`: err.Error()}))
	}
	jp := &transformer{c, parser.NewLocator(filename, string(content)), parser.DefaultFactory(), false, false}
	return jp.transformProgram(program)
}

// EvaluateJavaScript calls JavaScriptToAST to parse and transform the given JavaScript content into
// a Puppet AST which is then evaluated by the eval.Evaluator obtained from the given
// eval.Context. The result of the evaluation is returned.
func EvaluateJavaScript(c eval.Context, filename string, content []byte) (eval.Value, error) {
	program := JavaScriptToAST(c, filename, content)
	log.Println(program.ToPN())
	c.AddDefinitions(program)
	for _, def := range program.(*parser.Program).Definitions() {
		log.Println(def.ToPN())
	}
	return c.Evaluator().Evaluate(c, program)
}

func (jp *transformer) error(node ast.Node, code issue.Code, args issue.H) issue.Reported {
	pos := lpos(node)
	return eval.Error2(issue.NewLocation(jp.l.File(), jp.l.LineForOffset(pos), jp.l.PosOnLine(pos)), code, args)
}

func (jp *transformer) transformArray(jsArray *ast.ArrayLiteral) parser.Expression {
	return jp.f.Array(jp.transformExpressions(jsArray.Value), jp.l, lpos(jsArray), llen(jsArray))
}

func (jp *transformer) transformAssign(jsAsg *ast.AssignExpression) parser.Expression {
	lhs := jp.transformExpression(jsAsg.Left)
	rhs := jp.transformExpression(jsAsg.Right)
	op := jsAsg.Operator
	switch op {
	case token.ASSIGN:
		// No action
	case token.PLUS, token.MINUS, token.MULTIPLY, token.SLASH, token.REMAINDER:
		// +=, -=, *=, /=, %=
		rhs = jp.f.Arithmetic(op.String(), lhs, rhs, jp.l, lpos(jsAsg), llen(jsAsg))
		op = token.ASSIGN
	default:
		panic(eval.Error(EVAL_JS_UNHANDLED_EXPRESSION, issue.H{`expr`: jsAsg}))
	}
	return jp.f.Assignment(op.String(), lhs, rhs, jp.l, lpos(jsAsg), llen(jsAsg))
}

func (jp *transformer) transformBinary(jsBinary *ast.BinaryExpression) parser.Expression {
	lhs := jp.transformExpression(jsBinary.Left)
	rhs := jp.transformExpression(jsBinary.Right)
	switch jsBinary.Operator {
	case token.PLUS, token.MINUS, token.MULTIPLY, token.SLASH, token.REMAINDER:
		return jp.f.Arithmetic(jsBinary.Operator.String(), lhs, rhs, jp.l, lpos(jsBinary), llen(jsBinary))
	case token.LESS, token.LESS_OR_EQUAL, token.GREATER, token.GREATER_OR_EQUAL, token.EQUAL, token.NOT_EQUAL, token.STRICT_EQUAL, token.STRICT_NOT_EQUAL:
		return jp.f.Comparison(jsBinary.Operator.String(), lhs, rhs, jp.l, lpos(jsBinary), llen(jsBinary))
	case token.IN:
		return jp.f.In(lhs, rhs, jp.l, lpos(jsBinary), llen(jsBinary))
	case token.INSTANCEOF:
		return jp.transformInstanceOf(jsBinary)
	case token.LOGICAL_AND:
		return jp.f.And(lhs, rhs, jp.l, lpos(jsBinary), llen(jsBinary))
	case token.LOGICAL_OR:
		return jp.f.Or(lhs, rhs, jp.l, lpos(jsBinary), llen(jsBinary))
	default:
		panic(eval.Error(EVAL_JS_UNHANDLED_EXPRESSION, issue.H{`expr`: jsBinary}))
	}
}

func (jp *transformer) transformBlock(jsBlock *ast.BlockStatement, el []parser.Expression) []parser.Expression {
	for _, s := range jsBlock.List {
		el = jp.transformStatement(s, el)
	}
	return el
}

func (jp *transformer) transformBranch(jsBranch *ast.BranchStatement) parser.Expression {
	st := lpos(jsBranch)
	ln := llen(jsBranch)
	fn := ``
	switch jsBranch.Token {
	case token.CONTINUE:
		fn = `next`
	case token.BREAK:
		fn = `break`
	default:
		panic(eval.Error(EVAL_JS_UNHANDLED_EXPRESSION, issue.H{`expr`: jsBranch}))
	}
	return jp.f.CallNamed(
		jp.f.QualifiedName(fn, jp.l, st, ln), false, []parser.Expression{}, nil, jp.l, st, ln)
}

func (jp *transformer) transformBoolean(jsBool *ast.BooleanLiteral) parser.Expression {
	return jp.f.Boolean(jsBool.Value, jp.l, lpos(jsBool), llen(jsBool))
}

func (jp *transformer) transformBracket(jsBracket *ast.BracketExpression) parser.Expression {
	return jp.f.Access(
		jp.transformExpression(jsBracket.Left),
		[]parser.Expression{jp.transformExpression(jsBracket.Member)},
		jp.l, lpos(jsBracket), llen(jsBracket))
}

func (jp *transformer) transformCall(jsCall *ast.CallExpression) parser.Expression {
	callee := jp.transformFunctor(jsCall.Callee)
	args := jp.transformExpressions(jsCall.ArgumentList)
	argc := len(args)

	var lambda parser.Expression = nil
	if argc > 0 {
		last := args[argc-1]
		if l, ok := last.(*parser.LambdaExpression); ok {
			lambda = l
			argc--
			args = args[:argc]
		}
	}

	if na, ok := callee.(*parser.NamedAccessExpression); ok {
		return jp.f.CallMethod(na, args, lambda, jp.l, lpos(jsCall), llen(jsCall))
	}
	return jp.f.CallNamed(callee, false, args, lambda, jp.l, lpos(jsCall), llen(jsCall))
}

func (jp *transformer) transformCase(jsCase *ast.CaseStatement) parser.Expression {
	be := make([]parser.Expression, 0, len(jsCase.Consequent))
	for _, jsStmt := range jsCase.Consequent {
		be = jp.transformStatement(jsStmt, be)
	}
	var test parser.Expression
	if jsCase.Test != nil {
		test = jp.transformExpression(jsCase.Test)
	}
	return NewCaseExpression(test, jp.exprsAsBody(be), jp.l, lpos(jsCase), llen(jsCase))
}

func (jp *transformer) transformConditional(jsCond *ast.ConditionalExpression) parser.Expression {
	return jp.f.If(
		jp.transformExpression(jsCond.Test),
		jp.transformExpression(jsCond.Consequent),
		jp.transformExpression(jsCond.Alternate),
		jp.l, lpos(jsCond), llen(jsCond))
}

// Transform all function declarations into definitions, and all variable declarations into variable expressions
func (jp *transformer) transformDeclarations(decls []ast.Declaration, ns string, defs []parser.Definition) []parser.Definition {
	for _, decl := range decls {
		if f, ok := decl.(*ast.FunctionDeclaration); ok {
			defs = jp.transformFunctionDecl(f, ns, defs)
			continue
		}
		if _, ok := decl.(*ast.VariableDeclaration); ok {
			// Also included as statements
			continue
		}
		panic(eval.Error(EVAL_JS_UNHANDLED_EXPRESSION, issue.H{`expr`: decl}))
	}
	return defs
}

func (jp *transformer) transformDot(jsDot *ast.DotExpression) parser.Expression {
	id := jsDot.Identifier
	lhs := jp.transformExpression(jsDot.Left)
	rhs := jp.transformIdentifier(id, false)
	if qrr, ok := rhs.(*parser.QualifiedReference); ok {
		// Only time RHS can be a qualified reference is when
		// LHS also is. They then form a Puppet multi segment name
		// in the form Foo::Bar
		if qrl, ok := lhs.(*parser.QualifiedReference); ok {
			return jp.f.QualifiedReference(
				qrl.Name()+`::`+qrr.Name(), jp.l, lpos(jsDot), llen(jsDot))
		}
		panic(eval.Error(EVAL_JS_UNHANDLED_EXPRESSION, issue.H{`expr`: jsDot}))
	}
	if jp.functor {
		return jp.f.NamedAccess(
			lhs, rhs, jp.l, lpos(jsDot), llen(jsDot))
	}
	return jp.f.Access(lhs, []parser.Expression{rhs}, jp.l, lpos(jsDot), llen(jsDot))
}

func (jp *transformer) transformDoWhileStatement(jsDoWhile *ast.DoWhileStatement) parser.Expression {
	body := jp.exprsAsBody(jp.transformStatement(jsDoWhile.Body, []parser.Expression{}))
	return NewForExpression(
		body,
		jp.transformExpression(jsDoWhile.Test),
		jp.f.Nop(jp.l, lpos(jsDoWhile), 0),
		body,
		jp.l,
		lpos(jsDoWhile), llen(jsDoWhile))
}

func (jp *transformer) transformEmpty(jsEmpty ast.Node) parser.Expression {
	return jp.f.Nop(jp.l, lpos(jsEmpty), llen(jsEmpty))
}

func (jp *transformer) transformExpression(jsExpr ast.Expression) parser.Expression {
	switch jsExpr.(type) {
	case *ast.ArrayLiteral:
		return jp.transformArray(jsExpr.(*ast.ArrayLiteral))
	case *ast.AssignExpression:
		return jp.transformAssign(jsExpr.(*ast.AssignExpression))
	case *ast.BinaryExpression:
		return jp.transformBinary(jsExpr.(*ast.BinaryExpression))
	case *ast.BooleanLiteral:
		return jp.transformBoolean(jsExpr.(*ast.BooleanLiteral))
	case *ast.BracketExpression:
		return jp.transformBracket(jsExpr.(*ast.BracketExpression))
	case *ast.CallExpression:
		return jp.transformCall(jsExpr.(*ast.CallExpression))
	case *ast.ConditionalExpression:
		return jp.transformConditional(jsExpr.(*ast.ConditionalExpression))
	case *ast.DotExpression:
		return jp.transformDot(jsExpr.(*ast.DotExpression))
	case *ast.EmptyExpression:
		return jp.transformEmpty(jsExpr.(*ast.EmptyExpression))
	case *ast.FunctionLiteral:
		return jp.transformFunction(jsExpr.(*ast.FunctionLiteral))
	case *ast.Identifier:
		return jp.transformIdentifier(jsExpr.(*ast.Identifier), true)
	case *ast.NewExpression:
		return jp.transformNew(jsExpr.(*ast.NewExpression))
	case *ast.NullLiteral:
		return jp.transformNull(jsExpr.(*ast.NullLiteral))
	case *ast.NumberLiteral:
		return jp.transformNumber(jsExpr.(*ast.NumberLiteral))
	case *ast.ObjectLiteral:
		return jp.transformObject(jsExpr.(*ast.ObjectLiteral))
	case *ast.RegExpLiteral:
		return jp.transformRegexp(jsExpr.(*ast.RegExpLiteral))
	case *ast.SequenceExpression:
		return jp.transformSequence(jsExpr.(*ast.SequenceExpression))
	case *ast.StringLiteral:
		return jp.transformString(jsExpr.(*ast.StringLiteral))
	case *ast.ThisExpression:
		return jp.transformThis(jsExpr.(*ast.ThisExpression))
	case *ast.UnaryExpression:
		return jp.transformUnary(jsExpr.(*ast.UnaryExpression))
	case *ast.VariableExpression:
		return jp.transformVariable(jsExpr.(*ast.VariableExpression))
	default:
		panic(eval.Error(EVAL_JS_UNHANDLED_EXPRESSION, issue.H{`expr`: jsExpr}))
	}
}

func (jp *transformer) transformExpressions(jsExprs []ast.Expression) []parser.Expression {
	el := make([]parser.Expression, len(jsExprs))
	for i, jsExpr := range jsExprs {
		el[i] = jp.transformExpression(jsExpr)
	}
	return el
}

func (jp *transformer) exprsAsBody(be []parser.Expression) parser.Expression {
	bl := len(be)
	switch(bl) {
	case 0:
		return jp.f.Nop(jp.l, 0, 0)
	case 1:
		return be[0]
	default:
		bs := be[0].ByteOffset()
		l := be[bl-1]
		return jp.f.Block(be, jp.l, bs, (l.ByteOffset() - bs) + l.ByteLength())
	}
}

func (jp *transformer) transformFor(jsFor *ast.ForStatement) parser.Expression {
	return NewForExpression(
		jp.transformExpression(jsFor.Initializer),
		jp.transformExpression(jsFor.Test),
		jp.transformExpression(jsFor.Update),
		jp.exprsAsBody(jp.transformStatement(jsFor.Body, []parser.Expression{})),
		jp.l,
		lpos(jsFor), llen(jsFor))
}

// Transforms for(x in y) { ... } to js::forIn($y) |$x| { ... }
func (jp *transformer) transformForIn(jsForIn *ast.ForInStatement) parser.Expression {
	id, ok := jsForIn.Into.(*ast.Identifier)
	if !ok {
		panic(eval.Error(EVAL_JS_UNHANDLED_EXPRESSION, issue.H{`expr`: jsForIn}))
	}

	st := lpos(jsForIn)
	el := jp.transformStatement(jsForIn.Body, []parser.Expression{})
	args := []parser.Expression{jp.transformExpression(jsForIn.Source)}
	param := jp.f.Parameter(id.Name, nil, nil, false, jp.l, lpos(id), llen(id))
	body := jp.exprsAsBody(el)
	lambda := jp.f.Lambda([]parser.Expression{param}, body, nil, jp.l, body.ByteOffset(), body.ByteLength())
	name := jp.f.QualifiedName(`js::forIn`, jp.l, st, 3)

	return jp.f.CallNamed(name, false, args, lambda, jp.l, st, llen(jsForIn))
}

func (jp *transformer) transformFunction(jsFunc *ast.FunctionLiteral) parser.Expression {
	if jsFunc.Name != nil {
		panic(eval.Error(EVAL_JS_UNHANDLED_EXPRESSION, issue.H{`expr`: jsFunc}))
	}

	lambda := jp.lambda
	jp.lambda = true

	// defs := jp.transformDeclarations(jsFunc.DeclarationList, ``, []parser.Definition{})
	params := jp.transformParameters(jsFunc.ParameterList)
	stmts := jp.transformStatement(jsFunc.Body, []parser.Expression{})

	jp.lambda = lambda
	ns := len(stmts)

	if ns > 0 {
		// If last call is a call to next (i.e. originally a return), then unwrap it and return
		// it's argument expression instead.
		if nx, ok := stmts[ns-1].(*parser.CallNamedFunctionExpression); ok {
			if fc, ok := nx.Functor().(*parser.QualifiedName); ok && fc.Name() == `next` {
				args := nx.Arguments()
				switch len(args) {
				case 0:
					stmts = stmts[:len(stmts)-1]
				case 1:
					stmts[len(stmts)-1] = nx.Arguments()[0]
				}
			}
		}
	}
	return jp.f.Lambda(params, jp.exprsAsBody(stmts), nil, jp.l, lpos(jsFunc), llen(jsFunc))
}

func (jp *transformer) transformFunctionDecl(f *ast.FunctionDeclaration, ns string, defs []parser.Definition) []parser.Definition {
	fl := f.Function
	fn := fl.Name.Name
	if ns == `` {
		ns = fn
	} else {
		ns = fmt.Sprintf(`%s::%s`, ns, fn)
	}

	defs = jp.transformDeclarations(fl.DeclarationList, ns, defs)
	params := jp.transformParameters(fl.ParameterList)
	stmts := jp.transformStatement(fl.Body, []parser.Expression{})
	fnc := jp.f.Function(fn, params, jp.exprsAsBody(stmts), nil, jp.l, lpos(fl), llen(fl))
	return append(defs, fnc.(parser.Definition))
	return defs
}

func (jp *transformer) transformFunctor(jsFunctor ast.Expression) parser.Expression {
	if id, ok := jsFunctor.(*ast.Identifier); ok {
		// An identifier used as a functor is not a variable
		return jp.transformIdentifier(id, false)
	}
	functor := jp.functor
	jp.functor = true
	expr := jp.transformExpression(jsFunctor)
	jp.functor = functor
	return expr
}

func (jp *transformer) transformIdentifier(jsIdent *ast.Identifier, idAsVar bool) parser.Expression {
	name := jsIdent.Name
	if validator.CLASSREF_DECL.MatchString(name) {
		expr := jp.f.QualifiedName(name, jp.l, lpos(jsIdent), llen(jsIdent))
		if idAsVar {
			expr = jp.f.Variable(expr, jp.l, lpos(jsIdent), llen(jsIdent))
		}
		return expr
	}
	if validator.CLASSREF_EXT.MatchString(name) {
		return jp.f.QualifiedReference(name, jp.l, lpos(jsIdent), llen(jsIdent))
	}
	panic(jp.error(jsIdent, parser.LEX_INVALID_NAME, issue.NO_ARGS))
}

func (jp *transformer) transformIf(jsCond *ast.IfStatement) parser.Expression {
	return jp.f.If(
		jp.transformExpression(jsCond.Test),
		jp.exprsAsBody(jp.transformStatement(jsCond.Consequent, []parser.Expression{})),
		jp.exprsAsBody(jp.transformStatement(jsCond.Alternate, []parser.Expression{})),
		jp.l, lpos(jsCond), llen(jsCond))
}

func (jp *transformer) transformInstanceOf(jsInstOf *ast.BinaryExpression) parser.Expression {
	pos := lpos(jsInstOf)
	return jp.f.CallNamed(
		jp.f.QualifiedName(`js::isInstance`, jp.l, pos, 0), false, []parser.Expression{
			jp.transformExpression(jsInstOf.Right),
			jp.transformExpression(jsInstOf.Left)}, nil, jp.l, pos, llen(jsInstOf))
}

func (jp *transformer) transformNew(jsNew *ast.NewExpression) parser.Expression {
	pos := lpos(jsNew)
	callee := jp.f.NamedAccess(
		jp.transformExpression(jsNew.Callee),
		jp.f.QualifiedName(`new`, jp.l, pos, 3),
		jp.l, pos, llen(jsNew.Callee)-pos)
	return jp.f.CallMethod(callee, jp.transformExpressions(jsNew.ArgumentList), nil, jp.l, pos, llen(jsNew))
}

func (jp *transformer) transformNull(null *ast.NullLiteral) parser.Expression {
	return jp.f.Undef(jp.l, lpos(null), llen(null))
}

func (jp *transformer) transformNumber(jsNum *ast.NumberLiteral) parser.Expression {
	val := eval.Wrap(jp.c, jsNum.Value)
	if fp, ok := val.(*types.FloatValue); ok {
		return jp.f.Float(fp.Float(), jp.l, lpos(jsNum), llen(jsNum))
	}
	if in, ok := val.(*types.IntegerValue); ok {
		radix := 10
		lit := jsNum.Literal
		if len(lit) >= 2 {
			if lit[0] == '0' {
				if lit[1] == 'x' || lit[1] == 'X' {
					radix = 16
				} else {
					radix = 8
				}
			}
		}
		return jp.f.Integer(in.Int(), radix, jp.l, lpos(jsNum), llen(jsNum))
	}
	panic(jp.error(jsNum, EVAL_JS_INVALID_NUMBER, issue.H{`src`: jsNum.Literal}))
}

func (jp *transformer) transformObject(jsObj *ast.ObjectLiteral) parser.Expression {
	hel := make([]parser.Expression, len(jsObj.Value))
	for i, jsProp := range jsObj.Value {
		hel[i] = jp.transformProperty(&jsProp)
	}
	return jp.f.Hash(hel, jp.l, lpos(jsObj), llen(jsObj))
}

func (jp *transformer) transformParameters(params *ast.ParameterList) []parser.Expression {
	el := make([]parser.Expression, len(params.List))
	for i, id := range params.List {
		el[i] = jp.f.Parameter(id.Name, nil, nil, false, jp.l, lpos(id), llen(id))
	}
	return el
}

func (jp *transformer) transformProgram(program *ast.Program) parser.Expression {
	b := program.Body
	bl := len(b)
	if bl == 0 {
		return jp.f.Nop(jp.l, 0, 0)
	}

	defs := jp.transformDeclarations(program.DeclarationList, ``, []parser.Definition{})
	stmts := []parser.Expression{}
	for _, stmt := range program.Body {
		switch stmt.(type) {
		case *ast.FunctionStatement:
			// Already dealt with when transforming declarations
		default:
			stmts = jp.transformStatement(stmt, stmts)
		}
	}
	return jp.f.Program(
		jp.exprsAsBody(stmts),
		defs,
		jp.l,
		lpos(program),
		llen(program))
}

func (jp *transformer) transformProperty(jsProp *ast.Property) parser.Expression {
	exPos := lpos(jsProp.Value)
	keyLen := len(jsProp.Key)
	pStart := exPos - (keyLen + 2) // Approximation.
	return jp.f.KeyedEntry(
		jp.f.String(jsProp.Key, jp.l, pStart, keyLen),
		jp.transformExpression(jsProp.Value),
		jp.l, pStart, llen(jsProp.Value)+keyLen+2)
}

func (jp *transformer) transformRegexp(jsRx *ast.RegExpLiteral) parser.Expression {
	return jp.f.Regexp(jsRx.Value, jp.l, lpos(jsRx), llen(jsRx))
}

func (jp *transformer) transformReturn(jsReturn *ast.ReturnStatement) parser.Expression {
	fn := `return`
	if jp.lambda {
		fn = `next`
	}
	return jp.f.CallNamed(jp.f.QualifiedName(fn, jp.l, lpos(jsReturn), 6), false,
		[]parser.Expression{jp.transformExpression(jsReturn.Argument)}, nil, jp.l, lpos(jsReturn), llen(jsReturn))
}

func (jp *transformer) transformSequence(jpSeq *ast.SequenceExpression) parser.Expression {
	return jp.exprsAsBody(jp.transformExpressions(jpSeq.Sequence))
}

func (jp *transformer) transformStatement(stmt ast.Statement, el []parser.Expression) []parser.Expression {
	if stmt == nil {
		return el
	}
	switch stmt.(type) {
	case *ast.BranchStatement:
		return append(el, jp.transformBranch(stmt.(*ast.BranchStatement)))
	case *ast.BlockStatement:
		return jp.transformBlock(stmt.(*ast.BlockStatement), el)
	case *ast.DoWhileStatement:
		return append(el, jp.transformDoWhileStatement(stmt.(*ast.DoWhileStatement)))
	case *ast.EmptyStatement:
		return append(el, jp.transformEmpty(stmt.(*ast.EmptyStatement)))
	case *ast.ExpressionStatement:
		return append(el, jp.transformExpression(stmt.(*ast.ExpressionStatement).Expression))
	case *ast.ForStatement:
		return append(el, jp.transformFor(stmt.(*ast.ForStatement)))
	case *ast.ForInStatement:
		return append(el, jp.transformForIn(stmt.(*ast.ForInStatement)))
	case *ast.IfStatement:
		return append(el, jp.transformIf(stmt.(*ast.IfStatement)))
	case *ast.ReturnStatement:
		return append(el, jp.transformReturn(stmt.(*ast.ReturnStatement)))
	case *ast.SwitchStatement:
		return append(el, jp.transformSwitchStatement(stmt.(*ast.SwitchStatement)))
	case *ast.VariableStatement:
		return jp.transformVariableStatement(stmt.(*ast.VariableStatement), el)
	case *ast.WhileStatement:
		return append(el, jp.transformWhileStatement(stmt.(*ast.WhileStatement)))
	}
	panic(eval.Error(EVAL_JS_UNHANDLED_STATEMENT, issue.H{`stmt`: stmt}))
}

func (jp *transformer) transformString(jsStr *ast.StringLiteral) parser.Expression {
	return jp.f.String(jsStr.Value, jp.l, lpos(jsStr), llen(jsStr))
}

// transformSwitchStatement transforms the switch/case into a for statement:
//
// switch(<expr>) case 1: <stmts>, case 2: <stmts>
//
// becomes
//
// for(var js::switch_discriminant = <expr>;;) { if(js::switch_discriminant === 1) { <stmts1> } if(js::switch_discriminant === 2) { <stmts2> }...break; }
func (jp *transformer) transformSwitchStatement(jsSwitch *ast.SwitchStatement) parser.Expression {
	cases := make([]parser.Expression, len(jsSwitch.Body))
	for i, jsCase := range jsSwitch.Body {
		cases[i] = jp.transformCase(jsCase)
	}
	return NewSwitchExpression(
		jp.transformExpression(jsSwitch.Discriminant),
		cases,
		jp.l,
		lpos(jsSwitch), llen(jsSwitch))
}

func (jp *transformer) transformThis(jsThis *ast.ThisExpression) parser.Expression {
	return NewThisExpression(jp.l, lpos(jsThis), llen(jsThis))
}

func (jp *transformer) transformVariable(jsVar *ast.VariableExpression) parser.Expression {
	n := jp.f.QualifiedName(jsVar.Name, jp.l, lpos(jsVar), len(jsVar.Name))
	v := jp.transformExpression(jsVar.Initializer)
	return jp.f.Assignment(`=`, jp.f.Variable(n, jp.l, n.ByteOffset(), n.ByteLength()), v, jp.l, lpos(jsVar), llen(jsVar))
}

func (jp *transformer) transformUnary(jsUnary *ast.UnaryExpression) parser.Expression {
	or := jp.transformExpression(jsUnary.Operand)
	if v, ok := or.(*parser.VariableExpression); ok {
		switch jsUnary.Operator {
		case token.INCREMENT:
			return NewUnaryNumericExpression(v, jsUnary.Postfix, true, jp.l, lpos(jsUnary), llen(jsUnary))
		case token.DECREMENT:
			return NewUnaryNumericExpression(v, jsUnary.Postfix, false, jp.l, lpos(jsUnary), llen(jsUnary))
		}
	}
	panic(eval.Error(EVAL_JS_UNHANDLED_EXPRESSION, issue.H{`expr`: jsUnary}))
}

func (jp *transformer) transformVariableStatement(jsVar *ast.VariableStatement, el []parser.Expression) []parser.Expression {
	for _, s := range jsVar.List {
		el = append(el, jp.transformExpression(s))
	}
	return el
}

func (jp *transformer) transformWhileStatement(jsWhile *ast.WhileStatement) parser.Expression {
	return NewForExpression(
		jp.f.Nop(jp.l, lpos(jsWhile), 0),
		jp.transformExpression(jsWhile.Test),
		jp.f.Nop(jp.l, lpos(jsWhile), 0),
		jp.exprsAsBody(jp.transformStatement(jsWhile.Body, []parser.Expression{})),
		jp.l,
		lpos(jsWhile), llen(jsWhile))
}

func lpos(n ast.Node) int {
	return int(n.Idx0())
}

func llen(n ast.Node) int {
	return int(n.Idx1() - n.Idx0())
}
