package yaml2ast

import (
	"fmt"
	"strconv"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-parser/parser"
	"gopkg.in/yaml.v2"
	"github.com/puppetlabs/go-parser/validator"
)

type transformer struct {
	c eval.Context
	l *parser.Locator
	f parser.ExpressionFactory
	p []string
	plen int
}

// YamlToAST parses and transforms the given yaml content into a Puppet AST. It will
// panic with an issue.Reported unless the parsing and transformation was succesful.
func YamlToAST(c eval.Context, filename string, content []byte) parser.Expression {
	ms := make(yaml.MapSlice, 0)
	err := yaml.Unmarshal(content, &ms)
	if err != nil {
		panic(eval.Error(c, eval.EVAL_PARSE_ERROR, issue.H{`language`: `YAML`, `detail`: err.Error()}))
	}
	yp := &transformer{c, parser.NewLocator(filename, string(content)), parser.DefaultFactory(), []string{filename}, 1}
	return yp.transformMap(ms, true)
}

// EvaluateYaml calls YamlToAST to parse and transform the given YAML content into
// a Puppet AST which is then evaluated by the eval.Evaluator obtained from the given
// eval.Context. The result of the evaluation is returned.
func EvaluateYaml(c eval.Context, filename string, content []byte) eval.PValue {
	return c.Evaluate(YamlToAST(c, filename, content))
}

// transformMap transforms the supplied yaml.MapSlice into a parser.Expression. It will
// panic with an issue.Reported unless the parsing was succesful.
func (yp *transformer) transformMap(ms yaml.MapSlice, top bool) parser.Expression {
	es := make([]parser.Expression, len(ms))

	// Copy path and make room for key
	for i, mi := range ms {
		if top {
			es[i] = yp.transformMapItem(mi)
		} else {
			es[i] = yp.f.KeyedEntry(
				yp.transformValue(mi.Key, false),
				yp.transformValue(mi.Value, false),
				yp.l, 0, 0)
		}
	}

	if top {
		if len(es) == 1 {
			return es[0]
		}
		return yp.f.Array(es, yp.l, 0, 0)
	}

	if len(es) == 1 {
		// Check if this is a one element hash with "_eval" key.
		ke := es[0].(*parser.KeyedEntry)
		if key, ok := ke.Key().(*parser.LiteralString); ok && key.StringValue() == `_eval` {
			yp.pushPath(key.StringValue())
			expr := yp.transformEvalValue(ke.Value())
			yp.popPath()
			return expr
		}
	}
	return yp.f.Hash(es, yp.l, 0, 0)
}

func (yp *transformer) transformMapItem(mi yaml.MapItem) (expr parser.Expression) {
	yp.pushPath(mi.Key)
	switch mi.Key {
	case `parallel`, `sequential`:
		expr = yp.transformValue(mi.Value, true)
		if ll, ok := expr.(*parser.LiteralList); ok {
			expr = yp.f.Access(yp.f.QualifiedName(mi.Key.(string), yp.l, 0, 0), ll.Elements(), yp.l, 0, 0)
		} else {
			panic(eval.Error(yp.c, EVAL_YAML_ILLEGAL_TYPE, issue.H{`path`: yp.path(), `expected`: `List`, `actual`: expr.Label()}))
		}

	case `block`:
		expr = yp.transformValue(mi.Value, true)
		if ll, ok := expr.(*parser.LiteralList); ok {
			expr = yp.f.Block(ll.Elements(), yp.l, 0, 0)
		} else {
			panic(eval.Error(yp.c, EVAL_YAML_ILLEGAL_TYPE, issue.H{`path`: yp.path(), `expected`: `List`, `actual`: expr.Label()}))
		}

	case `_eval`:
		expr = yp.transformEvalValue(yp.transformValue(mi.Value, true))

	default:
		// Resource expression. Like a hash but must be expressed as list with one association for each hash entry (due to
		// the unordered nature of YAML hash)
		tv := yp.transformValue(mi.Key, false)
		var name *parser.QualifiedName
		if s, ok := tv.(*parser.LiteralString); ok {
			if validator.CLASSREF_DECL.MatchString(s.StringValue()) {
				// Can't distinguish this from a quoted string
				name = yp.f.QualifiedName(s.StringValue(), yp.l, 0, 0).(*parser.QualifiedName)
			}
		}
		if name == nil {
			panic(eval.Error(yp.c, EVAL_YAML_ILLEGAL_TYPE, issue.H{`path`: yp.path(), `expected`: `Name`, `actual`: tv.Label()}))
		}
		expr = yp.resourceExpression(name, yp.transformOrderedHash(yp.transformValue(mi.Value, false)))
	}
	yp.popPath()
	return expr
}

