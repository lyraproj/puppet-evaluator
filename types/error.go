package types

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"io"
)

var errorType eval.ObjectType

func init() {
	errorType = newObjectType(`Error`, `{
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
	func(ctx eval.Context, args []eval.PValue) eval.PValue {
		return NewError(ctx, args...)
	},
	func(ctx eval.Context, args []eval.PValue) eval.PValue {
		return NewErrorFromHash(ctx, args[0].(*HashValue))
	})
}

type Error struct {
	typ eval.PType
	message string
	kind string
	issue_code string
	partial_result eval.PValue
	details eval.PValue
}

func (e *Error) Message() string {
	return e.message
}

func (e *Error) Kind() string {
	return e.kind
}

func (e *Error) IssueCode() string {
	return e.issue_code
}

func (e *Error) PartialResult() eval.PValue {
	return e.partial_result
}

func (e *Error) Details() eval.PValue {
	return e.details
}

func (e *Error) String() string {
	return eval.ToString(e)
}

func (e *Error) Equals(other interface{}, guard eval.Guard) bool {
	if o, ok := other.(*Error); ok {
		return e.message == o.message && e.kind == o.kind && e.issue_code == o.issue_code &&
			eval.GuardedEquals(e.partial_result, o.partial_result, guard) &&
			eval.GuardedEquals(e.details, o.details, guard)
	}
	return false
}

func (e *Error) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	ObjectToString(e, s, b, g)
}

func (e *Error) Type() eval.PType {
	return e.typ
}

func (e *Error) Get(c eval.Context, key string) (value eval.PValue, ok bool) {
	switch key {
	case `message`:
		return WrapString(e.message), true
	case `kind`:
		return WrapString(e.kind), true
	case `issue_code`:
		return WrapString(e.issue_code), true
	case `partial_result`:
		return e.partial_result, true
	case `details`:
		return e.details, true
	default:
		return nil, false
	}
}

func (e *Error) InitHash() eval.KeyedValue {
	entries := []*HashEntry{WrapHashEntry2(`message`, WrapString(e.message))}
	if e.kind != `` {
		entries = append(entries, WrapHashEntry2(`kind`, WrapString(e.kind)))
	}
	if e.issue_code != `` {
		entries = append(entries, WrapHashEntry2(`issue_code`, WrapString(e.issue_code)))
	}
	if !e.partial_result.Equals(eval.UNDEF, nil) {
		entries = append(entries, WrapHashEntry2(`partial_result`, e.partial_result))
	}
	if !e.details.Equals(eval.UNDEF, nil) {
		entries = append(entries, WrapHashEntry2(`details`, e.details))
	}
	return WrapHash(entries)
}

func NewError(c eval.Context, args...eval.PValue) eval.PuppetObject {
	nargs := len(args)
	ev := &Error{partial_result: eval.UNDEF, details: eval.UNDEF}
	ev.message = args[0].String()
	if nargs > 1 {
		ev.kind = args[1].String()
		if nargs > 2 {
			ev.issue_code = args[2].String()
			if nargs > 3 {
				ev.partial_result = args[3]
				if nargs > 4 {
					ev.details = args[4].(*HashValue)
				}
			}
		}
	}
	ev.initType(c)
	return ev
}

func NewErrorFromHash(c eval.Context, hash *HashValue) eval.PuppetObject {
	ev := &Error{}
	ev.message = hash.Get5(`message`, eval.EMPTY_STRING).String()
	ev.kind = hash.Get5(`kind`, eval.EMPTY_STRING).String()
	ev.issue_code = hash.Get5(`issue_code`, eval.EMPTY_STRING).String()
	ev.partial_result = hash.Get5(`partial_result`, eval.UNDEF)
	ev.details = hash.Get5(`details`, eval.UNDEF)
	ev.initType(c)
	return ev
}

func (e *Error) initType(c eval.Context) {
	if e.kind == `` && e.issue_code == `` {
		e.typ = errorType
	} else {
		params := make([]*HashEntry, 0)
		if e.kind != `` {
			params = append(params, WrapHashEntry2(`kind`, WrapString(e.kind)))
		}
		if e.issue_code != `` {
			params = append(params, WrapHashEntry2(`issue_code`, WrapString(e.issue_code)))
		}
		e.typ = NewObjectTypeExtension(c, errorType, []eval.PValue{WrapHash(params)})
	}
}
