package evaluator

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/eval"
	"github.com/lyraproj/pcore/loader"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/puppet-evaluator/pdsl"
	"github.com/lyraproj/puppet-parser/parser"
)

func init() {
	loader.SmartPathFactories[eval.PuppetFunctionPath] = newPuppetFunctionPath
	loader.SmartPathFactories[eval.PuppetDataTypePath] = newPuppetDataTypePath
	loader.SmartPathFactories[eval.PlanPath] = newPuppetPlanPath
	loader.SmartPathFactories[eval.TaskPath] = newPuppetTaskPath
}

func newPuppetDataTypePath(ml eval.ModuleLoader, moduleNameRelative bool) loader.SmartPath {
	return loader.NewSmartPath(`types`, `.pp`, ml, eval.NsType, moduleNameRelative, false, InstantiatePuppetType)
}

func newPuppetFunctionPath(ml eval.ModuleLoader, moduleNameRelative bool) loader.SmartPath {
	return loader.NewSmartPath(`functions`, `.pp`, ml, eval.NsFunction, moduleNameRelative, false, InstantiatePuppetFunction)
}

func newPuppetPlanPath(ml eval.ModuleLoader, moduleNameRelative bool) loader.SmartPath {
	return loader.NewSmartPath(`plans`, `.pp`, ml, eval.NsPlan, moduleNameRelative, false, InstantiatePuppetPlan)
}

func newPuppetTaskPath(ml eval.ModuleLoader, moduleNameRelative bool) loader.SmartPath {
	return loader.NewSmartPath(`tasks`, ``, ml, eval.NsTask, moduleNameRelative, true, InstantiatePuppetTask)
}

func InstantiatePuppetActivityFromFile(ctx eval.Context, loader loader.ContentProvidingLoader, file string) eval.TypedName {
	ec := ctx.(pdsl.EvaluationContext)
	content := string(loader.GetContent(ctx, file))
	expr := ec.ParseAndValidate(file, content, false)
	name := `<any name>`
	fd, ok := getDefinition(expr, eval.NsActivity, name).(parser.NamedDefinition)
	if !ok {
		panic(ctx.Error(expr, eval.NoDefinition, issue.H{`source`: expr.File(), `type`: eval.NsActivity, `name`: name}))
	}
	ec.AddDefinitions(expr)
	ec.ResolveDefinitions()
	return eval.NewTypedName(eval.NsActivity, fd.Name())
}

func InstantiatePuppetFunction(ctx eval.Context, loader loader.ContentProvidingLoader, tn eval.TypedName, sources []string) {
	instantiatePuppetFunction(ctx, loader, tn, sources)
}

func InstantiatePuppetPlan(ctx eval.Context, loader loader.ContentProvidingLoader, tn eval.TypedName, sources []string) {
	instantiatePuppetFunction(ctx, loader, tn, sources)
}

func instantiatePuppetFunction(ctx eval.Context, loader loader.ContentProvidingLoader, tn eval.TypedName, sources []string) {
	ec := ctx.(pdsl.EvaluationContext)
	source := sources[0]
	content := string(loader.GetContent(ctx, source))
	expr := ec.ParseAndValidate(source, content, false)
	name := tn.Name()
	fd, ok := getDefinition(expr, tn.Namespace(), name).(parser.NamedDefinition)
	if !ok {
		panic(ctx.Error(expr, eval.NoDefinition, issue.H{`source`: expr.File(), `type`: tn.Namespace(), `name`: name}))
	}
	if !strings.EqualFold(fd.Name(), name) {
		panic(ctx.Error(expr, eval.WrongDefinition, issue.H{`source`: expr.File(), `type`: tn.Namespace(), `expected`: name, `actual`: fd.Name()}))
	}
	ec.AddDefinitions(expr)
	ec.ResolveDefinitions()
}

