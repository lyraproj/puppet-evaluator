package resource

import (
	"fmt"
	"strings"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-parser/parser"
)

// FindNode returns the node that contains a given resource reference
func FindNode(c eval.Context, v eval.PValue) (Node, bool) {
	ref, err := reference(v)
	if err != nil {
		return nil, false
	}
	cn := getCurrentNode(c)
	if cn == nil {
		return nil, false
	}

	return cn.findResource(c, ref)
}

// Reference returns the string T[<title>] where T is the lower case name of a resource type
// and <title> is the unique title of the instance that is referenced
func Reference(value eval.PValue) string {
	n, err := reference(value)
	if err != nil {
		panic(err)
	}
	return n
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

func getTitles(c eval.Context, expr parser.Expression, titles []string) []string {
	switch expr.(type) {
	case *parser.LiteralList:
		for _, e := range expr.(*parser.LiteralList).Elements() {
			titles = getTitles(c, e, titles)
		}
	case *parser.LiteralString:
		titles = append(titles, expr.(*parser.LiteralString).StringValue())
	case *parser.QualifiedName:
		titles = append(titles, expr.(*parser.QualifiedName).Name())
	case *parser.QualifiedReference:
		titles = append(titles, expr.(*parser.QualifiedReference).Name())
	case parser.LiteralValue:
		titles = append(titles, c.Call(`new`, []eval.PValue{types.DefaultStringType(), c.Evaluate(expr)}, nil).String())
	}
	return titles
}

func reference(value eval.PValue) (string, issue.Reported) {
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
