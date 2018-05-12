package types

import (
	"io"

	"fmt"
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-issues/issue"
	"github.com/puppetlabs/go-parser/parser"
	"github.com/puppetlabs/go-semver/semver"
	"math"
	"strings"
	"sync/atomic"
)

const (
	KEY_NAME_AUTHORITY = `name_authority`
	KEY_REFERENCES     = `references`
	KEY_TYPES          = `types`
	KEY_VERSION        = `version`
	KEY_VERSION_RANGE  = `version_range`
)

var TypeSet_Type eval.ObjectType

var TYPE_STRING_OR_VERSION = NewVariantType([]eval.PType{
	stringType_NOT_EMPTY,
	DefaultSemVerType()})

var TYPE_SIMPLE_TYPE_NAME = NewPatternType([]*RegexpType{NewRegexpType(`\A[A-Z]\w*\z`)})
var TYPE_QUALIFIED_REFERENCE = NewPatternType([]*RegexpType{NewRegexpType(`\A[A-Z][\w]*(?:::[A-Z][\w]*)*\z`)})

var TYPE_STRING_OR_RANGE = NewVariantType([]eval.PType{
	stringType_NOT_EMPTY,
	DefaultSemVerRangeType()})

var TYPE_STRING_OR_URI = NewVariantType([]eval.PType{
	stringType_NOT_EMPTY,
	DefaultUriType()})

var TYPE_TYPE_REFERENCE_INIT = NewStructType([]*StructElement{
	NewStructElement2(KEY_NAME, TYPE_QUALIFIED_REFERENCE),
	NewStructElement2(KEY_VERSION_RANGE, TYPE_STRING_OR_RANGE),
	NewStructElement(NewOptionalType3(KEY_NAME_AUTHORITY), TYPE_STRING_OR_URI),
	NewStructElement(NewOptionalType3(KEY_ANNOTATIONS), TYPE_ANNOTATIONS)})

var TYPE_TYPESET_INIT = NewStructType([]*StructElement{
	NewStructElement(NewOptionalType3(eval.KEY_PCORE_URI), TYPE_STRING_OR_URI),
	NewStructElement2(eval.KEY_PCORE_VERSION, TYPE_STRING_OR_VERSION),
	NewStructElement(NewOptionalType3(KEY_NAME_AUTHORITY), TYPE_STRING_OR_URI),
	NewStructElement(NewOptionalType3(KEY_NAME), TYPE_OBJECT_NAME),
	NewStructElement(NewOptionalType3(KEY_VERSION), TYPE_STRING_OR_VERSION),
	NewStructElement(NewOptionalType3(KEY_TYPES),
		NewHashType(TYPE_SIMPLE_TYPE_NAME, NewVariantType([]eval.PType{
			DefaultTypeType(), TYPE_OBJECT_INIT_HASH}), NewIntegerType(1, math.MaxInt64))),
	NewStructElement(NewOptionalType3(KEY_REFERENCES),
		NewHashType(TYPE_SIMPLE_TYPE_NAME, TYPE_TYPE_REFERENCE_INIT, NewIntegerType(1, math.MaxInt64))),
	NewStructElement(NewOptionalType3(KEY_ANNOTATIONS), TYPE_ANNOTATIONS)})

func init() {
	oneArgCtor := func(ctx eval.Context, args []eval.PValue) eval.PValue {
		return NewTypeSetType2(ctx, args[0].(*HashValue), ctx.Loader())
	}
	TypeSet_Type = newObjectType2(`Pcore::TypeSetType`, Any_Type,
		WrapHash3(map[string]eval.PValue{
			`attributes`: SingletonHash2(`_pcore_init_hash`, TYPE_TYPESET_INIT)}),
		// Hash constructor is equal to the positional arguments constructor
		oneArgCtor, oneArgCtor)
}

