package yamlparser

import (
	"bytes"
	"fmt"
	"io"
	"strconv"

	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-parser/parser"
	"github.com/puppetlabs/go-parser/validator"
	"gopkg.in/yaml.v2"
)

type yamlParser struct {
	c eval.Context
	l *parser.Locator
	f parser.ExpressionFactory
}

func NewYAMLParser(c eval.Context, filename string, content []byte) *yamlParser {
	yp := &yamlParser{c, parser.NewLocator(filename, string(content)), parser.DefaultFactory()}
	return yp
}

func EvaluateYaml(c eval.Context, filename string, content []byte) eval.PValue {
	ms := make(yaml.MapSlice, 0)
	err := yaml.Unmarshal(content, &ms)
	if err != nil {
		panic(eval.Error(c, eval.EVAL_PARSE_ERROR, issue.H{`language`: `YAML`, `detail`: err.Error()}))
	}
	yp := &yamlParser{c, parser.NewLocator(filename, string(content)), parser.DefaultFactory()}
	expr := yp.ParseYaml([]string{}, ms, true)
	fmt.Println(expr.ToPN())
	return c.Evaluate(expr)
}

func (yp *yamlParser) ParseYaml(path []string, ms yaml.MapSlice, top bool) parser.Expression {
	es := make([]parser.Expression, len(ms))

	// Copy path and make room for key
	plen := len(path)
	lp := make([]string, plen, plen+1)
	copy(lp, path)
	for i, mi := range ms {
		n, ok := mi.Key.(string)
		if !ok {
			n = fmt.Sprintf(`%v`, mi.Key)
		}
		lp = append(lp, n)
		if top {
			es[i] = yp.parseMapItem(lp, n, mi.Value)
		} else {
			es[i] = yp.f.KeyedEntry(
				yp.parseValue(path, mi.Key, false),
				yp.parseValue(path, mi.Value, false),
				yp.l, 0, 0)
		}
		lp = lp[:plen]
	}
	if top {
		if len(es) == 1 {
			return es[0]
		}
		return yp.f.Array(es, yp.l, 0, 0)
	}

	if len(es) == 1 {
		ke := es[0].(*parser.KeyedEntry)
		if key, ok := ke.Key().(*parser.LiteralString); ok && key.StringValue() == `_eval` {
			if str, ok := ke.Value().(*parser.LiteralString); ok {
				return yp.c.ParseAndValidate(yp.l.File(), str.StringValue(), true)
			}
			if str, ok := ke.Value().(*parser.ConcatenatedString); ok {
				bld := bytes.NewBufferString(``)
				for _, s := range str.Segments() {
					strtemp := yp.c.Evaluate(s).(*types.StringValue).String()
					bld.WriteString(strtemp)
				}
				return yp.c.ParseAndValidate(yp.l.File(), bld.String(), true)
			}
			panic(eval.Error(yp.c, EVAL_YAML_ILLEGAL_TYPE, issue.H{`path`: path, `key`: key, `expected`: `String`, `actual`: ke.Value().Label()}))
		}
	}
	return yp.f.Hash(es, yp.l, 0, 0)
}

func (yp *yamlParser) parseMapItem(path []string, key string, value interface{}) (expr parser.Expression) {
	switch key {
	case `parallel`, `sequential`:
		// Will return literal array for multiple expressions which in turn is interpreted as
		// parallel by the evaluator
		expr = yp.parseValue(path, value, true)
		if ll, ok := expr.(*parser.LiteralList); ok {
			return yp.f.Access(yp.f.QualifiedName(key, yp.l, 0, 0), ll.Elements(), yp.l, 0, 0)
		}
		panic(eval.Error(yp.c, EVAL_YAML_ILLEGAL_TYPE, issue.H{`path`: path, `key`: key, `expected`: `LiteralList`, `actual`: expr.Label()}))
	case `resources`:
		expr = yp.parseValue(path, value, false)
		if hash, ok := expr.(*parser.LiteralHash); ok {
			// Resources are keyed by the resource type. Each value underneath is a
			// hash keyed by titles. The special '_default' title holds default
			// values for all titles
			re := make([]parser.Expression, len(hash.Entries()))
			for i, ev := range hash.Entries() {
				entry := ev.(*parser.KeyedEntry)
				tv := entry.Key()
				name, ok := tv.(*parser.QualifiedName)
				if !ok {
					panic(eval.Error(yp.c, EVAL_YAML_ILLEGAL_TYPE, issue.H{`path`: path, `key`: key, `expected`: `QualifiedName`, `actual`: tv.Label()}))
				}
				tv = entry.Value()
				rHash, ok := tv.(*parser.LiteralHash)
				if !ok {
					panic(eval.Error(yp.c, EVAL_YAML_ILLEGAL_TYPE, issue.H{`path`: path, `key`: tv, `expected`: `LiteralHash`, `actual`: tv.Label()}))
				}
				re[i] = yp.resourceExpression(path, name, rHash)
			}
			expr = yp.f.Array(re, yp.l, 0, 0)
		} else {
			panic(eval.Error(yp.c, EVAL_YAML_ILLEGAL_TYPE, issue.H{`path`: path, `key`: key, `expected`: `LiteralHash`, `actual`: expr.Label()}))
		}
	case `functions`:
		expr = yp.parseValue(path, value, false)
	default:
		panic(eval.Error(yp.c, EVAL_YAML_UNRECOGNIZED_TOP_CONSTRUCT, issue.H{`path`: path, `key`: key}))
	}
	return expr
}

