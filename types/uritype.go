package types

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/utils"
	"github.com/puppetlabs/go-parser/issue"
	"io"
	"net/url"
	"strings"
	"github.com/puppetlabs/go-evaluator/errors"
	"strconv"
	"fmt"
	"bytes"
)

var uriType_Default = &UriType{_UNDEF}

var URI_Type eval.ObjectType

var members = map[string]uriMemberFunc{
	`scheme`: func(uri *url.URL) eval.PValue {
		if uri.Scheme != `` {
			return WrapString(strings.ToLower(uri.Scheme))
		}
		return _UNDEF
	},
	`userinfo`: func(uri *url.URL) eval.PValue {
		if uri.User != nil {
			return WrapString(uri.User.String())
		}
		return _UNDEF
	},
	`host`: func(uri *url.URL) eval.PValue {
		if uri.Host != `` {
			h := uri.Host
			colon := strings.IndexByte(h, ':')
			if colon >= 0 {
				h = h[:colon]
			}
			return WrapString(strings.ToLower(h))
		}
		return _UNDEF
	},
	`port`: func(uri *url.URL) eval.PValue {
		port := uri.Port()
		if port != `` {
			if pn, err := strconv.Atoi(port); err == nil {
				return WrapInteger(int64(pn))
			}
		}
		return _UNDEF
	},
	`path`: func(uri *url.URL) eval.PValue {
		if uri.Path != `` {
			return WrapString(uri.Path)
		}
		return _UNDEF
	},
	`query`: func(uri *url.URL) eval.PValue {
		if uri.RawQuery != `` {
			return WrapString(uri.RawQuery)
		}
		return _UNDEF
	},
	`fragment`: func(uri *url.URL) eval.PValue {
		if uri.Fragment != `` {
			return WrapString(uri.Fragment)
		}
		return _UNDEF
	},
	`opaque`: func(uri *url.URL) eval.PValue {
		if uri.Opaque != `` {
			return WrapString(uri.Opaque)
		}
		return _UNDEF
	},
}