type (
	typeSetReference struct {
		annotatable
		name          string
		owner         *TypeSetType
		nameAuthority eval.URI
		versionRange  semver.VersionRange
		typeSet       *TypeSetType
	}

	TypeSetType struct {
		annotatable
		hashKey            eval.HashKey
		dcToCcMap          map[string]string
		name               string
		nameAuthority      eval.URI
		pcoreURI           eval.URI
		pcoreVersion       semver.Version
		version            semver.Version
		types              *HashValue
		references         map[string]*typeSetReference
		loader             eval.Loader
		initHashExpression interface{}
	}
)

func newTypeSetReference(c eval.Context, t *TypeSetType, ref *HashValue) *typeSetReference {
	r := &typeSetReference{
		owner:         t,
		nameAuthority: uriArg(c, ref, KEY_NAME_AUTHORITY, t.nameAuthority),
		name:          stringArg(ref, KEY_NAME, ``),
		versionRange:  versionRangeArg(c, ref, KEY_VERSION_RANGE, nil),
	}
	r.annotatable.initialize(ref)
	return r
}

func (r *typeSetReference) initHash() map[string]eval.PValue {
	h := r.annotatable.initHash()
	if r.nameAuthority != r.owner.nameAuthority {
		h[KEY_NAME_AUTHORITY] = WrapString(string(r.nameAuthority))
	}
	h[KEY_NAME] = WrapString(r.name)
	h[KEY_VERSION_RANGE] = WrapSemVerRange(r.versionRange)
	return h
}

func (r *typeSetReference) Equals(other interface{}, g eval.Guard) bool {
	if or, ok := other.(*typeSetReference); ok {
		return r.name == or.name && r.nameAuthority == or.nameAuthority && r.versionRange == or.versionRange && r.typeSet.Equals(or.typeSet, g)
	}
	return false
}

func (r *typeSetReference) resolve(c eval.Context) {
	tn := eval.NewTypedName2(eval.TYPE, r.name, r.nameAuthority)
	loadedEntry := c.Loader().LoadEntry(c, tn)
	if loadedEntry != nil {
		if typeSet, ok := loadedEntry.Value().(*TypeSetType); ok {
			typeSet = typeSet.Resolve(c).(*TypeSetType)
			if r.versionRange.Includes(typeSet.version) {
				r.typeSet = typeSet
				return
			}
			panic(eval.Error(c, eval.EVAL_TYPESET_REFERENCE_MISMATCH, issue.H{`name`: r.owner.name, `ref_name`: r.name, `version_range`: r.versionRange, `actual`: typeSet.version}))
		}
	}
	var v interface{}
	if loadedEntry != nil {
		v = loadedEntry.Value()
	}
	if v == nil {
		panic(eval.Error(c, eval.EVAL_TYPESET_REFERENCE_UNRESOLVED, issue.H{`name`: r.owner.name, `ref_name`: r.name}))
	}
	var typeName string
	if vt, ok := v.(eval.PType); ok {
		typeName = vt.Name()
	} else if vv, ok := v.(eval.PValue); ok {
		typeName = vv.Type().Name()
	} else {
		typeName = fmt.Sprintf("%T", v)
	}
	panic(eval.Error(c, eval.EVAL_TYPESET_REFERENCE_BAD_TYPE, issue.H{`name`: r.owner.name, `ref_name`: r.name, `type_name`: typeName}))
}

var typeSetType_DEFAULT = &TypeSetType{
	name:          `DefaultTypeSet`,
	nameAuthority: eval.RUNTIME_NAME_AUTHORITY,
	pcoreURI:      eval.PCORE_URI,
	pcoreVersion:  eval.PCORE_VERSION,
	version:       semver.Zero,
}

var typeSetId = int64(0)