func (yp *transformer) transformEvalValue(expr parser.Expression) parser.Expression {
	switch expr.(type) {
	case *parser.LiteralString:
		return yp.c.ParseAndValidate(yp.l.File(), expr.(*parser.LiteralString).StringValue(), true)
	case *parser.LiteralList, *parser.LiteralHash:
		return expr
	default:
		panic(eval.Error(yp.c, EVAL_YAML_ILLEGAL_TYPE, issue.H{`path`: yp.path(), `expected`: `String, List, or Hash`, `actual`: expr.Label()}))
	}
}

func (yp *transformer) transformValue(value interface{}, top bool) parser.Expression {
	if value == nil {
		return yp.f.Undef(yp.l, 0, 0)
	}

	switch value.(type) {
	case yaml.MapSlice:
		return yp.transformMap(value.(yaml.MapSlice), top)
	case []interface{}:
		vs := value.([]interface{})
		exprs := make([]parser.Expression, len(vs))
		for i, v := range vs {
			yp.pushPath(i)
			exprs[i] = yp.transformValue(v, top)
			yp.popPath()
		}
		return yp.f.Array(exprs, yp.l, 0, 0)
	case string:
		return yp.f.String(value.(string), yp.l, 0, 0)
	case bool:
		return yp.f.Boolean(value.(bool), yp.l, 0, 0)
	case float32:
		return yp.f.Float(float64(value.(float32)), yp.l, 0, 0)
	case float64:
		return yp.f.Float(value.(float64), yp.l, 0, 0)
	case int:
		return yp.f.Integer(int64(value.(int)), 10, yp.l, 0, 0)
	case int8:
		return yp.f.Integer(int64(value.(int8)), 10, yp.l, 0, 0)
	case int16:
		return yp.f.Integer(int64(value.(int16)), 10, yp.l, 0, 0)
	case int32:
		return yp.f.Integer(int64(value.(int32)), 10, yp.l, 0, 0)
	case int64:
		return yp.f.Integer(value.(int64), 10, yp.l, 0, 0)
	case uint:
		return yp.f.Integer(int64(value.(uint)), 10, yp.l, 0, 0)
	case uint8:
		return yp.f.Integer(int64(value.(uint8)), 10, yp.l, 0, 0)
	case uint16:
		return yp.f.Integer(int64(value.(uint16)), 10, yp.l, 0, 0)
	case uint32:
		return yp.f.Integer(int64(value.(uint32)), 10, yp.l, 0, 0)
	case uint64:
		return yp.f.Integer(int64(value.(uint64)), 10, yp.l, 0, 0)
	default:
		panic(fmt.Errorf(`Unknown type '%T' with value '%v'`, value, value))
	}
}

// Transform a list consisting of one element hashes into an ordered hash
func (yp *transformer) transformOrderedHash(expr parser.Expression) *parser.LiteralHash {
	ev, ok := expr.(*parser.LiteralList)
	if !ok {
		panic(eval.Error(yp.c, EVAL_YAML_ILLEGAL_TYPE, issue.H{`path`: yp.path(), `expected`: `List`, `actual`: expr.Label()}))
	}

	assocs := ev.Elements()
	sz := len(assocs)
	entries := make([]parser.Expression, sz)
	unique := make(map[string]bool, sz)
	for i, e := range assocs {
		yp.pushPath(i)
		assoc := yp.transformAssoc(e)
		key := ``
		key, ok = stringValue(assoc.Key())
		if !ok {
			if _, ok := assoc.Key().(*parser.ConcatenatedString); ok {
				// This will eventually evaluate to a string so it's OK. Use PN representation to form a unique key.
				ok = true
				key = assoc.Key().ToPN().String()
			} else {
				panic(eval.Error(yp.c, EVAL_YAML_ILLEGAL_TYPE, issue.H{`path`: yp.path(), `expected`: `String`, `actual`: assoc.Key().Label()}))
			}
		}
		if _, ok = unique[key]; ok {
			panic(eval.Error(yp.c, EVAL_YAML_DUPLICATE_KEY, issue.H{`path`: yp.path(), `key`: key}))
		}
		yp.popPath()
		unique[key] = true
		entries[i] = assoc
	}
	return yp.f.Hash(entries, yp.l, 0, 0).(*parser.LiteralHash)
}

