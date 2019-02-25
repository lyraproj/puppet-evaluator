package types

import (
	"bytes"
	"fmt"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/errors"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/utils"
	"io"
	"net/url"
	"strconv"
	"strings"
)

var uriType_Default = &UriType{_UNDEF}

var URIMetaType eval.ObjectType

var members = map[string]uriMemberFunc{
	`scheme`: func(uri *url.URL) eval.Value {
		if uri.Scheme != `` {
			return stringValue(strings.ToLower(uri.Scheme))
		}
		return _UNDEF
	},
	`userinfo`: func(uri *url.URL) eval.Value {
		if uri.User != nil {
			return stringValue(uri.User.String())
		}
		return _UNDEF
	},
	`host`: func(uri *url.URL) eval.Value {
		if uri.Host != `` {
			h := uri.Host
			colon := strings.IndexByte(h, ':')
			if colon >= 0 {
				h = h[:colon]
			}
			return stringValue(strings.ToLower(h))
		}
		return _UNDEF
	},
	`port`: func(uri *url.URL) eval.Value {
		port := uri.Port()
		if port != `` {
			if pn, err := strconv.Atoi(port); err == nil {
				return integerValue(int64(pn))
			}
		}
		return _UNDEF
	},
	`path`: func(uri *url.URL) eval.Value {
		if uri.Path != `` {
			return stringValue(uri.Path)
		}
		return _UNDEF
	},
	`query`: func(uri *url.URL) eval.Value {
		if uri.RawQuery != `` {
			return stringValue(uri.RawQuery)
		}
		return _UNDEF
	},
	`fragment`: func(uri *url.URL) eval.Value {
		if uri.Fragment != `` {
			return stringValue(uri.Fragment)
		}
		return _UNDEF
	},
	`opaque`: func(uri *url.URL) eval.Value {
		if uri.Opaque != `` {
			return stringValue(uri.Opaque)
		}
		return _UNDEF
	},
}

