package impl

import (
	"testing"

	"github.com/puppetlabs/go-evaluator/eval"
	_ "github.com/puppetlabs/go-evaluator/loader"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-parser/issue"
	"github.com/puppetlabs/go-parser/parser"
	"github.com/puppetlabs/go-parser/validator"
)

func TestIT(t *testing.T) {
	x := eval.PValue(types.WrapInteger(32))
	switch x.(type) {
	case eval.PValue:
		t.Log("Is eval.PValue")
	default:
		t.Error("Not eval.PValue")
	}

	switch x.Type().(type) {
	case *types.IntegerType:
		t.Log("Type is IntegerType")
	default:
		t.Error("Not IntegerType")
	}
}

func TestMisc(t *testing.T) {
	parser := parser.CreateParser()
	prog, ex := parser.Parse(`testfile.pp`, `
    type MyType = Variant[String,Integer]
    MyType
    `, false)

	if ex != nil {
		t.Fatalf(ex.Error())
	}

	checker := validator.NewChecker(validator.STRICT_ERROR)
	checker.Validate(prog)
	issues := checker.Issues()
	if len(issues) > 0 {
		severity := issue.SEVERITY_IGNORE
		for _, issue := range issues {
			t.Log(issue.Error())
			if issue.Severity() > severity {
				severity = issue.Severity()
			}
		}
		if severity == issue.SEVERITY_ERROR {
			t.Fail()
		}
	}

	e := NewEvaluator(eval.NewParentedLoader(eval.StaticLoader()), eval.NewStdLogger())
	e.AddDefinitions(prog)
	v, err := e.Evaluate(prog, NewScope(), nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	ta, ok := v.(*types.TypeAliasType)
	if !ok {
		t.Fatalf(`Eval did not return a TypeAliasType. It returned a %T`, v)
	}
	t.Logf("Successfully created a %T", ta)
	rt := ta.ResolvedType()
	va, ok := rt.(*types.VariantType)
	if !ok {
		t.Fatalf(`Not alias for Variant. Is %T`, va)
	}
	first := va.Types()[0]
	_, ok = first.(*types.StringType)
	if !ok {
		t.Fatalf(`First in Variant is not string. Is %T`, first)
	}
}

func TestEmpty(t *testing.T) {
	parser := parser.CreateParser()
	prog, ex := parser.Parse(`testfile.pp`, ``, false)

	if ex != nil {
		t.Fatalf(ex.Error())
	}

	checker := validator.NewChecker(validator.STRICT_ERROR)
	checker.Validate(prog)
	issues := checker.Issues()
	if len(issues) > 0 {
		severity := issue.SEVERITY_IGNORE
		for _, issue := range issues {
			t.Log(issue.Error())
			if issue.Severity() > severity {
				severity = issue.Severity()
			}
		}
		if severity == issue.SEVERITY_ERROR {
			t.Fail()
		}
	}

	e := NewEvaluator(eval.NewParentedLoader(eval.StaticLoader()), eval.NewStdLogger())
	e.AddDefinitions(prog)
	v, err := e.Evaluate(prog, NewScope(), nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	_, ok := v.(*types.UndefValue)
	if !ok {
		t.Fatalf(`Eval did not return a undef. It returned a %T`, v)
	}
}