// Re-escape characters and add double quotes to turn this into a Puppet double qouted string
// that can be evaluated to satisfy interpolations
func requote(s string) string {
	b := bytes.NewBufferString(``)
	b.WriteByte('"')
	for _, c := range s {
		switch c {
		case '\t':
			io.WriteString(b, `\t`)
		case '\n':
			io.WriteString(b, `\n`)
		case '\r':
			io.WriteString(b, `\r`)
		case '"':
			io.WriteString(b, `\"`)
		case '\\':
			io.WriteString(b, `\\`)
		default:
			if c < 0x20 {
				fmt.Fprintf(b, `\u{%X}`, c)
			} else {
				b.WriteRune(c)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

func (yp *yamlParser) parseValue(path []string, value interface{}, top bool) parser.Expression {
	if value == nil {
		return yp.f.Undef(yp.l, 0, 0)
	}

	switch value.(type) {
	case yaml.MapSlice:
		return yp.ParseYaml(path, value.(yaml.MapSlice), top)
	case []interface{}:
		vs := value.([]interface{})
		exprs := make([]parser.Expression, len(vs))
		// Copy path and make room for index
		plen := len(path)
		lp := make([]string, plen, plen+1)
		copy(lp, path)
		for i, v := range vs {
			lp = append(lp, strconv.Itoa(i))
			exprs[i] = yp.parseValue(lp, v, top)
			lp = lp[:plen]
		}
		return yp.f.Array(exprs, yp.l, 0, 0)
	case string:
		// Treat the string as a Puppet string
		s := value.(string)
		if validator.CLASSREF_DECL.MatchString(s) {
			// Can't distinguish this from a quoted string
			return yp.f.QualifiedName(s, yp.l, 0, 0)
		}
		return yp.c.ParseAndValidate(yp.l.File(), requote(value.(string)), true)
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

// convert name and hash into a resource expression with bodies
func (yp *yamlParser) resourceExpression(path []string, name *parser.QualifiedName, hash *parser.LiteralHash) parser.Expression {
	plen := len(path)
	lp := make([]string, plen, plen+2)
	copy(lp, path)
	lp = append(lp, name.Name())

	var defaultAttrs []parser.Expression

	entries := hash.Entries()
	bodies := make([]parser.Expression, 0, len(entries))
	for _, ev := range entries {
		entry := ev.(*parser.KeyedEntry)
		title := entry.Key()
		tn := ``
		switch title.(type) {
		case *parser.LiteralString:
			tn = title.(*parser.LiteralString).StringValue()
		case *parser.QualifiedName:
			tn = title.(*parser.QualifiedName).Name()
		case *parser.QualifiedReference:
			tn = title.(*parser.QualifiedReference).Name()
		default:
			tn = `complex title`
		}
		attrOps, ok := entry.Value().(*parser.LiteralHash)
		if !ok {
			panic(eval.Error(yp.c, EVAL_YAML_ILLEGAL_TYPE, issue.H{`path`: lp, `key`: tn, `expected`: `LiteralHash`, `actual`: entry.Value().Label()}))
		}

		lp = append(lp, tn)
		attrs := yp.attributeOperations(lp, attrOps)
		if tn == `_defaults` {
			defaultAttrs = attrs
		} else {
			bodies = append(bodies, yp.f.ResourceBody(title, attrs, yp.l, 0, 0))
		}
		lp = lp[:len(lp)-1]
	}
	if defaultAttrs != nil {
		// Amend all bodies with defaults
		for i, body := range bodies {
			bodies[i] = yp.amendWithDefaults(body.(*parser.ResourceBody), defaultAttrs)
		}
	}
	return yp.f.Resource(parser.REGULAR, name, bodies, yp.l, 0, 0)
}

func (yp *yamlParser) amendWithDefaults(body *parser.ResourceBody, defaultAttrs []parser.Expression) parser.Expression {
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

// Convert literal hash into attribute operations
func (yp *yamlParser) attributeOperations(path []string, hash *parser.LiteralHash) []parser.Expression {
	entries := hash.Entries()
	attrs := make([]parser.Expression, len(entries))
	for i, ev := range entries {
		entry := ev.(*parser.KeyedEntry)
		name := entry.Key()
		tn := ``
		switch name.(type) {
		case *parser.LiteralString:
			tn = name.(*parser.LiteralString).StringValue()
		case *parser.QualifiedName:
			tn = name.(*parser.QualifiedName).Name()
		default:
			panic(eval.Error(yp.c, EVAL_YAML_ILLEGAL_TYPE, issue.H{`path`: path, `key`: path, `expected`: `Name`, `actual`: name.Label()}))
		}
		if tn == `*=>` {
			attrs[i] = yp.f.AttributesOp(entry.Value(), yp.l, 0, 0)
		} else {
			attrs[i] = yp.f.AttributeOp(`=>`, tn, entry.Value(), yp.l, 0, 0)
		}
	}
	return attrs
}
