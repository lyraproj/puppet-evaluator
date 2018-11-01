package resource

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"io"
)

type Result interface {
	eval.PuppetObject

	// Id() returns the identifier for this result
	Id() eval.Value

	// Message returns an optional message associated with the result
	Message() string

	// Ok() returns false if the Value() method returns a *types.error and
	// true for any other value
	Ok() bool

	// Value returns the result value. It will be a *types.error when
	// the Ok() method returns false
	Value() eval.Value
}

type ResultSet interface {
	eval.PuppetObject

	// At returns the result for a given key together with a boolean
	// indicating if the value was found or not
	At(eval.Context, eval.Value) (eval.Value, bool)

	// ErrorSet returns the subset of the receiver that contains only
	// error results
	ErrorSet() ResultSet

	// First returns the first entry in the result set. It will return
	// eval.UNDEF for an empty set
	First() eval.Value

	// Ok returns true if the receiver contains no errors, false otherwise
	Ok() bool

	// OkSet returns the subset of the receiver that contains only
	// ok results
	OkSet() ResultSet

	// Results returns all the results contained in this set
	Results() eval.List
}

type result struct {
	id      eval.Value
	message string
	value   eval.Value
}

type resultSet struct {
	results eval.List
}

func NewResult(id, value eval.Value, message string) Result {
	return &result{id, message, value}
}

func NewErrorResult(id eval.Value, error eval.ErrorObject) Result {
	return &result{id, ``, error}
}

func NewResult2(args ...eval.Value) Result {
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
		hash.Get6(`value`, func() eval.Value { return hash.Get5(`error`, eval.UNDEF) }),
		hash.Get5(`message`, eval.EMPTY_STRING).String())
}

func (r *result) Equals(other interface{}, guard eval.Guard) bool {
	if o, ok := other.(*result); ok {
		return r.id.Equals(o.id, guard) && r.message == o.message && r.value.Equals(o.value, guard)
	}
	return false
}

func (r *result) Get(key string) (value eval.Value, ok bool) {
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

func (r *result) Id() eval.Value {
	return r.id
}

func (r *result) InitHash() eval.OrderedMap {
	v := map[string]eval.Value{`id`: r.id}
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

func (r *result) PType() eval.Type {
	return resultType
}

func (r *result) Value() eval.Value {
	return r.value
}

func NewResultSet(args ...eval.Value) ResultSet {
	if len(args) == 0 {
		return &resultSet{eval.EMPTY_ARRAY}
	}
	return &resultSet{args[0].(*types.ArrayValue)}
}

func NewResultSetFromHash(hash *types.HashValue) ResultSet {
	return &resultSet{hash.Get5(`results`, eval.EMPTY_ARRAY).(*types.ArrayValue)}
}

func (rs *resultSet) Call(c eval.Context, method string, args []eval.Value, block eval.Lambda) (result eval.Value, ok bool) {
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
		return rs.results.Map(func(r eval.Value) eval.Value { return r.(Result).Id() }), true
	case `ok_set`:
		return rs.OkSet(), true
	}
	return nil, false
}

func (rs *resultSet) ErrorSet() ResultSet {
	return &resultSet{rs.results.Reject(func(r eval.Value) bool { return r.(Result).Ok() }).(*types.ArrayValue)}
}

func (rs *resultSet) At(c eval.Context, id eval.Value) (eval.Value, bool) {
	if rt, ok := id.(eval.ParameterizedType); ok {
		id = types.WrapString(Reference(rt))
	}
	return rs.results.Find(func(r eval.Value) bool { return eval.Equals(id, r.(Result).Id()) })
}

func (rs *resultSet) First() eval.Value {
	if rs.results.Len() > 0 {
		return rs.results.At(0)
	}
	return eval.UNDEF
}

func (rs *resultSet) Ok() bool {
	return rs.results.All(func(r eval.Value) bool { return r.(Result).Ok() })
}

func (rs *resultSet) OkSet() ResultSet {
	return &resultSet{rs.results.Select(func(r eval.Value) bool { return r.(Result).Ok() }).(*types.ArrayValue)}
}

func (rs *resultSet) Results() eval.List {
	return rs.results
}

func (rs *resultSet) Get(key string) (value eval.Value, ok bool) {
	if key == `results` {
		return rs.results, true
	}
	return nil, false
}

func (rs *resultSet) InitHash() eval.OrderedMap {
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

func (rs *resultSet) PType() eval.Type {
	return resultSetType
}