func init() {
	newAliasType(`Pcore::URIStringParam`, `Variant[String[1],Regexp,Type[Pattern],Type[Enum],Type[NotUndef],Type[Undef]]`)
	newAliasType(`Pcore::URIIntParam`, `Variant[Integer[0],Type[NotUndef],Type[Undef]]`)

	URI_Type = newObjectType(`Pcore::URIType`,
		`Pcore::AnyType{
  attributes => {
    parameters => {
      type => Variant[Undef, String[1], URI, Struct[
        Optional['scheme'] => URIStringParam,
        Optional['userinfo'] => URIStringParam,
        Optional['host'] => URIStringParam,
        Optional['port'] => URIIntParam,
        Optional['path'] => URIStringParam,
        Optional['query'] => URIStringParam,
        Optional['fragment'] => URIStringParam,
        Optional['opaque'] => URIStringParam,
      ]],
      value => undef
    }
  }
}`, func(ctx eval.EvalContext, args []eval.PValue) eval.PValue {
			return NewUriType2(args...)
		})

	newGoConstructor(`URI`,
		func(d eval.Dispatch) {
			d.Param(`String[1]`)
			d.OptionalParam(`Boolean`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				strict := false
				str := args[0].String()
				if len(args) > 1 {
					strict = args[1].(*BooleanValue).Bool()
				}
				u, err := ParseURI2(str, strict)
				if err != nil {
					panic(eval.Error(eval.EVAL_INVALID_URI, issue.H{`str`: str, `detail`: err.Error()}))
				}
				return WrapURI(u)
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Hash[String[1],ScalarData]`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				return WrapURI(URIFromHash(args[0].(*HashValue)))
			})
		})
}

type (
	UriType struct {
		parameters interface{} // string, URL, or hash
	}

	UriValue struct {
		UriType
	}

	uriMemberFunc func(*url.URL) eval.PValue

	uriMember struct {
	  memberFunc uriMemberFunc
	}
)

func (um *uriMember) Call(c eval.EvalContext, receiver eval.PValue, block eval.Lambda, args []eval.PValue) eval.PValue {
	return um.memberFunc(receiver.(*UriValue).URL())
}

func DefaultUriType() *UriType {
	return uriType_Default
}

func NewUriType(uri *url.URL) *UriType {
	return &UriType{uri}
}

func NewUriType3(parameters *HashValue) *UriType {
	if parameters.IsEmpty() {
		return uriType_Default
	}
	return &UriType{parameters}
}

func NewUriType2(args ...eval.PValue) *UriType {
	switch len(args) {
	case 0:
		return DefaultUriType()
	case 1:
		if str, ok := args[0].(*StringValue); ok {
			return NewUriType(ParseURI(str.String()))
		}
		if uri, ok := args[0].(*UriValue); ok {
			return NewUriType(uri.URL())
		}
		if hash, ok := args[0].(*HashValue); ok {
			return NewUriType3(hash)
		}
		panic(NewIllegalArgumentType2(`URI[]`, 0, `Variant[URI, Hash]`, args[0]))
	default:
		panic(errors.NewIllegalArgumentCount(`URI[]`, `0 or 1`, len(args)))
	}
}

// ParseURI parses a string into a uri.URL and panics with an issue code if the parse fails
func ParseURI(str string) *url.URL {
	uri, err := url.Parse(str)
	if err != nil {
		panic(eval.Error(eval.EVAL_INVALID_URI, issue.H{`str`: str, `detail`: err.Error()}))
	}
	return uri
}

func ParseURI2(str string, strict bool) (*url.URL, error) {
	if strict {
		return url.ParseRequestURI(str)
	}
	return url.Parse(str)
}

func URIFromHash(hash *HashValue) *url.URL {
	uri := &url.URL{}
	if scheme, ok := hash.Get4(`scheme`); ok {
		uri.Scheme = scheme.String()
	}
	if user, ok := hash.Get4(`userinfo`); ok {
		ustr := user.String()
		colon := strings.IndexByte(ustr, ':')
		if colon >= 0 {
			uri.User = url.UserPassword(ustr[:colon], ustr[colon+1:])
		} else {
			uri.User = url.User(ustr)
		}
	}
	if host, ok := hash.Get4(`host`); ok {
		uri.Host = host.String()
	}
	if port, ok := hash.Get4(`port`); ok {
		uri.Host = fmt.Sprintf(`%s:%d`, uri.Host, port.(*IntegerValue).Int())
	}
	if path, ok := hash.Get4(`path`); ok {
		uri.Path = path.String()
	}
	if query, ok := hash.Get4(`query`); ok {
		uri.RawQuery = query.String()
	}
	if fragment, ok := hash.Get4(`fragment`); ok {
		uri.Fragment = fragment.String()
	}
	if opaque, ok := hash.Get4(`opaque`); ok {
		uri.Opaque = opaque.String()
	}
	return uri
}

func (t *UriType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
}

func (t *UriType) Default() eval.PType {
	return uriType_Default
}

func (t *UriType) Equals(other interface{}, g eval.Guard) bool {
	if ot, ok := other.(*UriType); ok {
		switch t.parameters.(type) {
		case *UndefValue:
			return _UNDEF.Equals(ot.parameters, g)
		case *HashValue:
			return t.parameters.(*HashValue).Equals(ot.paramsAsHash(), g)
		default:
			if _UNDEF.Equals(ot.parameters, g) {
				return false
			}
			if _, ok := ot.parameters.(*url.URL); ok {
				return eval.Equals(t.parameters, ot.parameters)
			}
			return t.paramsAsHash().Equals(ot.paramsAsHash(), g)
		}
	}
	return false
}

func (t *UriType) Get(key string) (value eval.PValue, ok bool) {
	if key == `parameters` {
		switch t.parameters.(type) {
		case *UndefValue:
			return _UNDEF, true
		case *HashValue:
			return t.parameters.(*HashValue), true
		default:
			return urlToHash(t.parameters.(*url.URL)), true
		}
	}
	return nil, false
}

func (t *UriType) IsAssignable(other eval.PType, g eval.Guard) bool {
	if ot, ok := other.(*UriType); ok {
		switch t.parameters.(type) {
		case *UndefValue:
			return true
		default:
			oParams := ot.paramsAsHash()
			return t.paramsAsHash().AllPairs(func(k, b eval.PValue) bool {
				if a, ok := oParams.Get(k); ok {
					if at, ok := a.(eval.PType); ok {
						bt, ok := b.(eval.PType)
						return ok && isAssignable(bt, at)
					}
					return eval.PuppetMatch(a, b)
				}
				return false
			})
		}
	}
	return false
}

func (t *UriType) IsInstance(other eval.PValue, g eval.Guard) bool {
	if ov, ok := other.(*UriValue); ok {
		switch t.parameters.(type) {
		case *UndefValue:
			return true
		default:
			ovUri := ov.URL()
			return t.paramsAsHash().AllPairs(func(k, v eval.PValue) bool {
				return eval.PuppetMatch(v, getURLField(ovUri, k.String()))
			})
		}
	}
	return false
}


func (t *UriType) Member(name string) (eval.CallableMember, bool) {
	if member, ok := members[name]; ok {
		return &uriMember{member}, true
	}
	return nil, false
}

func (t *UriType) MetaType() eval.ObjectType {
	return URI_Type
}

func (t *UriType) Name() string {
	return `URI`
}

func (t *UriType) Parameters() []eval.PValue {
	switch t.parameters.(type) {
	case *UndefValue:
		return eval.EMPTY_VALUES
	case *HashValue:
		return []eval.PValue{t.parameters.(*HashValue)}
	default:
		return []eval.PValue{urlToHash(t.parameters.(*url.URL))}
	}
}

func (t *UriType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *UriType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *UriType) Type() eval.PType {
	return &TypeType{t}
}

func (t *UriType) paramsAsHash() *HashValue {
	switch t.parameters.(type) {
	case *UndefValue:
		return _EMPTY_MAP
	case *HashValue:
		return t.parameters.(*HashValue)
	default:
		return urlToHash(t.parameters.(*url.URL))
	}
}

func urlToHash(uri *url.URL) *HashValue {
	entries := make([]*HashEntry, 0, 8)
	if uri.Scheme != `` {
		entries = append(entries, WrapHashEntry2(`scheme`, WrapString(strings.ToLower(uri.Scheme))))
	}
	if uri.User != nil {
		entries = append(entries, WrapHashEntry2(`userinfo`, WrapString(uri.User.String())))
	}
	if uri.Host != `` {
		h := uri.Host
		colon := strings.IndexByte(h, ':')
		if colon >= 0 {
			entries = append(entries, WrapHashEntry2(`host`, WrapString(strings.ToLower(h[:colon]))))
			if p, err := strconv.Atoi(uri.Port()); err == nil {
				entries = append(entries, WrapHashEntry2(`port`, WrapInteger(int64(p))))
			}
		} else {
			entries = append(entries, WrapHashEntry2(`host`, WrapString(strings.ToLower(h))))
		}
	}
	if uri.Path != `` {
		entries = append(entries, WrapHashEntry2(`path`, WrapString(uri.Path)))
	}
	if uri.RawQuery != `` {
		entries = append(entries, WrapHashEntry2(`query`, WrapString(uri.RawQuery)))
	}
	if uri.Fragment != `` {
		entries = append(entries, WrapHashEntry2(`fragment`, WrapString(uri.Fragment)))
	}
	if uri.Opaque != `` {
		entries = append(entries, WrapHashEntry2(`opaque`, WrapString(uri.Opaque)))
	}
	return WrapHash(entries)
}

func getURLField(uri *url.URL, key string) eval.PValue {
	if member, ok := members[key]; ok {
		return member(uri)
	}
	return _UNDEF
}

func WrapURI(uri *url.URL) *UriValue {
	return &UriValue{UriType{uri}}
}

func WrapURI2(str string) *UriValue {
	return WrapURI(ParseURI(str))
}

func (u *UriValue) Equals(other interface{}, guard eval.Guard) bool {
	if ou, ok := other.(*UriValue); ok {
		return *u.URL() == *ou.URL()
	}
	return false
}

func (u *UriValue) Get(key string) (eval.PValue, bool) {
	if member, ok := members[key]; ok {
		return member(u.URL()), true
	}
	return _UNDEF, false
}

func (u *UriValue) SerializationString() string {
	return u.String()
}

func (u *UriValue) String() string {
	return eval.ToString(u)
}

func (u *UriValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	f := eval.GetFormat(s.FormatMap(), u.Type())
	val := u.URL().String()
	switch f.FormatChar() {
	case 's':
		f.ApplyStringFlags(b, val, f.IsAlt())
	case 'p':
		io.WriteString(b, `URI(`)
		utils.PuppetQuote(b, val)
		utils.WriteByte(b, ')')
	default:
		panic(s.UnsupportedFormat(u.Type(), `sp`, f))
	}
}

func (u *UriValue) ToKey(b *bytes.Buffer) {
	b.WriteByte(1)
	b.WriteByte(HK_URI)
	b.Write([]byte(u.URL().String()))
}

func (u *UriValue) Type() eval.PType {
	return &u.UriType
}

func (u *UriValue) URL() *url.URL {
	return u.parameters.(*url.URL)
}
