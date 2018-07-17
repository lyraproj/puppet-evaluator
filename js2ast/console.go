package js2ast

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"io"
)

var consoleType eval.PType

type Console interface {
	eval.PuppetObject

	Assert(c eval.Context, assertion bool, message eval.PValue)

	Clear()

	Count(c eval.Context, label string)

	Error(c eval.Context, message eval.PValue)

	Group(c eval.Context, label string)

	GroupCollapsed(c eval.Context, label string)

	GroupEnd(c eval.Context)

	Info(c eval.Context, message eval.PValue)

	Log(c eval.Context, message eval.PValue)

	Table(tc eval.Context, ableData, tableColums eval.PValue)

	Time(c eval.Context, label string)

	TimeEnd(lc eval.Context, abel string)

	Trace(c eval.Context, label string)

	Warn(c eval.Context, message eval.PValue)
}

type console struct {
	name string
}

func NewConsole(name string) Console {
	return &console{name: name}
}

func (cs *console) String() string {
	return eval.ToString(cs)
}

func (cs *console) Equals(other interface{}, guard eval.Guard) bool {
	return cs == other
}

func (cs *console) ToString(bld io.Writer, format eval.FormatContext, g eval.RDetect) {
	types.ObjectToString(cs, format, bld, g)
}

func (cs *console) Type() eval.PType {
	return consoleType
}

func (cs *console) Call(c eval.Context, method string, args []eval.PValue, block eval.Lambda) (result eval.PValue, ok bool) {
	switch method {
	case `assert`:
		cs.Assert(c, args[0].(*types.BooleanValue).Bool(), args[1])
	case `clear`:
		cs.Clear()
	case `count`:
		cs.Count(c, optLabel(args))
	case `error`:
		cs.Error(c, args[0])
	case `group`:
		cs.Group(c, optLabel(args))
	case `group_collapsed`, `groupCollapsed`:
		cs.GroupCollapsed(c, optLabel(args))
	case `group_end`, `groupEnd`:
		cs.GroupEnd(c)
	case `info`:
		cs.Info(c, args[0])
	case `log`:
		cs.Log(c, args[0])
	case `table`:
		if len(args) == 2 {
			cs.Table(c, args[0], args[1])
		} else {
			cs.Table(c, args[0], eval.UNDEF)
		}
	case `time`:
		cs.Time(c, optLabel(args))
	case `time_end`, `timeEnd`:
		cs.TimeEnd(c, optLabel(args))
	case `trace`:
		cs.Trace(c, optLabel(args))
	case `warn`:
		cs.Warn(c, args[0])
	default:
		return nil, false
	}
	return eval.UNDEF, true
}

func optLabel(args []eval.PValue) string {
	if len(args) == 0 {
		return ``
	}
	return args[0].String()
}

func (cs *console) Get(c eval.Context, key string) (value eval.PValue, ok bool) {
	switch key {
	case `name`:
		return types.WrapString(cs.name), true
	default:
		return nil, false
	}
}

func (cs *console) InitHash() eval.KeyedValue {
	return types.SingletonHash2(`name`, types.WrapString(cs.name))
}

func (cs *console) Assert(c eval.Context, assertion bool, message eval.PValue) {
	if !assertion {
		cs.Error(c, message)
	}
}

func (cs *console) Clear() {
}

func (cs *console) Count(c eval.Context, label string) {
	// TODO implement Console.Count
}

func (cs *console) Error(c eval.Context, message eval.PValue) {
	c.Logger().Log(eval.ERR, message)
}

func (cs *console) Group(c eval.Context, label string) {
	// TODO implement Console.Group
}

func (cs *console) GroupCollapsed(c eval.Context, label string) {
	// TODO implement Console.GroupCollapsed
}

func (cs *console) GroupEnd(c eval.Context) {
	// TODO implement Console.GroupEnd
}

func (cs *console) Info(c eval.Context, message eval.PValue) {
	c.Logger().Log(eval.INFO, message)
}

func (cs *console) Log(c eval.Context, message eval.PValue) {
	c.Logger().Log(eval.NOTICE, message)
}

func (cs *console) Table(c eval.Context, tableData, tableColums eval.PValue) {
	// TODO implement Console.Table
}

func (cs *console) Time(c eval.Context, label string) {
	// TODO implement Console.Time
}

func (cs *console) TimeEnd(c eval.Context, label string) {
	// TODO implement Console.TimeEnd
}

func (cs *console) Trace(c eval.Context, label string) {
	// TODO implement Console.Trace
}

func (cs *console) Warn(c eval.Context, message eval.PValue) {
	c.Logger().Log(eval.WARNING, message)
}


func initConsole(c eval.Context) {
	eval.NewTypeAlias(`JS::Object`, `Hash[String,Variant[ScalarData,JS::Object]]`)
	eval.NewTypeAlias(`JS::ConsoleArg`, `Variant[ScalarData,JS::Object]`)

	consoleType = eval.NewObjectType(`JS::Console`, `{
		attributes => {
      name => String
    },
    functions => {
      assert => Callable[Boolean,JS::ConsoleArg],
      clear => Callable[],
      count => Callable[String,0,1],
      error => Callable[JS::ConsoleArg],
      group => Callable[String,0,1],
      groupCollapsed => Callable[String,0,1],
      groupEnd => Callable[],
      info => Callable[JS::ConsoleArg],
      log => Callable[JS::ConsoleArg],
      table => Callable[Variant[Array[JS::ConsoleArg],DataHash], Array[String]],
      time => Callable[String,0,1],
      timeEnd => Callable[String,0,1],
      trace => Callable[String,0,1],
      warn => Callable[JS::ConsoleArg],
    }}`,
		func(ctx eval.Context, args []eval.PValue) eval.PValue {
			return NewConsole(args[0].String())
		},
		func(ctx eval.Context, args []eval.PValue) eval.PValue {
			return NewConsole(args[0].(eval.KeyedValue).Get5(`name`, eval.EMPTY_STRING).String())
		})
}