func AllocTypeSetType() *TypeSetType {
	return &TypeSetType{
		annotatable: annotatable{annotations: _EMPTY_MAP},
		dcToCcMap:   make(map[string]string, 17),
		hashKey:     eval.HashKey(fmt.Sprintf("\x00tTypeSet%d", atomic.AddInt64(&typeSetId, 1))),
		types:       _EMPTY_MAP,
		references:  map[string]*typeSetReference{},
	}
}

func (t *TypeSetType) Initialize(c eval.Context, args []eval.PValue) {
	if len(args) == 1 {
		if hash, ok := args[0].(eval.KeyedValue); ok {
			t.InitFromHash(c, hash)
			return
		}
	}
	panic(eval.Error(nil, eval.EVAL_FAILURE, issue.H{`message`: `internal error when creating an TypeSet data type`}))
}

func NewTypeSetType(na eval.URI, name string, initHashExpression interface{}) *TypeSetType {
	obj := AllocTypeSetType()
	obj.nameAuthority = na
	obj.name = name
	obj.initHashExpression = initHashExpression
	return obj
}

func NewTypeSetType2(c eval.Context, initHash *HashValue, loader eval.Loader) *TypeSetType {
	if initHash.IsEmpty() {
		return DefaultTypeSetType()
	}
	obj := AllocTypeSetType()
	obj.InitFromHash(c, initHash)
	obj.loader = loader
	return obj
}

func DefaultTypeSetType() *TypeSetType {
	return typeSetType_DEFAULT
}

func (t *TypeSetType) Annotations() *HashValue {
	return t.annotations
}

func (t *TypeSetType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
	// TODO: Visit typeset members
}

func (t *TypeSetType) Default() eval.PType {
	return typeSetType_DEFAULT
}

func (t *TypeSetType) Equals(other interface{}, guard eval.Guard) bool {
	if ot, ok := other.(*TypeSetType); ok {
		return t.name == ot.name && t.nameAuthority == ot.nameAuthority && t.pcoreURI == ot.pcoreURI && t.pcoreVersion == ot.pcoreVersion && t.version == ot.version
	}
	return false
}

func (t *TypeSetType) Generic() eval.PType {
	return DefaultTypeSetType()
}

func (t *TypeSetType) Parameters() []eval.PValue {
	if t.Equals(typeSetType_DEFAULT, nil) {
		return eval.EMPTY_VALUES
	}
	return []eval.PValue{t.InitHash()}
}

func (t *TypeSetType) InitFromHash(c eval.Context, initHash eval.KeyedValue) {
	eval.AssertInstance(c, `typeset initializer`, TYPE_TYPESET_INIT, initHash)
	t.name = stringArg(initHash, KEY_NAME, t.name)
	t.nameAuthority = uriArg(c, initHash, KEY_NAME_AUTHORITY, t.nameAuthority)

	t.pcoreVersion = versionArg(c, initHash, eval.KEY_PCORE_VERSION, nil)
	if !eval.PARSABLE_PCORE_VERSIONS.Includes(t.pcoreVersion) {
		panic(eval.Error(nil, eval.EVAL_UNHANDLED_PCORE_VERSION,
			issue.H{`name`: t.name, `expected_range`: eval.PARSABLE_PCORE_VERSIONS, `pcore_version`: t.pcoreVersion}))
	}
	t.pcoreURI = uriArg(c, initHash, eval.KEY_PCORE_URI, ``)
	t.version = versionArg(c, initHash, KEY_VERSION, nil)
	t.types = hashArg(initHash, KEY_TYPES)
	t.types.EachKey(func(kv eval.PValue) {
		key := kv.String()
		t.dcToCcMap[strings.ToLower(key)] = key
	})

	refs := hashArg(initHash, KEY_REFERENCES)
	if !refs.IsEmpty() {
		refMap := make(map[string]*typeSetReference, 7)
		rootMap := make(map[eval.URI]map[string][]semver.VersionRange, 7)
		refs.EachPair(func(k, v eval.PValue) {
			refAlias := k.String()

			if t.types.IncludesKey(k) {
				panic(eval.Error(nil, eval.EVAL_TYPESET_ALIAS_COLLIDES,
					issue.H{`name`: t.name, `ref_alias`: refAlias}))
			}

			if _, ok := refMap[refAlias]; ok {
				panic(eval.Error(nil, eval.EVAL_TYPESET_REFERENCE_DUPLICATE,
					issue.H{`name`: t.name, `ref_alias`: refAlias}))
			}

			ref := newTypeSetReference(c, t, v.(*HashValue))
			refName := ref.name
			refNA := ref.nameAuthority
			naRoots, found := rootMap[refNA]
			if !found {
				naRoots = make(map[string][]semver.VersionRange, 3)
				rootMap[refNA] = naRoots
			}

			if ranges, found := naRoots[refName]; found {
				for _, rng := range ranges {
					if rng.Intersection(ref.versionRange) != nil {
						panic(eval.Error(nil, eval.EVAL_TYPESET_REFERENCE_OVERLAP,
							issue.H{`name`: t.name, `ref_na`: refNA, `ref_name`: refName}))
					}
				}
				naRoots[refName] = append(ranges, ref.versionRange)
			} else {
				naRoots[refName] = []semver.VersionRange{ref.versionRange}
			}

			refMap[refAlias] = ref
			t.dcToCcMap[strings.ToLower(refAlias)] = refAlias
		})
		t.references = refMap
	}
	t.annotatable.initialize(initHash.(*HashValue))
}

