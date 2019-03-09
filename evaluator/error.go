package evaluator

import (
	"io"
	"reflect"

	"github.com/lyraproj/puppet-evaluator/pdsl"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

var ErrorMetaType px.ObjectType

func init() {
	ErrorMetaType = px.NewGoObjectType(`Error`, reflect.TypeOf((*pdsl.ErrorObject)(nil)).Elem(), `{
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
		func(ctx px.Context, args []px.Value) px.Value {
			return newError2(ctx, args...)
		},
		func(ctx px.Context, args []px.Value) px.Value {
			return newErrorFromHash(ctx, args[0].(px.OrderedMap))
		})

	pdsl.NewError = newError
	pdsl.ErrorFromReported = errorFromReported
}

type errorObj struct {
	typ           px.Type
	message       string
	kind          string
	issueCode     string
	partialResult px.Value
	details       px.OrderedMap
}

func newError2(c px.Context, args ...px.Value) pdsl.ErrorObject {
	argc := len(args)
	ev := &errorObj{partialResult: px.Undef, details: px.EmptyMap}
	ev.message = args[0].String()
	if argc > 1 {
		ev.kind = args[1].String()
		if argc > 2 {
			ev.issueCode = args[2].String()
			if argc > 3 {
				ev.partialResult = args[3]
				if argc > 4 {
					ev.details = args[4].(px.OrderedMap)
				}
			}
		}
	}
	ev.initType(c)
	return ev
}

func newError(c px.Context, message, kind, issueCode string, partialResult px.Value, details px.OrderedMap) pdsl.ErrorObject {
	if partialResult == nil {
		partialResult = px.Undef
	}
	if details == nil {
		details = px.EmptyMap
	}
	ev := &errorObj{message: message, kind: kind, issueCode: issueCode, partialResult: partialResult, details: details}
	ev.initType(c)
	return ev
}

func errorFromReported(c px.Context, err issue.Reported) pdsl.ErrorObject {
	ev := &errorObj{partialResult: px.Undef, details: px.EmptyMap}
	ev.message = err.Error()
	ev.kind = `PUPPET_ERROR`
	ev.issueCode = string(err.Code())
	if loc := err.Location(); loc != nil {
		ev.details = px.SingletonMap(`location`, types.WrapString(issue.LocationString(loc)))
	}
	ev.initType(c)
	return ev
}

func newErrorFromHash(c px.Context, hash px.OrderedMap) pdsl.ErrorObject {
	ev := &errorObj{}
	ev.message = hash.Get5(`message`, px.EmptyString).String()
	ev.kind = hash.Get5(`kind`, px.EmptyString).String()
	ev.issueCode = hash.Get5(`issue_code`, px.EmptyString).String()
	ev.partialResult = hash.Get5(`partial_result`, px.Undef)
	ev.details = hash.Get5(`details`, px.EmptyMap).(px.OrderedMap)
	ev.initType(c)
	return ev
}

func (e *errorObj) Details() px.OrderedMap {
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

func (e *errorObj) PartialResult() px.Value {
	return e.partialResult
}

func (e *errorObj) String() string {
	return px.ToString(e)
}

func (e *errorObj) Equals(other interface{}, guard px.Guard) bool {
	if o, ok := other.(*errorObj); ok {
		return e.message == o.message && e.kind == o.kind && e.issueCode == o.issueCode &&
			px.Equals(e.partialResult, o.partialResult, guard) &&
			px.Equals(e.details, o.details, guard)
	}
	return false
}

func (e *errorObj) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	types.ObjectToString(e, s, b, g)
}

func (e *errorObj) PType() px.Type {
	return e.typ
}

func (e *errorObj) Get(key string) (value px.Value, ok bool) {
	switch key {
	case `message`:
		return types.WrapString(e.message), true
	case `kind`:
		return types.WrapString(e.kind), true
	case `issue_code`:
		return types.WrapString(e.issueCode), true
	case `partial_result`:
		return e.partialResult, true
	case `details`:
		return e.details, true
	default:
		return nil, false
	}
}

func (e *errorObj) InitHash() px.OrderedMap {
	entries := []*types.HashEntry{types.WrapHashEntry2(`message`, types.WrapString(e.message))}
	if e.kind != `` {
		entries = append(entries, types.WrapHashEntry2(`kind`, types.WrapString(e.kind)))
	}
	if e.issueCode != `` {
		entries = append(entries, types.WrapHashEntry2(`issue_code`, types.WrapString(e.issueCode)))
	}
	if !e.partialResult.Equals(px.Undef, nil) {
		entries = append(entries, types.WrapHashEntry2(`partial_result`, e.partialResult))
	}
	if !e.details.Equals(px.EmptyMap, nil) {
		entries = append(entries, types.WrapHashEntry2(`details`, e.details))
	}
	return types.WrapHash(entries)
}

func (e *errorObj) initType(c px.Context) {
	if e.kind == `` && e.issueCode == `` {
		e.typ = ErrorMetaType
	} else {
		params := make([]*types.HashEntry, 0)
		if e.kind != `` {
			params = append(params, types.WrapHashEntry2(`kind`, types.WrapString(e.kind)))
		}
		if e.issueCode != `` {
			params = append(params, types.WrapHashEntry2(`issue_code`, types.WrapString(e.issueCode)))
		}
		e.typ = types.NewObjectTypeExtension(c, ErrorMetaType, []px.Value{types.WrapHash(params)})
	}
}