func init() {
	newTypeAlias(`Pcore::URIStringParam`, `Variant[String[1],Regexp,Type[Pattern],Type[Enum],Type[NotUndef],Type[Undef]]`)
	newTypeAlias(`Pcore::URIIntParam`, `Variant[Integer[0],Type[NotUndef],Type[Undef]]`)

	URIMetaType = newObjectType(`Pcore::URIType`,
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
}`, func(ctx eval.Context, args []eval.Value) eval.Value {
			return NewUriType2(args...)
		})

	newGoConstructor(`URI`,
		func(d eval.Dispatch) {
			d.Param(`String[1]`)
			d.OptionalParam(`Boolean`)
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
				strict := false
				str := args[0].String()
				if len(args) > 1 {
					strict = args[1].(booleanValue).Bool()
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
			d.Function(func(c eval.Context, args []eval.Value) eval.Value {
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

	uriMemberFunc func(*url.URL) eval.Value

	uriMember struct {
		memberFunc uriMemberFunc
	}
)

func (um *uriMember) Call(c eval.Context, receiver eval.Value, block eval.Lambda, args []eval.Value) eval.Value {
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

func NewUriType2(args ...eval.Value) *UriType {
	switch len(args) {
	case 0:
		return DefaultUriType()
	case 1:
		if str, ok := args[0].(stringValue); ok {
			return NewUriType(ParseURI(string(str)))
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
		uri.Host = fmt.Sprintf(`%s:%d`, uri.Host, port.(integerValue).Int())
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

func (t *UriType) Default() eval.Type {
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

func (t *UriType) Get(key string) (value eval.Value, ok bool) {
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

func (t *UriType) IsAssignable(other eval.Type, g eval.Guard) bool {
	if ot, ok := other.(*UriType); ok {
		switch t.parameters.(type) {
		case *UndefValue:
			return true
		default:
			oParams := ot.paramsAsHash()
			return t.paramsAsHash().AllPairs(func(k, b eval.Value) bool {
				if a, ok := oParams.Get(k); ok {
					if at, ok := a.(eval.Type); ok {
						bt, ok := b.(eval.Type)
						return ok && isAssignable(bt, at)
					}
					return eval.PuppetMatch(nil, a, b)
				}
				return false
			})
		}
	}
	return false
}

func (t *UriType) IsInstance(other eval.Value, g eval.Guard) bool {
	if ov, ok := other.(*UriValue); ok {
		switch t.parameters.(type) {
		case *UndefValue:
			return true
		default:
			ovUri := ov.URL()
			return t.paramsAsHash().AllPairs(func(k, v eval.Value) bool {
				return eval.PuppetMatch(nil, v, getURLField(ovUri, k.String()))
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
	return URIMetaType
}

func (t *UriType) Name() string {
	return `URI`
}

func (t *UriType) Parameters() []eval.Value {
	switch t.parameters.(type) {
	case *UndefValue:
		return eval.EMPTY_VALUES
	case *HashValue:
		return []eval.Value{t.parameters.(*HashValue)}
	default:
		return []eval.Value{urlToHash(t.parameters.(*url.URL))}
	}
}

func (t *UriType) CanSerializeAsString() bool {
	return true
}

func (t *UriType) SerializationString() string {
	return t.String()
}

func (t *UriType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *UriType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *UriType) PType() eval.Type {
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
		entries = append(entries, WrapHashEntry2(`scheme`, stringValue(strings.ToLower(uri.Scheme))))
	}
	if uri.User != nil {
		entries = append(entries, WrapHashEntry2(`userinfo`, stringValue(uri.User.String())))
	}
	if uri.Host != `` {
		h := uri.Host
		colon := strings.IndexByte(h, ':')
		if colon >= 0 {
			entries = append(entries, WrapHashEntry2(`host`, stringValue(strings.ToLower(h[:colon]))))
			if p, err := strconv.Atoi(uri.Port()); err == nil {
				entries = append(entries, WrapHashEntry2(`port`, integerValue(int64(p))))
			}
		} else {
			entries = append(entries, WrapHashEntry2(`host`, stringValue(strings.ToLower(h))))
		}
	}
	if uri.Path != `` {
		entries = append(entries, WrapHashEntry2(`path`, stringValue(uri.Path)))
	}
	if uri.RawQuery != `` {
		entries = append(entries, WrapHashEntry2(`query`, stringValue(uri.RawQuery)))
	}
	if uri.Fragment != `` {
		entries = append(entries, WrapHashEntry2(`fragment`, stringValue(uri.Fragment)))
	}
	if uri.Opaque != `` {
		entries = append(entries, WrapHashEntry2(`opaque`, stringValue(uri.Opaque)))
	}
	return WrapHash(entries)
}

func getURLField(uri *url.URL, key string) eval.Value {
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

func (u *UriValue) Get(key string) (eval.Value, bool) {
	if member, ok := members[key]; ok {
		return member(u.URL()), true
	}
	return _UNDEF, false
}

func (u *UriValue) CanSerializeAsString() bool {
	return true
}

func (u *UriValue) SerializationString() string {
	return u.String()
}

func (u *UriValue) String() string {
	return eval.ToString(u)
}

func (u *UriValue) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	f := eval.GetFormat(s.FormatMap(), u.PType())
	val := u.URL().String()
	switch f.FormatChar() {
	case 's':
		f.ApplyStringFlags(b, val, f.IsAlt())
	case 'p':
		io.WriteString(b, `URI(`)
		utils.PuppetQuote(b, val)
		utils.WriteByte(b, ')')
	default:
		panic(s.UnsupportedFormat(u.PType(), `sp`, f))
	}
}

func (u *UriValue) ToKey(b *bytes.Buffer) {
	b.WriteByte(1)
	b.WriteByte(HK_URI)
	b.Write([]byte(u.URL().String()))
}

func (u *UriValue) PType() eval.Type {
	return &u.UriType
}

func (u *UriValue) URL() *url.URL {
	return u.parameters.(*url.URL)
}