func (t *TypeSetType) Get(c eval.Context, key string) (value eval.PValue, ok bool) {
	switch key {
	case eval.KEY_PCORE_URI:
		if t.pcoreURI == `` {
			return _UNDEF, true
		}
		return WrapURI2(string(t.pcoreURI)), true
	case eval.KEY_PCORE_VERSION:
		return WrapSemVer(t.pcoreVersion), true
	case KEY_NAME_AUTHORITY:
		if t.nameAuthority == `` {
			return _UNDEF, true
		}
		return WrapURI2(string(t.nameAuthority)), true
	case KEY_NAME:
		return WrapString(t.name), true
	case KEY_VERSION:
		if t.version == nil {
			return _UNDEF, true
		}
		return WrapSemVer(t.version), true
	case KEY_TYPES:
		return t.types, true
	case KEY_REFERENCES:
		return t.referencesHash(), true
	}
	return nil, false
}

func (t *TypeSetType) GetType(typedName eval.TypedName) (eval.PType, bool) {
	if !(typedName.Namespace() == eval.TYPE && typedName.NameAuthority() == t.nameAuthority) {
		return nil, false
	}

	segments := typedName.NameParts()
	first := segments[0]
	if len(segments) == 1 {
		if found, ok := t.GetType2(first); ok {
			return found, true
		}
	}

	if len(t.references) == 0 {
		return nil, false
	}

	tsRef, ok := t.references[first]
	if !ok {
		tsRef, ok = t.references[t.dcToCcMap[first]]
		if !ok {
			return nil, false
		}
	}

	typeSet := tsRef.typeSet
	switch len(segments) {
	case 1:
		return typeSet, true
	case 2:
		return typeSet.GetType2(segments[1])
	default:
		return typeSet.GetType(typedName.Child())
	}
}

func (t *TypeSetType) GetType2(name string) (eval.PType, bool) {
	v := t.types.Get6(name, func() eval.PValue {
		return t.types.Get5(t.dcToCcMap[name], nil)
	})
	if found, ok := v.(eval.PType); ok {
		return found, true
	}
	return nil, false
}

func (t *TypeSetType) referencesHash() *HashValue {
	if len(t.references) == 0 {
		return _EMPTY_MAP
	}
	entries := make([]*HashEntry, len(t.references))
	idx := 0
	for key, tr := range t.references {
		entries[idx] = WrapHashEntry2(key, WrapHash3(tr.initHash()))
		idx++
	}
	return WrapHash(entries)
}