func (yp *transformer) transformAssoc(expr parser.Expression) *parser.KeyedEntry {
	ev, ok := expr.(*parser.LiteralHash)
	if !ok && len(ev.Entries()) == 1 {
		panic(eval.Error(yp.c, EVAL_YAML_ILLEGAL_TYPE, issue.H{`path`: yp.path(), `expected`: `one element Hash`, `actual`: expr.Label()}))
	}
	return ev.Entries()[0].(*parser.KeyedEntry)
}

// convert name and hash into a resource expression with bodies
func (yp *transformer) resourceExpression(name *parser.QualifiedName, hash *parser.LiteralHash) parser.Expression {
	yp.pushPath(name.Name())

	var defaultAttrs []parser.Expression

	bodies := make([]parser.Expression, 0, len(hash.Entries()))
	for i, ev := range hash.Entries() {
		yp.pushPath(i)
		entry := ev.(*parser.KeyedEntry)
		title := entry.Key()
		tn, ok := stringValue(title)
		if !ok {
			tn = `complex title`
		}
		yp.pushPath(tn)

		attrOps, ok := entry.Value().(*parser.LiteralHash)
		if !ok {
			panic(eval.Error(yp.c, EVAL_YAML_ILLEGAL_TYPE, issue.H{`path`: yp.path(), `expected`: `LiteralHash`, `actual`: entry.Value().Label()}))
		}
		attrs := yp.attributeOperations(attrOps)
		if tn == `_defaults` {
			defaultAttrs = attrs
		} else {
			bodies = append(bodies, yp.f.ResourceBody(title, attrs, yp.l, 0, 0))
		}
		yp.popPath()
		yp.popPath()
	}
	if defaultAttrs != nil {
		// Amend all bodies with defaults
		for i, body := range bodies {
			bodies[i] = yp.amendWithDefaults(body.(*parser.ResourceBody), defaultAttrs)
		}
	}
	yp.popPath()
	return yp.f.Resource(parser.REGULAR, name, bodies, yp.l, 0, 0)
}

func (yp *transformer) amendWithDefaults(body *parser.ResourceBody, defaultAttrs []parser.Expression) parser.Expression {
	modified := false
	ops := body.Operations()
	for _, dflt := range defaultAttrs {
		if do, ok := dflt.(*parser.AttributeOperation); ok {
			found := false
			for _, attr := range ops {
				if ao, ok := attr.(*parser.AttributeOperation); ok {
					if do.Name() == ao.Name() {
						found = true
						break
					}
				}
			}
			if !found {
				ops = append(ops, dflt)
				modified = true
			}
		}
	}
	if modified {
		return yp.f.ResourceBody(body.Title(), ops, yp.l, 0, 0)
	}
	return body
}

// attributeOperations converts literal hash into attribute operations
func (yp *transformer) attributeOperations(hash *parser.LiteralHash) []parser.Expression {
	entries := hash.Entries()
	attrs := make([]parser.Expression, len(entries))
	for i, ev := range entries {
		yp.pushPath(i)
		entry := ev.(*parser.KeyedEntry)
		name := entry.Key()
		tn, ok := stringValue(name)
		if !ok {
			panic(eval.Error(yp.c, EVAL_YAML_ILLEGAL_TYPE, issue.H{`path`: yp.path(), `expected`: `String`, `actual`: name.Label()}))
		}
		if tn == `*=>` {
			attrs[i] = yp.f.AttributesOp(entry.Value(), yp.l, 0, 0)
		} else {
			attrs[i] = yp.f.AttributeOp(`=>`, tn, entry.Value(), yp.l, 0, 0)
		}
		yp.popPath()
	}
	return attrs
}

func (yp *transformer) pushPath(elem interface{}) {
	s := ``
	switch elem.(type) {
	case string:
		s = elem.(string)
	case int:
		s = strconv.Itoa(elem.(int))
	default:
		s = fmt.Sprintf(`%v`, elem)
	}
	if len(yp.p) > yp.plen {
		yp.p[yp.plen] = s
	} else {
		yp.p = append(yp.p, s)
	}
	yp.plen++
}

func (yp *transformer) popPath() {
	yp.plen--
}

func (yp *transformer) path() []string {
	return yp.p[0:yp.plen]
}

func stringValue(expr parser.Expression) (string, bool) {
	switch expr.(type) {
	case *parser.LiteralString:
		return expr.(*parser.LiteralString).StringValue(), true
	case *parser.QualifiedName:
		return expr.(*parser.QualifiedName).Name(), true
	case *parser.QualifiedReference:
		return expr.(*parser.QualifiedReference).Name(), true
	default:
		return ``, false
	}
}
