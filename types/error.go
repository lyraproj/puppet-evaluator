package types

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/eval"
	"io"
	"reflect"
)

var ErrorMetaType eval.ObjectType

func init() {
	ErrorMetaType = newGoObjectType(`Error`, reflect.TypeOf((*eval.ErrorObject)(nil)).Elem(), `{
		type_parameters => {
		  kind => Optional[Variant[String,Regexp,Type[Enum],Type[Pattern],Type[NotUndef],Type[Undef]]],
	  	issue_code => Optional[Variant[String,Regexp,Type[Enum],Type[Pattern],Type[NotUndef],Type[Undef]]]
		},
		attributes => {
		  message => String[1],
	  	kind => { type => Optional[String[1]], value => undef },
		  issue_code => { type => Optional[String[1]], value => undef },
		  partial_result => { type => Data, value => undef },
	  	details => { type => Optional[Hash[String[1],Data]], value => undef },
		}}`,
		func(ctx eval.Context, args []eval.Value) eval.Value {
			return newError2(ctx, args...)
		},
		func(ctx eval.Context, args []eval.Value) eval.Value {
			return newErrorFromHash(ctx, args[0].(*HashValue))
		})

	eval.NewError = newError
	eval.ErrorFromReported = errorFromReported
}

type errorObj struct {
	typ           eval.Type
	message       string
	kind          string
	issueCode     string
	partialResult eval.Value
	details       eval.OrderedMap
}

func newError2(c eval.Context, args ...eval.Value) eval.ErrorObject {
	argc := len(args)
	ev := &errorObj{partialResult: eval.Undef, details: eval.EmptyMap}
	ev.message = args[0].String()
	if argc > 1 {
		ev.kind = args[1].String()
		if argc > 2 {
			ev.issueCode = args[2].String()
			if argc > 3 {
				ev.partialResult = args[3]
				if argc > 4 {
					ev.details = args[4].(*HashValue)
				}
			}
		}
	}
	ev.initType(c)
	return ev
}

func newError(c eval.Context, message, kind, issueCode string, partialResult eval.Value, details eval.OrderedMap) eval.ErrorObject {
	if partialResult == nil {
		partialResult = eval.Undef
	}
	if details == nil {
		details = eval.EmptyMap
	}
	ev := &errorObj{message: message, kind: kind, issueCode: issueCode, partialResult: partialResult, details: details}
	ev.initType(c)
	return ev
}

func errorFromReported(c eval.Context, err issue.Reported) eval.ErrorObject {
	ev := &errorObj{partialResult: eval.Undef, details: eval.EmptyMap}
	ev.message = err.Error()
	ev.kind = `PUPPET_ERROR`
	ev.issueCode = string(err.Code())
	if loc := err.Location(); loc != nil {
		ev.details = SingletonHash2(`location`, stringValue(issue.LocationString(loc)))
	}
	ev.initType(c)
	return ev
}

func newErrorFromHash(c eval.Context, hash *HashValue) eval.ErrorObject {
	ev := &errorObj{}
	ev.message = hash.Get5(`message`, eval.EmptyString).String()
	ev.kind = hash.Get5(`kind`, eval.EmptyString).String()
	ev.issueCode = hash.Get5(`issue_code`, eval.EmptyString).String()
	ev.partialResult = hash.Get5(`partial_result`, eval.Undef)
	ev.details = hash.Get5(`details`, eval.EmptyMap).(eval.OrderedMap)
	ev.initType(c)
	return ev
}

func (e *errorObj) Details() eval.OrderedMap {
	return e.details
}

func (e *errorObj) IssueCode() string {
	return e.issueCode
}

func (e *errorObj) Kind() string {
	return e.kind
}

func (e *errorObj) Message() string {
	return e.message
}

func (e *errorObj) PartialResult() eval.Value {
	return e.partialResult
}

func (e *errorObj) String() string {
	return eval.ToString(e)
}

func (e *errorObj) Equals(other interface{}, guard eval.Guard) bool {
	if o, ok := other.(*errorObj); ok {
		return e.message == o.message && e.kind == o.kind && e.issueCode == o.issueCode &&
			eval.GuardedEquals(e.partialResult, o.partialResult, guard) &&
			eval.GuardedEquals(e.details, o.details, guard)
	}
	return false
}

func (e *errorObj) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	ObjectToString(e, s, b, g)
}

func (e *errorObj) PType() eval.Type {
	return e.typ
}

func (e *errorObj) Get(key string) (value eval.Value, ok bool) {
	switch key {
	case `message`:
		return stringValue(e.message), true
	case `kind`:
		return stringValue(e.kind), true
	case `issue_code`:
		return stringValue(e.issueCode), true
	case `partial_result`:
		return e.partialResult, true
	case `details`:
		return e.details, true
	default:
		return nil, false
	}
}

func (e *errorObj) InitHash() eval.OrderedMap {
	entries := []*HashEntry{WrapHashEntry2(`message`, stringValue(e.message))}
	if e.kind != `` {
		entries = append(entries, WrapHashEntry2(`kind`, stringValue(e.kind)))
	}
	if e.issueCode != `` {
		entries = append(entries, WrapHashEntry2(`issue_code`, stringValue(e.issueCode)))
	}
	if !e.partialResult.Equals(eval.Undef, nil) {
		entries = append(entries, WrapHashEntry2(`partial_result`, e.partialResult))
	}
	if !e.details.Equals(eval.EmptyMap, nil) {
		entries = append(entries, WrapHashEntry2(`details`, e.details))
	}
	return WrapHash(entries)
}

func (e *errorObj) initType(c eval.Context) {
	if e.kind == `` && e.issueCode == `` {
		e.typ = ErrorMetaType
	} else {
		params := make([]*HashEntry, 0)
		if e.kind != `` {
			params = append(params, WrapHashEntry2(`kind`, stringValue(e.kind)))
		}
		if e.issueCode != `` {
			params = append(params, WrapHashEntry2(`issue_code`, stringValue(e.issueCode)))
		}
		e.typ = NewObjectTypeExtension(c, ErrorMetaType, []eval.Value{WrapHash(params)})
	}
}
