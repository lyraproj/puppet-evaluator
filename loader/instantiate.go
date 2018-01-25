package loader

import (
	. "github.com/puppetlabs/go-evaluator/evaluator"
	. "github.com/puppetlabs/go-parser/parser"
	"strings"
	. "github.com/puppetlabs/go-parser/issue"
	"path/filepath"
	"encoding/json"
	"github.com/puppetlabs/go-evaluator/types"
	"bytes"
)

type Instantiator func(loader ContentProvidingLoader, tn TypedName, sources []string)

func InstantiatePuppetFunction(loader ContentProvidingLoader, tn TypedName, sources []string) {
	instantiatePuppetFunction(loader, tn, sources)
}

func InstantiatePuppetPlan(loader ContentProvidingLoader, tn TypedName, sources []string) {
	instantiatePuppetFunction(loader, tn, sources)
}

func instantiatePuppetFunction(loader ContentProvidingLoader, tn TypedName, sources []string) {
	source := sources[0]
	content := string(loader.GetContent(source))
	ctx := CurrentContext()
	expr := ctx.ParseAndValidate(source, content, false)
	name := tn.Name()
	fd, ok := getDefinition(expr, tn.Namespace(), name).(NamedDefinition)
	if !ok {
		panic(ctx.Error(expr, EVAL_NO_DEFINITION, H{`source`: expr.File(), `type`: tn.Namespace(), `name`: name}))
	}
	if strings.ToLower(fd.Name()) != strings.ToLower(name) {
		panic(ctx.Error(expr, EVAL_WRONG_DEFINITION, H{`source`: expr.File(), `type`: tn.Namespace(), `expected`: name, `actual`: fd.Name()}))
	}
	e := ctx.Evaluator()
	e.AddDefinitions(expr)
	e.ResolveDefinitions(ctx)
}

func InstantiatePuppetType(loader ContentProvidingLoader, tn TypedName, sources []string) {
	content := string(loader.GetContent(sources[0]))
	ctx := CurrentContext()
	expr := ctx.ParseAndValidate(sources[0], content, false)
	name := tn.Name()
	def := getDefinition(expr, TYPE, name)
	var tdn string
	switch def.(type) {
	case *TypeAlias:
		tdn = def.(*TypeAlias).Name()
	case *TypeDefinition:
		tdn = def.(*TypeDefinition).Name()
	case *TypeMapping:
		tdn = def.(*TypeMapping).Type().Label()
	default:
		panic(ctx.Error(expr, EVAL_NO_DEFINITION, H{`source`: expr.File(), `type`: TYPE, `name`: name}))
	}
	if strings.ToLower(tdn) != strings.ToLower(name) {
		panic(ctx.Error(expr, EVAL_WRONG_DEFINITION, H{`source`: expr.File(), `type`: TYPE, `expected`: name, `actual`: tdn}))
	}
	e := ctx.Evaluator()
	e.AddDefinitions(expr)
	e.ResolveDefinitions(ctx)
}

func InstantiatePuppetTask(loader ContentProvidingLoader, tn TypedName, sources []string) {
	name := tn.Name()
	metadata := ``
	taskSource := ``
	for _, sourceRef := range sources {
		if strings.HasSuffix(sourceRef, `.json`) {
			metadata = sourceRef
		} else if taskSource == `` {
			taskSource = sourceRef
		} else {
			panic(Error(EVAL_TASK_TOO_MANY_FILES, H{`name`: name, `directory`: filepath.Dir(sourceRef)}))
		}
	}

	if taskSource == `` {
		panic(Error(EVAL_TASK_NO_EXECUTABLE_FOUND, H{`name`: name, `directory`: filepath.Dir(sources[0])}))
	}
	task := createTask(CurrentContext(), loader, name, taskSource, metadata)
	origin := metadata
	if origin == `` {
		origin = taskSource
	}
	loader.(DefiningLoader).SetEntry(tn, NewLoaderEntry(task, origin))
}

func createTask(ctx EvalContext, loader ContentProvidingLoader, name, taskSource, metadata string) PValue {
  if metadata == `` {
  	return createTaskFromHash(ctx, name, taskSource, map[string]interface{}{})
  }
  jsonText := loader.GetContent(metadata)
  var parsedValue interface{}
  d := json.NewDecoder(bytes.NewReader(jsonText))
  d.UseNumber()
  if err := d.Decode(&parsedValue); err != nil {
  	panic(Error(EVAL_TASK_BAD_JSON, H{`path`: metadata, `detail`: err}))
  }
  if jo, ok := parsedValue.(map[string]interface{}); ok {
  	return createTaskFromHash(ctx, name, taskSource, jo)
  }
	panic(Error(EVAL_TASK_NOT_JSON_OBJECT, H{`path`: metadata}))
}

func createTaskFromHash(ctx EvalContext, name, taskSource string, hash map[string]interface{}) PValue {
	arguments := make(map[string]interface{}, 7)
	arguments[`name`] = types.WrapString(name)
	arguments[`executable`] = types.WrapString(taskSource)
	for key, value := range hash {
    if key == `parameters` || key == `output` {
	    if params, ok := value.(map[string]interface{}); ok {
	    	for _, param := range params {
	    		if paramHash, ok := param.(map[string]interface{}); ok {
				    if t, ok := paramHash[`type`]; ok {
				    	if s, ok := t.(string); ok {
						    paramHash[`type`] = ctx.ParseResolve(s)
					    }
				    } else {
					    paramHash[`type`] = types.DefaultDataType()
				    }
			    }
		    }
	    }
    }
    arguments[key] = value
	}

	if taskCtor, ok := Load(ctx.Loader(), NewTypedName(CONSTRUCTOR, `Task`)); ok {
		return taskCtor.(Function).Call(ctx, nil, types.WrapHash4(arguments))
	}
  panic(Error(EVAL_TASK_INITIALIZER_NOT_FOUND, NO_ARGS))
}

// Extract a single Definition and return it. Will fail and report an error unless the program contains
// only one Definition
func getDefinition(expr Expression, ns Namespace, name string) Definition {
	if p, ok := expr.(*Program); ok {
		if b, ok := p.Body().(*BlockExpression); ok {
			switch len(b.Statements()) {
			case 0:
			case 1:
				if d, ok := b.Statements()[0].(Definition); ok {
					return d
				}
			default:
				panic(Error2(expr, EVAL_NOT_ONLY_DEFINITION, H{`source`: expr.File(), `type`: ns, `name`: name}))
			}
		}
	}
	panic(Error2(expr, EVAL_NO_DEFINITION, H{`source`: expr.File(), `type`: ns, `name`: name}))
}