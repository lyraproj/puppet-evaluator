package resource

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"io"
)

type Result interface {
	eval.PuppetObject

	// Id() returns the identifier for this result
	Id() eval.PValue

	// Message returns an optional message associated with the result
	Message() string

	// Ok() returns false if the Value() method returns a *types.error and
	// true for any other value
	Ok() bool

	// Value returns the result value. It will be a *types.error when
	// the Ok() method returns false
	Value() eval.PValue
}

type ResultSet interface {
	eval.PuppetObject

	// At returns the result for a given key together with a boolean
	// indicating if the value was found or not
	At(eval.Context, eval.PValue) (eval.PValue, bool)

	// ErrorSet returns the subset of the receiver that contains only
	// error results
	ErrorSet() ResultSet

	// First returns the first entry in the result set. It will return
	// eval.UNDEF for an empty set
	First() eval.PValue

	// Ok returns true if the receiver contains no errors, false otherwise
	Ok() bool

	// OkSet returns the subset of the receiver that contains only
	// ok results
	OkSet() ResultSet

	// Results returns all the results contained in this set
	Results() eval.IndexedValue
}

type result struct {
	id eval.PValue
	message string
	value eval.PValue
}

type resultSet struct {
	results eval.IndexedValue
}

func NewResult(id, value eval.PValue, message string) Result {
	return &result{id, message, value}
}

func NewErrorResult(id eval.PValue, error  eval.ErrorObject) Result {
	return &result{id, ``, error}
}

func NewResult2(args...eval.PValue) Result {
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

func NewResultFromHash(hash *types.HashValue) Result {
	return NewResult(
		hash.Get5(`id`, eval.UNDEF),
		hash.Get6(`value`, func() eval.PValue { return hash.Get5(`error`, eval.UNDEF) }),
		hash.Get5(`message`, eval.EMPTY_STRING).String())
}

func (r *result) Equals(other interface{}, guard eval.Guard) bool {
	if o, ok := other.(*result); ok {
		return r.id.Equals(o.id, guard) && r.message == o.message && r.value.Equals(o.value, guard)
	}
	return false
}

func (r *result) Get(c eval.Context, key string) (value eval.PValue, ok bool) {
	switch key {
	case `error`:
		if err, ok := r.Error(); ok {
			return err, true
		}
		return eval.UNDEF, true
	case `id`:
		return r.Id(), true
	case `message`:
		msg := r.Message()
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

func (r *result) Id() eval.PValue {
	return r.id
}

func (r *result) InitHash() eval.KeyedValue {
	v := map[string]eval.PValue{`id`: r.id}
	if err, ok := r.value.(eval.ErrorObject); ok {
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

func (r *result) Error() (eval.ErrorObject, bool) {
	if e, ok := r.value.(eval.ErrorObject); ok {
		return e, true
	}
	return nil, false
}

func (r *result) Message() string {
	return r.message
}

func (r *result) Ok() bool {
	_, isError := r.value.(eval.ErrorObject)
	return !isError
}

func (r *result) String() string {
	return eval.ToString(r)
}

func (r *result) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	types.ObjectToString(r, s, b, g)
}

func (r *result) Type() eval.PType {
	return resultType
}

func (r *result) Value() eval.PValue {
	return r.value
}

func NewResultSet(args...eval.PValue) ResultSet {
	if len(args) == 0 {
		return &resultSet{eval.EMPTY_ARRAY}
	}
	return &resultSet{args[0].(*types.ArrayValue)}
}

func NewResultSetFromHash(hash *types.HashValue) ResultSet {
	return &resultSet{hash.Get5(`results`, eval.EMPTY_ARRAY).(*types.ArrayValue)}
}

func (rs *resultSet) Call(c eval.Context, method string, args []eval.PValue, block eval.Lambda) (result eval.PValue, ok bool) {
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
		if found, ok := rs.At(c, args[0]); ok {
			return found, true
		}
		return eval.UNDEF, true
	case `first`:
		return rs.First(), true
	case `ids`:
		return rs.results.Map(func(r eval.PValue) eval.PValue { return r.(Result).Id() }), true
	case `ok_set`:
		return rs.OkSet(), true
	}
	return nil, false
}

func (rs *resultSet) ErrorSet() ResultSet {
	return &resultSet{rs.results.Reject(func(r eval.PValue) bool { return r.(Result).Ok() }).(*types.ArrayValue)}
}

func (rs *resultSet) At(c eval.Context, id eval.PValue) (eval.PValue, bool) {
	if rt, ok := id.(eval.ParameterizedType); ok {
		id = types.WrapString(Reference(c, rt))
	}
	return rs.results.Find(func(r eval.PValue) bool { return eval.Equals(id, r.(Result).Id()) })
}

func (rs *resultSet) First() eval.PValue {
	if rs.results.Len() > 0 {
		return rs.results.At(0)
	}
	return eval.UNDEF
}

func (rs *resultSet) Ok() bool {
	return rs.results.All(func(r eval.PValue) bool { return r.(Result).Ok() })
}

func (rs *resultSet) OkSet() ResultSet {
	return &resultSet{rs.results.Select(func(r eval.PValue) bool { return r.(Result).Ok() }).(*types.ArrayValue)}
}

func (rs *resultSet) Results() eval.IndexedValue {
	return rs.results
}

func (rs *resultSet) Get(c eval.Context, key string) (value eval.PValue, ok bool) {
	if key == `results` {
		return rs.results, true
	}
	return nil, false
}

func (rs *resultSet) InitHash() eval.KeyedValue {
	return types.SingletonHash2(`results`, rs.results)
}

func (rs *resultSet) String() string {
	return eval.ToString(rs)
}

func (rs *resultSet) Equals(other interface{}, guard eval.Guard) bool {
	if o, ok := other.(*resultSet); ok {
		return rs.results.Equals(o.results, guard)
	}
	return false
}

func (rs *resultSet) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	types.ObjectToString(rs, s, b, g)
}

func (rs *resultSet) Type() eval.PType {
	return resultSetType
}