func InstantiatePuppetType(ctx eval.Context, loader loader.ContentProvidingLoader, tn eval.TypedName, sources []string) {
	ec := ctx.(pdsl.EvaluationContext)
	content := string(loader.GetContent(ctx, sources[0]))
	expr := ec.ParseAndValidate(sources[0], content, false)
	name := tn.Name()
	def := getDefinition(expr, eval.NsType, name)
	var tdn string
	switch def := def.(type) {
	case *parser.TypeAlias:
		tdn = def.Name()
	case *parser.TypeDefinition:
		tdn = def.Name()
	case *parser.TypeMapping:
		tdn = def.Type().Label()
	default:
		panic(ctx.Error(expr, eval.NoDefinition, issue.H{`source`: expr.File(), `type`: eval.NsType, `name`: name}))
	}
	if !strings.EqualFold(tdn, name) {
		panic(ctx.Error(expr, eval.WrongDefinition, issue.H{`source`: expr.File(), `type`: eval.NsType, `expected`: name, `actual`: tdn}))
	}
	ec.AddDefinitions(expr)
	ec.ResolveDefinitions()
}

func InstantiatePuppetTask(ctx eval.Context, loader loader.ContentProvidingLoader, tn eval.TypedName, sources []string) {
	name := tn.Name()
	metadata := ``
	taskSource := ``
	for _, sourceRef := range sources {
		if strings.HasSuffix(sourceRef, `.json`) {
			metadata = sourceRef
		} else if taskSource == `` {
			taskSource = sourceRef
		} else {
			panic(eval.Error(pdsl.TaskTooManyFiles, issue.H{`name`: name, `directory`: filepath.Dir(sourceRef)}))
		}
	}

	if taskSource == `` {
		panic(eval.Error(pdsl.TaskNoExecutableFound, issue.H{`name`: name, `directory`: filepath.Dir(sources[0])}))
	}
	task := createTask(ctx, loader, name, taskSource, metadata)
	origin := metadata
	if origin == `` {
		origin = taskSource
	}
	loader.(eval.DefiningLoader).SetEntry(tn, eval.NewLoaderEntry(task, issue.NewLocation(origin, 0, 0)))
}

func createTask(ctx eval.Context, loader loader.ContentProvidingLoader, name, taskSource, metadata string) eval.Value {
	if metadata == `` {
		return createTaskFromHash(ctx, name, taskSource, map[string]interface{}{})
	}
	jsonText := loader.GetContent(ctx, metadata)
	var parsedValue interface{}
	d := json.NewDecoder(bytes.NewReader(jsonText))
	d.UseNumber()
	if err := d.Decode(&parsedValue); err != nil {
		panic(eval.Error(pdsl.TaskBadJson, issue.H{`path`: metadata, `detail`: err}))
	}
	if jo, ok := parsedValue.(map[string]interface{}); ok {
		return createTaskFromHash(ctx, name, taskSource, jo)
	}
	panic(eval.Error(pdsl.TaskNotJsonObject, issue.H{`path`: metadata}))
}

func createTaskFromHash(ctx eval.Context, name, taskSource string, hash map[string]interface{}) eval.Value {
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
								paramHash[`type`] = ctx.ParseType2(s)
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

	if taskCtor, ok := eval.Load(ctx, eval.NewTypedName(eval.NsConstructor, `Task`)); ok {
		return taskCtor.(eval.Function).Call(ctx, nil, types.WrapStringToInterfaceMap(ctx, arguments))
	}
	panic(eval.Error(pdsl.TaskInitializerNotFound, issue.NO_ARGS))
}

// Extract a single Definition and return it. Will fail and report an error unless the program contains
// only one Definition
func getDefinition(expr parser.Expression, ns eval.Namespace, name string) parser.Definition {
	if p, ok := expr.(*parser.Program); ok {
		if b, ok := p.Body().(*parser.BlockExpression); ok {
			switch len(b.Statements()) {
			case 0:
			case 1:
				if d, ok := b.Statements()[0].(parser.Definition); ok {
					return d
				}
			default:
				panic(eval.Error2(expr, pdsl.NotOnlyDefinition, issue.H{`source`: expr.File(), `type`: ns, `name`: name}))
			}
		}
	}
	panic(eval.Error2(expr, eval.NoDefinition, issue.H{`source`: expr.File(), `type`: ns, `name`: name}))
}
