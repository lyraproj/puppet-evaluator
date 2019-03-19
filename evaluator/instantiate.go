package evaluator

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/loader"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/puppet-evaluator/pdsl"
	"github.com/lyraproj/puppet-parser/parser"
)

func init() {
	loader.SmartPathFactories[px.PuppetFunctionPath] = newPuppetFunctionPath
	loader.SmartPathFactories[px.PlanPath] = newPuppetPlanPath
	loader.SmartPathFactories[px.TaskPath] = newPuppetTaskPath
}

func newPuppetFunctionPath(ml px.ModuleLoader, moduleNameRelative bool) loader.SmartPath {
	return loader.NewSmartPath(`functions`, `.pp`, ml, []px.Namespace{px.NsFunction}, moduleNameRelative, false, InstantiatePuppetFunction)
}

func newPuppetPlanPath(ml px.ModuleLoader, moduleNameRelative bool) loader.SmartPath {
	return loader.NewSmartPath(`plans`, `.pp`, ml, []px.Namespace{px.NsPlan}, moduleNameRelative, false, InstantiatePuppetPlan)
}

func newPuppetTaskPath(ml px.ModuleLoader, moduleNameRelative bool) loader.SmartPath {
	return loader.NewSmartPath(`tasks`, ``, ml, []px.Namespace{px.NsTask}, moduleNameRelative, true, InstantiatePuppetTask)
}

func InstantiatePuppetActivityFromFile(ctx px.Context, loader loader.ContentProvidingLoader, file string) px.TypedName {
	ec := ctx.(pdsl.EvaluationContext)
	content := string(loader.GetContent(ctx, file))
	expr := ec.ParseAndValidate(file, content, false)
	name := `<any name>`
	fd, ok := getDefinition(expr, px.NsActivity, name).(parser.NamedDefinition)
	if !ok {
		panic(ctx.Error(expr, px.NoDefinition, issue.H{`source`: expr.File(), `type`: px.NsActivity, `name`: name}))
	}
	ec.AddDefinitions(expr)
	ec.ResolveDefinitions()
	return px.NewTypedName(px.NsActivity, fd.Name())
}

func InstantiatePuppetFunction(ctx px.Context, loader loader.ContentProvidingLoader, tn px.TypedName, sources []string) {
	instantiatePuppetFunction(ctx, loader, tn, sources)
}

func InstantiatePuppetPlan(ctx px.Context, loader loader.ContentProvidingLoader, tn px.TypedName, sources []string) {
	instantiatePuppetFunction(ctx, loader, tn, sources)
}

func instantiatePuppetFunction(ctx px.Context, loader loader.ContentProvidingLoader, tn px.TypedName, sources []string) {
	ec := ctx.(pdsl.EvaluationContext)
	source := sources[0]
	content := string(loader.GetContent(ctx, source))
	expr := ec.ParseAndValidate(source, content, false)
	name := tn.Name()
	fd, ok := getDefinition(expr, tn.Namespace(), name).(parser.NamedDefinition)
	if !ok {
		panic(ctx.Error(expr, px.NoDefinition, issue.H{`source`: expr.File(), `type`: tn.Namespace(), `name`: name}))
	}
	if !strings.EqualFold(fd.Name(), name) {
		panic(ctx.Error(expr, px.WrongDefinition, issue.H{`source`: expr.File(), `type`: tn.Namespace(), `expected`: name, `actual`: fd.Name()}))
	}
	ec.AddDefinitions(expr)
	ec.ResolveDefinitions()
}

func InstantiatePuppetTask(ctx px.Context, loader loader.ContentProvidingLoader, tn px.TypedName, sources []string) {
	name := tn.Name()
	metadata := ``
	taskSource := ``
	for _, sourceRef := range sources {
		if strings.HasSuffix(sourceRef, `.json`) {
			metadata = sourceRef
		} else if taskSource == `` {
			taskSource = sourceRef
		} else {
			panic(px.Error(pdsl.TaskTooManyFiles, issue.H{`name`: name, `directory`: filepath.Dir(sourceRef)}))
		}
	}

	if taskSource == `` {
		panic(px.Error(pdsl.TaskNoExecutableFound, issue.H{`name`: name, `directory`: filepath.Dir(sources[0])}))
	}
	task := createTask(ctx, loader, name, taskSource, metadata)
	origin := metadata
	if origin == `` {
		origin = taskSource
	}
	loader.(px.DefiningLoader).SetEntry(tn, px.NewLoaderEntry(task, issue.NewLocation(origin, 0, 0)))
}

func createTask(ctx px.Context, loader loader.ContentProvidingLoader, name, taskSource, metadata string) px.Value {
	if metadata == `` {
		return createTaskFromHash(ctx, name, taskSource, map[string]interface{}{})
	}
	jsonText := loader.GetContent(ctx, metadata)
	var parsedValue interface{}
	d := json.NewDecoder(bytes.NewReader(jsonText))
	d.UseNumber()
	if err := d.Decode(&parsedValue); err != nil {
		panic(px.Error(pdsl.TaskBadJson, issue.H{`path`: metadata, `detail`: err}))
	}
	if jo, ok := parsedValue.(map[string]interface{}); ok {
		return createTaskFromHash(ctx, name, taskSource, jo)
	}
	panic(px.Error(pdsl.TaskNotJsonObject, issue.H{`path`: metadata}))
}

func createTaskFromHash(ctx px.Context, name, taskSource string, hash map[string]interface{}) px.Value {
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
								paramHash[`type`] = ctx.ParseType(s)
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

	if taskCtor, ok := px.Load(ctx, px.NewTypedName(px.NsConstructor, `Task`)); ok {
		return taskCtor.(px.Function).Call(ctx, nil, types.WrapStringToInterfaceMap(ctx, arguments))
	}
	panic(px.Error(pdsl.TaskInitializerNotFound, issue.NoArgs))
}

// Extract a single Definition and return it. Will fail and report an error unless the program contains
// only one Definition
func getDefinition(expr parser.Expression, ns px.Namespace, name string) parser.Definition {
	if p, ok := expr.(*parser.Program); ok {
		if b, ok := p.Body().(*parser.BlockExpression); ok {
			switch len(b.Statements()) {
			case 0:
			case 1:
				if d, ok := b.Statements()[0].(parser.Definition); ok {
					return d
				}
			default:
				panic(px.Error2(expr, pdsl.NotOnlyDefinition, issue.H{`source`: expr.File(), `type`: ns, `name`: name}))
			}
		}
	}
	panic(px.Error2(expr, px.NoDefinition, issue.H{`source`: expr.File(), `type`: ns, `name`: name}))
}