func (t *TypeSetType) InitHash() eval.KeyedValue {
	return WrapHash3(t.initHash())
}

func (t *TypeSetType) initHash() map[string]eval.PValue {
	h := t.annotatable.initHash()
	if t.pcoreURI != `` {
		h[eval.KEY_PCORE_URI] = WrapURI2(string(t.pcoreURI))
	}
	h[eval.KEY_PCORE_VERSION] = WrapSemVer(t.pcoreVersion)
	if t.nameAuthority != `` {
		h[KEY_NAME_AUTHORITY] = WrapURI2(string(t.nameAuthority))
	}
	h[KEY_NAME] = WrapString(t.name)
	if t.version != nil {
		h[KEY_VERSION] = WrapSemVer(t.version)
	}
	if !t.types.IsEmpty() {
		h[KEY_TYPES] = t.types
	}
	if len(t.references) > 0 {
		h[KEY_REFERENCES] = t.referencesHash()
	}
	return h
}

func (t *TypeSetType) IsInstance(c eval.Context, o eval.PValue, g eval.Guard) bool {
	return t.IsAssignable(o.Type(), g)
}

func (t *TypeSetType) IsAssignable(other eval.PType, g eval.Guard) bool {
	if ot, ok := other.(*TypeSetType); ok {
		return t.Equals(typeSetType_DEFAULT, g) || t.Equals(ot, g)
	}
	return false
}

func (t *TypeSetType) MetaType() eval.ObjectType {
	return TypeSet_Type
}

func (t *TypeSetType) Name() string {
	return t.name
}

func (t *TypeSetType) NameAuthority() eval.URI {
	return t.nameAuthority
}

func (t *TypeSetType) Resolve(c eval.Context) eval.PType {
	ihe := t.initHashExpression
	if ihe == nil {
		return t
	}

	t.initHashExpression = nil
	var initHash *HashValue
	if lh, ok := ihe.(*parser.LiteralHash); ok {
		initHash = t.resolveLiteralHash(c, lh)
	} else {
		initHash = resolveTypeRefs(c, ihe.(*HashValue)).(*HashValue)
	}
	t.loader = c.Loader()
	t.InitFromHash(c, initHash)

	for _, ref := range t.references {
		ref.resolve(c)
	}
	tsaCtx := c.WithLoader(eval.NewTypeSetLoader(c.Loader(), t))
	t.types = t.types.MapValues(func(tp eval.PValue) eval.PValue {
		if rtp, ok := tp.(eval.ResolvableType); ok {
			return rtp.Resolve(tsaCtx)
		}
		return tp
	}).(*HashValue)

	return t
}

