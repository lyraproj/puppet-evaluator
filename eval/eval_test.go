package eval

import (
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-parser/issue"
	"github.com/puppetlabs/go-parser/parser"
	"github.com/puppetlabs/go-parser/validator"
	"testing"
)

func TestIT(t *testing.T) {
	x := PValue(WrapInteger(32))
	switch x.(type) {
	case PValue:
		t.Log("Is PValue")
	default:
		t.Error("Not PValue")
	}

	switch x.Type().(type) {
	case *IntegerType:
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
    `, false, false)

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

	e := NewEvaluator(NewParentedLoader(StaticLoader()), NewStdLogger())
	e.AddDefinitions(prog)
	v, err := e.Evaluate(prog, NewScope(), nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	ta, ok := v.(*TypeAliasType)
	if !ok {
		t.Fatalf(`Eval did not return a TypeAliasType. It returned a %T`, v)
	}
	t.Logf("Successfully created a %T", ta)
	rt := ta.ResolvedType()
	va, ok := rt.(*VariantType)
	if !ok {
		t.Fatalf(`Not alias for Variant. Is %T`, va)
	}
	first := va.Types()[0]
	_, ok = first.(*StringType)
	if !ok {
		t.Fatalf(`First in Variant is not string. Is %T`, first)
	}
}
