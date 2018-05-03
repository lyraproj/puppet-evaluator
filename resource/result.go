package resource

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"io"
)

var resultType, resultSetType eval.PType

func init() {
	resultType = eval.NewObjectType(`Result`, `{
    attributes => {
      'id' => ScalarData,
      'value' => { type => RichData, value => undef },
      'message' => { type => Optional[String], value => undef },
      'error' => { type => Boolean, kind => derived },
      'ok' => { type => Boolean, kind => derived }
    }
  }`, func(ctx eval.Context, args []eval.PValue) eval.PValue {
		return NewResult2(ctx, args...)
	}, func(ctx eval.Context, args []eval.PValue) eval.PValue {
		return NewResultFromHash(ctx, args[0].(*types.HashValue))
	})

	resultSetType = eval.NewObjectType(`ResultSet`, `{
    attributes => {
      'results' => Array[Result],
    },
    functions => {
      count => Callable[[], Integer],
      empty => Callable[[], Boolean],
      error_set => Callable[[], ResultSet],
      first => Callable[[], Optional[Result]],
      ids => Callable[[], Array[ScalarData]],
      ok => Callable[[], Boolean],
      ok_set => Callable[[], ResultSet],
      '[]' => Callable[[ScalarData], Optional[Result]],
    }
  }`, func(ctx eval.Context, args []eval.PValue) eval.PValue {
		return NewResultSet(ctx, args...)
	}, func(ctx eval.Context, args []eval.PValue) eval.PValue {
		return NewResultSetFromHash(ctx, args[0].(*types.HashValue))
	})
}

type Result struct {
	id eval.PValue
	message string
	value eval.PValue
}

type ResultSet struct {
	results *types.ArrayValue
}

func NewResult(id, value eval.PValue, message string) *Result {
	return &Result{id, message, value}
}

func NewErrorResult(id eval.PValue, error *types.Error) *Result {
	return &Result{id, ``, error}
}

func NewResult2(c eval.Context, args...eval.PValue) eval.PuppetObject {
	value := eval.UNDEF
	var message string
	if len(args) > 1 {
		value = args[1]
		if len(args) > 2 {
			message = args[2].String()
		}
	}
	return NewResult(args[0], value, message)
}

func NewResultFromHash(c eval.Context, hash *types.HashValue) eval.PuppetObject {
	return NewResult(
		hash.Get5(`id`, eval.UNDEF),
		hash.Get6(`value`, func() eval.PValue { return hash.Get5(`error`, eval.UNDEF) }),
		hash.Get5(`message`, eval.EMPTY_STRING).String())
}

func (r *Result) Equals(other interface{}, guard eval.Guard) bool {
	if o, ok := other.(*Result); ok {
		return r.id.Equals(o.id, guard) && r.message == o.message && r.value.Equals(o.value, guard)
	}
	return false
}

func (r *Result) Get(c eval.Context, key string) (value eval.PValue, ok bool) {
	switch key {
	case `error`:
		if err, ok := r.Error(); ok {
			return err, true
		}
		return eval.UNDEF, true
	case `id`:
		return r.Id(), true
	case `message`:
		msg := r.Message(c)
		if msg == `` {
			return eval.UNDEF, true
		}
		return types.WrapString(msg), true
	case `ok`:
		return types.WrapBoolean(r.Ok()), true
	case `value`:
		return r.Value(), true
	default:
		return nil, false
	}
}

func (r *Result) Id() eval.PValue {
	return r.id
}

func (r *Result) InitHash() eval.KeyedValue {
	v := map[string]eval.PValue{`id`: r.id}
	if err, ok := r.value.(*types.Error); ok {
		v[`error`] = err
	} else {
		if r.message != `` {
			v[`message`] = types.WrapString(r.message)
		}
		if !r.value.Equals(eval.UNDEF, nil) {
			v[`value`] = r.value
		}
	}
	return types.WrapHash3(v)
}

func (r *Result) Error() (*types.Error, bool) {
	if e, ok := r.value.(*types.Error); ok {
		return e, true
	}
	return nil, false
}

func (r *Result) Message(c eval.Context) string {
	return r.message
}

func (r *Result) Ok() bool {
	_, isError := r.value.(*types.Error)
	return !isError
}

func (r *Result) String() string {
	return eval.ToString(r)
}

func (r *Result) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	types.ObjectToString(r, s, b, g)
}

func (r *Result) Type() eval.PType {
	return resultType
}

func (r *Result) Value() eval.PValue {
	return r.value
}

func NewResultSet(c eval.Context, args...eval.PValue) eval.PuppetObject {
	return &ResultSet{args[0].(*types.ArrayValue)}
}

func NewResultSetFromHash(c eval.Context, hash *types.HashValue) eval.PuppetObject {
	return &ResultSet{hash.Get5(`results`, eval.EMPTY_ARRAY).(*types.ArrayValue)}
}

func (rs *ResultSet) Call(method string, args []eval.PValue, block eval.Lambda) (result eval.PValue, ok bool) {
	switch method {
	case `ok`:
		return types.WrapBoolean(rs.Ok()), true
	case `count`:
		return types.WrapInteger(int64(rs.results.Len())), true
	case `empty`:
		return types.WrapBoolean(rs.results.IsEmpty()), true
	case `error_set`:
		return rs.ErrorSet(), true
	case `[]`:
		if found, ok := rs.At(args[0]); ok {
			return found, true
		}
		return eval.UNDEF, true
	case `first`:
		if found, ok := rs.First(); ok {
			return found, true
		}
		return eval.UNDEF, true
	case `ids`:
		return rs.results.Map(func(r eval.PValue) eval.PValue { return r.(*Result).Id() }), true
	case `ok_set`:
		return rs.OkSet(), true
	}
	return nil, false
}

func (rs *ResultSet) ErrorSet() *ResultSet {
	return &ResultSet{rs.results.Reject(func(r eval.PValue) bool { return r.(*Result).Ok() }).(*types.ArrayValue)}
}

func (rs *ResultSet) At(id eval.PValue) (eval.PValue, bool) {
	return rs.results.Find(func(r eval.PValue) bool { return eval.Equals(id, r.(*Result).Id()) })
}

func (rs *ResultSet) First() (eval.PValue, bool) {
	if rs.results.Len() > 0 {
		return rs.results.At(0), true
	}
	return eval.UNDEF, false
}

func (rs *ResultSet) Ok() bool {
	return rs.results.All(func(r eval.PValue) bool { return r.(*Result).Ok() })
}

func (rs *ResultSet) OkSet() *ResultSet {
	return &ResultSet{rs.results.Select(func(r eval.PValue) bool { return r.(*Result).Ok() }).(*types.ArrayValue)}
}

func (rs *ResultSet) Get(c eval.Context, key string) (value eval.PValue, ok bool) {
	if key == `results` {
		return rs.results, true
	}
	return nil, false
}

func (rs *ResultSet) InitHash() eval.KeyedValue {
	return types.WrapHash3(map[string]eval.PValue{`results`: rs.results})
}

func (rs *ResultSet) String() string {
	return eval.ToString(rs)
}

func (rs *ResultSet) Equals(other interface{}, guard eval.Guard) bool {
	if o, ok := other.(*ResultSet); ok {
		return rs.results.Equals(o.results, guard)
	}
	return false
}

func (rs *ResultSet) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	types.ObjectToString(rs, s, b, g)
}

func (rs *ResultSet) Type() eval.PType {
	return resultSetType
}