func (t *TypeSetType) resolveLiteralHash(c eval.Context, lh *parser.LiteralHash) *HashValue {
	entries := make([]*HashEntry, 0)
	types := map[string]parser.Expression{}

	var typesHash *HashValue

	for _, ex := range lh.Entries() {
		entry := ex.(*parser.KeyedEntry)
		key := c.Evaluate(entry.Key()).String()
		if key == KEY_TYPES || key == KEY_REFERENCES {
			if key == KEY_TYPES {
				typesHash = _EMPTY_MAP
			}

			// Avoid resolving qualified references into types
			if vh, ok := entry.Value().(*parser.LiteralHash); ok {
				xes := make([]*HashEntry, 0)
				for _, hex := range vh.Entries() {
					he := hex.(*parser.KeyedEntry)
					name := ``
					if qref, ok := he.Key().(*parser.QualifiedReference); ok {
						name = qref.Name()
					} else {
						name = c.Evaluate(he.Key()).String()
					}
					if key == KEY_TYPES {
						// Defer type resolution until all types are known
						types[name] = he.Value()
					} else {
						xes = append(xes, WrapHashEntry2(name, c.Evaluate(he.Value())))
					}
				}
				if key == KEY_REFERENCES {
					entries = append(entries, WrapHashEntry2(key, WrapHash(xes)))
				}
			} else {
				// Probably a bogus entry, will cause type error further on
				entries = append(entries, WrapHashEntry2(key, c.Evaluate(entry.Value())))
				if key == KEY_TYPES {
					typesHash = nil
				}
			}
		} else {
			entries = append(entries, WrapHashEntry2(key, c.Evaluate(entry.Value())))
		}
	}

	result := WrapHash(entries)
	nameAuth := t.resolveNameAuthority(result, c, lh)
	if len(types) > 0 {
		factory := parser.DefaultFactory()
		typesMap := make(map[string]eval.PValue, len(types))
		for typeName, value := range types {
			fullName := fmt.Sprintf(`%s::%s`, t.name, typeName)
			typedName := eval.NewTypedName2(eval.TYPE, fullName, nameAuth)
			if rde, ok := value.(*parser.ResourceDefaultsExpression); ok {
				// This is actually a <Parent> { <key-value entries> } notation. Convert to a literal
				// hash that contains the parent
				attrs := make([]parser.Expression, 0)
				if qr, ok := rde.TypeRef().(*parser.QualifiedReference); ok {
					name := qr.Name()
					if !(name == `Object` || name == `TypeSet`) {
						// the name `parent` is not allowed here
						for _, op := range rde.Operations() {
							if op.(issue.Named).Name() == KEY_PARENT {
								panic(eval.Error2(op, eval.EVAL_DUPLICATE_KEY, issue.H{`key`: KEY_PARENT}))
							}
						}
						attrs = append(attrs, factory.KeyedEntry(
							factory.QualifiedName(KEY_PARENT, qr.Locator(), qr.ByteOffset(), 0),
							qr, qr.Locator(), qr.ByteOffset(), qr.ByteLength()))
					}
				}
				for _, op := range rde.Operations() {
					if ao, ok := op.(*parser.AttributeOperation); ok {
						name := ao.Name()
						expr := ao.Value()
						attrs = append(attrs, factory.KeyedEntry(
							factory.QualifiedName(name, op.Locator(), op.ByteOffset(), len(name)),
							expr, ao.Locator(), ao.ByteOffset(), ao.ByteLength()))
					}
				}
				value = factory.Hash(attrs, value.Locator(), value.ByteOffset(), value.ByteLength())
			}
			tp := createTypeDefinition(nameAuth, fullName, value)
			typesMap[typeName] = tp
			c.Loader().(eval.DefiningLoader).SetEntry(typedName, eval.NewLoaderEntry(tp, value))
		}
		typesHash = WrapHash3(typesMap)
	}
	if typesHash != nil {
		result = WrapHash(append(entries, WrapHashEntry2(KEY_TYPES, typesHash)))
	}
	return result
}

func (t *TypeSetType) resolveNameAuthority(hash *HashValue, c eval.Context, location issue.Location) eval.URI {
	nameAuth := t.nameAuthority
	if nameAuth == `` {
		nameAuth = uriArg(c, hash, KEY_NAME_AUTHORITY, ``)
		if nameAuth == `` {
			if tsLoader, ok := c.Loader().(eval.TypeSetLoader); ok {
				nameAuth = tsLoader.TypeSet().(*TypeSetType).NameAuthority()
				if nameAuth == `` {
					n := t.name
					if n == `` {
						n = stringArg(hash, KEY_NAME, ``)
					}
					panic(eval.Error2(location, eval.EVAL_TYPESET_MISSING_NAME_AUTHORITY, issue.H{`name`: n}))
				}
			}
		}
	}
	return nameAuth
}

func (t *TypeSetType) String() string {
	return eval.ToString2(t, EXPANDED)
}

func (t *TypeSetType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *TypeSetType) Type() eval.PType {
	return &TypeType{t}
}
