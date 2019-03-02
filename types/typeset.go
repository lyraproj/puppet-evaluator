package types

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"runtime"
	"strings"
	"sync/atomic"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/hash"
	"github.com/lyraproj/puppet-evaluator/utils"
	"github.com/lyraproj/puppet-parser/parser"
	"github.com/lyraproj/semver/semver"
)

const (
	KeyNameAuthority = `name_authority`
	KeyReferences    = `references`
	KeyTypes         = `types`
	KeyVersion       = `version`
	KeyVersionRange  = `version_range`
)

var TypeSetMetaType eval.ObjectType

var TypeSimpleTypeName = NewPatternType([]*RegexpType{NewRegexpType(`\A[A-Z]\w*\z`)})
var TypeQualifiedReference = NewPatternType([]*RegexpType{NewRegexpType(`\A[A-Z][\w]*(?:::[A-Z][\w]*)*\z`)})

var typeStringOrVersion = NewVariantType(stringTypeNotEmpty, DefaultSemVerType())
var typeStringOrRange = NewVariantType(stringTypeNotEmpty, DefaultSemVerRangeType())
var typeStringOrUri = NewVariantType(stringTypeNotEmpty, DefaultUriType())

var typeTypeReferenceInit = NewStructType([]*StructElement{
	newStructElement2(keyName, TypeQualifiedReference),
	newStructElement2(KeyVersionRange, typeStringOrRange),
	NewStructElement(newOptionalType3(KeyNameAuthority), typeStringOrUri),
	NewStructElement(newOptionalType3(keyAnnotations), typeAnnotations)})

var typeTypesetInit = NewStructType([]*StructElement{
	NewStructElement(newOptionalType3(eval.KeyPcoreUri), typeStringOrUri),
	newStructElement2(eval.KeyPcoreVersion, typeStringOrVersion),
	NewStructElement(newOptionalType3(KeyNameAuthority), typeStringOrUri),
	NewStructElement(newOptionalType3(keyName), TypeTypeName),
	NewStructElement(newOptionalType3(KeyVersion), typeStringOrVersion),
	NewStructElement(newOptionalType3(KeyTypes),
		NewHashType(TypeSimpleTypeName,
			NewVariantType(DefaultTypeType(), TypeObjectInitHash), NewIntegerType(1, math.MaxInt64))),
	NewStructElement(newOptionalType3(KeyReferences),
		NewHashType(TypeSimpleTypeName, typeTypeReferenceInit, NewIntegerType(1, math.MaxInt64))),
	NewStructElement(newOptionalType3(keyAnnotations), typeAnnotations)})

func init() {
	oneArgCtor := func(ctx eval.Context, args []eval.Value) eval.Value {
		return newTypeSetType2(args[0].(*HashValue), ctx.Loader())
	}
	TypeSetMetaType = newObjectType3(`Pcore::TypeSet`, AnyMetaType,
		WrapStringToValueMap(map[string]eval.Value{
			`attributes`: SingletonHash2(`_pcore_init_hash`, typeTypesetInit)}),
		// Hash constructor is equal to the positional arguments constructor
		oneArgCtor, oneArgCtor)
}

type (
	typeSetReference struct {
		annotatable
		name          string
		owner         *typeSet
		nameAuthority eval.URI
		versionRange  semver.VersionRange
		typeSet       *typeSet
	}

	typeSet struct {
		annotatable
		hashKey            eval.HashKey
		dcToCcMap          map[string]string
		name               string
		nameAuthority      eval.URI
		pcoreURI           eval.URI
		pcoreVersion       semver.Version
		version            semver.Version
		typedName          eval.TypedName
		types              *HashValue
		references         map[string]*typeSetReference
		loader             eval.Loader
		initHashExpression interface{}
	}
)

func newTypeSetReference(t *typeSet, ref *HashValue) *typeSetReference {
	r := &typeSetReference{
		owner:         t,
		nameAuthority: uriArg(ref, KeyNameAuthority, t.nameAuthority),
		name:          stringArg(ref, keyName, ``),
		versionRange:  versionRangeArg(ref, KeyVersionRange, nil),
	}
	r.annotatable.initialize(ref)
	return r
}

func (r *typeSetReference) initHash() *hash.StringHash {
	h := r.annotatable.initHash()
	if r.nameAuthority != r.owner.nameAuthority {
		h.Put(KeyNameAuthority, stringValue(string(r.nameAuthority)))
	}
	h.Put(keyName, stringValue(r.name))
	h.Put(KeyVersionRange, WrapSemVerRange(r.versionRange))
	return h
}

func (r *typeSetReference) Equals(other interface{}, g eval.Guard) bool {
	if or, ok := other.(*typeSetReference); ok {
		return r.name == or.name && r.nameAuthority == or.nameAuthority && r.versionRange == or.versionRange && r.typeSet.Equals(or.typeSet, g)
	}
	return false
}

func (r *typeSetReference) resolve(c eval.Context) {
	tn := eval.NewTypedName2(eval.NsType, r.name, r.nameAuthority)
	loadedEntry := c.Loader().LoadEntry(c, tn)
	if loadedEntry != nil {
		if ts, ok := loadedEntry.Value().(*typeSet); ok {
			ts = ts.Resolve(c).(*typeSet)
			if r.versionRange.Includes(ts.version) {
				r.typeSet = ts
				return
			}
			panic(eval.Error(eval.TypesetReferenceMismatch, issue.H{`name`: r.owner.name, `ref_name`: r.name, `version_range`: r.versionRange, `actual`: ts.version}))
		}
	}
	var v interface{}
	if loadedEntry != nil {
		v = loadedEntry.Value()
	}
	if v == nil {
		panic(eval.Error(eval.TypesetReferenceUnresolved, issue.H{`name`: r.owner.name, `ref_name`: r.name}))
	}
	var typeName string
	if vt, ok := v.(eval.Type); ok {
		typeName = vt.Name()
	} else if vv, ok := v.(eval.Value); ok {
		typeName = vv.PType().Name()
	} else {
		typeName = fmt.Sprintf("%T", v)
	}
	panic(eval.Error(eval.TypesetReferenceBadType, issue.H{`name`: r.owner.name, `ref_name`: r.name, `type_name`: typeName}))
}

var typeSetTypeDefault = &typeSet{
	name:          `TypeSet`,
	nameAuthority: eval.RuntimeNameAuthority,
	pcoreURI:      eval.PcoreUri,
	pcoreVersion:  eval.PcoreVersion,
	version:       semver.Zero,
}

var typeSetId = int64(0)

func AllocTypeSetType() *typeSet {
	return &typeSet{
		annotatable: annotatable{annotations: emptyMap},
		dcToCcMap:   make(map[string]string, 17),
		hashKey:     eval.HashKey(fmt.Sprintf("\x00tTypeSet%d", atomic.AddInt64(&typeSetId, 1))),
		types:       emptyMap,
		references:  map[string]*typeSetReference{},
	}
}

func (t *typeSet) Initialize(c eval.Context, args []eval.Value) {
	if len(args) == 1 {
		if h, ok := args[0].(eval.OrderedMap); ok {
			t.InitFromHash(c, h)
			return
		}
	}
	panic(eval.Error(eval.Failure, issue.H{`message`: `internal error when creating an TypeSet data type`}))
}

func NewTypeSetType(na eval.URI, name string, initHashExpression *parser.LiteralHash) eval.TypeSet {
	obj := AllocTypeSetType()
	obj.nameAuthority = na
	obj.name = name
	obj.initHashExpression = initHashExpression
	return obj
}

func newTypeSetType3(na eval.URI, name string, initHash eval.OrderedMap) *typeSet {
	obj := AllocTypeSetType()
	obj.nameAuthority = na
	if name == `` {
		if initHash.IsEmpty() {
			return DefaultTypeSetType().(*typeSet)
		}
		name = initHash.Get5(keyName, emptyString).String()
	}
	obj.name = name
	obj.initHashExpression = initHash
	return obj
}

func newTypeSetType2(initHash eval.OrderedMap, loader eval.Loader) eval.TypeSet {
	obj := newTypeSetType3(loader.NameAuthority(), ``, initHash)
	obj.loader = loader
	return obj
}

func DefaultTypeSetType() eval.TypeSet {
	return typeSetTypeDefault
}

func (t *typeSet) Annotations() *HashValue {
	return t.annotations
}

func (t *typeSet) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
	// TODO: Visit typeset members
}

func (t *typeSet) Default() eval.Type {
	return typeSetTypeDefault
}

func (t *typeSet) Equals(other interface{}, guard eval.Guard) bool {
	if ot, ok := other.(*typeSet); ok {
		return t.name == ot.name && t.nameAuthority == ot.nameAuthority && t.pcoreURI == ot.pcoreURI && t.pcoreVersion.Equals(ot.pcoreVersion) && t.version.Equals(ot.version)
	}
	return false
}

func (t *typeSet) Generic() eval.Type {
	return DefaultTypeSetType()
}

func (t *typeSet) InitFromHash(c eval.Context, initHash eval.OrderedMap) {
	eval.AssertInstance(`typeset initializer`, typeTypesetInit, initHash)
	t.name = stringArg(initHash, keyName, t.name)
	t.nameAuthority = uriArg(initHash, KeyNameAuthority, t.nameAuthority)

	t.pcoreVersion = versionArg(initHash, eval.KeyPcoreVersion, nil)
	if !eval.ParsablePcoreVersions.Includes(t.pcoreVersion) {
		panic(eval.Error(eval.UnhandledPcoreVersion,
			issue.H{`name`: t.name, `expected_range`: eval.ParsablePcoreVersions, `pcore_version`: t.pcoreVersion}))
	}
	t.pcoreURI = uriArg(initHash, eval.KeyPcoreUri, ``)
	t.version = versionArg(initHash, KeyVersion, nil)
	t.types = hashArg(initHash, KeyTypes)
	t.types.EachKey(func(kv eval.Value) {
		key := kv.String()
		t.dcToCcMap[strings.ToLower(key)] = key
	})

	refs := hashArg(initHash, KeyReferences)
	if !refs.IsEmpty() {
		refMap := make(map[string]*typeSetReference, 7)
		rootMap := make(map[eval.URI]map[string][]semver.VersionRange, 7)
		refs.EachPair(func(k, v eval.Value) {
			refAlias := k.String()

			if t.types.IncludesKey(k) {
				panic(eval.Error(eval.TypesetAliasCollides,
					issue.H{`name`: t.name, `ref_alias`: refAlias}))
			}

			if _, ok := refMap[refAlias]; ok {
				panic(eval.Error(eval.TypesetReferenceDuplicate,
					issue.H{`name`: t.name, `ref_alias`: refAlias}))
			}

			ref := newTypeSetReference(t, v.(*HashValue))
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
						panic(eval.Error(eval.TypesetReferenceOverlap,
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

func (t *typeSet) Get(key string) (value eval.Value, ok bool) {
	switch key {
	case eval.KeyPcoreUri:
		if t.pcoreURI == `` {
			return undef, true
		}
		return WrapURI2(string(t.pcoreURI)), true
	case eval.KeyPcoreVersion:
		return WrapSemVer(t.pcoreVersion), true
	case KeyNameAuthority:
		if t.nameAuthority == `` {
			return undef, true
		}
		return WrapURI2(string(t.nameAuthority)), true
	case keyName:
		return stringValue(t.name), true
	case KeyVersion:
		if t.version == nil {
			return undef, true
		}
		return WrapSemVer(t.version), true
	case KeyTypes:
		return t.types, true
	case KeyReferences:
		return t.referencesHash(), true
	}
	return nil, false
}

func (t *typeSet) GetType(typedName eval.TypedName) (eval.Type, bool) {
	if !(typedName.Namespace() == eval.NsType && typedName.Authority() == t.nameAuthority) {
		return nil, false
	}

	segments := typedName.Parts()
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

func (t *typeSet) GetType2(name string) (eval.Type, bool) {
	v := t.types.Get6(name, func() eval.Value {
		return t.types.Get5(t.dcToCcMap[name], nil)
	})
	if found, ok := v.(eval.Type); ok {
		return found, true
	}
	return nil, false
}

func (t *typeSet) InitHash() eval.OrderedMap {
	return WrapStringPValue(t.initHash())
}

func (t *typeSet) IsInstance(o eval.Value, g eval.Guard) bool {
	return t.IsAssignable(o.PType(), g)
}

func (t *typeSet) IsAssignable(other eval.Type, g eval.Guard) bool {
	if ot, ok := other.(*typeSet); ok {
		return t.Equals(typeSetTypeDefault, g) || t.Equals(ot, g)
	}
	return false
}

func (t *typeSet) MetaType() eval.ObjectType {
	return TypeSetMetaType
}

func (t *typeSet) Name() string {
	return t.name
}

func (t *typeSet) NameAuthority() eval.URI {
	return t.nameAuthority
}

func (t *typeSet) TypedName() eval.TypedName {
	return t.typedName
}

func (t *typeSet) Parameters() []eval.Value {
	if t.Equals(typeSetTypeDefault, nil) {
		return eval.EmptyValues
	}
	return []eval.Value{t.InitHash()}
}

func (t *typeSet) Resolve(c eval.Context) eval.Type {
	ihe := t.initHashExpression
	if ihe == nil {
		return t
	}

	t.initHashExpression = nil
	var initHash *HashValue
	if lh, ok := ihe.(*parser.LiteralHash); ok {
		c.(eval.EvaluationContext).DoStatic(func() {
			initHash = t.resolveLiteralHash(c.(eval.EvaluationContext), lh)
		})
	} else {
		initHash = t.resolveDeferred(c, ihe.(eval.OrderedMap))
	}
	t.loader = c.Loader()
	t.InitFromHash(c, initHash)
	t.typedName = eval.NewTypedName2(eval.NsType, t.name, t.nameAuthority)

	for _, ref := range t.references {
		ref.resolve(c)
	}
	c.DoWithLoader(eval.NewTypeSetLoader(c.Loader(), t), func() {
		if t.nameAuthority == `` {
			t.nameAuthority = t.resolveNameAuthority(initHash, c, nil)
		}
		t.types = t.types.MapValues(func(tp eval.Value) eval.Value {
			if rtp, ok := tp.(eval.ResolvableType); ok {
				return rtp.Resolve(c)
			}
			return tp
		}).(*HashValue)
	})
	return t
}

func (t *typeSet) Types() eval.OrderedMap {
	return t.types
}

func (t *typeSet) Version() semver.Version {
	return t.version
}

func (t *typeSet) String() string {
	return eval.ToString2(t, Expanded)
}

func (t *typeSet) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	f := eval.GetFormat(s.FormatMap(), t.PType())
	switch f.FormatChar() {
	case 's', 'p':
		quoted := f.IsAlt() && f.FormatChar() == 's'
		if quoted || f.HasStringFlags() {
			bld := bytes.NewBufferString(``)
			t.basicTypeToString(bld, f, s, g)
			f.ApplyStringFlags(b, bld.String(), quoted)
		} else {
			t.basicTypeToString(b, f, s, g)
		}
	default:
		panic(s.UnsupportedFormat(t.PType(), `sp`, f))
	}
}

func (t *typeSet) PType() eval.Type {
	return &TypeType{t}
}

func (t *typeSet) basicTypeToString(b io.Writer, f eval.Format, s eval.FormatContext, g eval.RDetect) {
	if t.Equals(DefaultTypeSetType(), nil) {
		utils.WriteString(b, `TypeSet`)
		return
	}

	if ex, ok := s.Property(`expanded`); !(ok && ex == `true`) {
		name := t.Name()
		if ts, ok := s.Property(`typeSet`); ok {
			name = stripTypeSetName(ts, name)
		}
		utils.WriteString(b, name)
		return
	}
	s = s.WithProperties(map[string]string{`typeSet`: t.Name()})

	utils.WriteString(b, `TypeSet[{`)
	indent1 := s.Indentation()
	indent2 := indent1.Increase(f.IsAlt())
	indent3 := indent2.Increase(f.IsAlt())
	padding1 := ``
	padding2 := ``
	padding3 := ``
	if f.IsAlt() {
		padding1 = indent1.Padding()
		padding2 = indent2.Padding()
		padding3 = indent3.Padding()
	}

	cf := f.ContainerFormats()
	if cf == nil {
		cf = DefaultContainerFormats
	}

	ctx2 := eval.NewFormatContext2(indent2, s.FormatMap(), s.Properties())
	cti2 := eval.NewFormatContext2(indent2, cf, s.Properties())
	ctx3 := eval.NewFormatContext2(indent3, s.FormatMap(), s.Properties())

	first2 := true
	t.initHash().EachPair(func(key string, vi interface{}) {
		if first2 {
			first2 = false
		} else {
			utils.WriteString(b, `,`)
			if !f.IsAlt() {
				utils.WriteString(b, ` `)
			}
		}
		value := vi.(eval.Value)
		if f.IsAlt() {
			utils.WriteString(b, "\n")
			utils.WriteString(b, padding2)
		}
		utils.WriteString(b, key)
		utils.WriteString(b, ` => `)
		switch key {
		case `pcore_uri`, `pcore_version`, `name_authority`, `version`:
			utils.PuppetQuote(b, value.String())
		case `types`, `references`:
			// The keys should not be quoted in this hash
			utils.WriteString(b, `{`)
			first3 := true
			value.(*HashValue).EachPair(func(typeName, typ eval.Value) {
				if first3 {
					first3 = false
				} else {
					utils.WriteString(b, `,`)
					if !f.IsAlt() {
						utils.WriteString(b, ` `)
					}
				}
				if f.IsAlt() {
					utils.WriteString(b, "\n")
					utils.WriteString(b, padding3)
				}
				utils.WriteString(b, typeName.String())
				utils.WriteString(b, ` => `)
				typ.ToString(b, ctx3, g)
			})
			if f.IsAlt() {
				utils.WriteString(b, "\n")
				utils.WriteString(b, padding2)
			}
			utils.WriteString(b, "}")
		default:
			cx := cti2
			if isContainer(value, s) {
				cx = ctx2
			}
			value.ToString(b, cx, g)
		}
	})
	if f.IsAlt() {
		utils.WriteString(b, "\n")
		utils.WriteString(b, padding1)
	}
	utils.WriteString(b, "}]")
}

func (t *typeSet) initHash() *hash.StringHash {
	h := t.annotatable.initHash()
	if t.pcoreURI != `` {
		h.Put(eval.KeyPcoreUri, WrapURI2(string(t.pcoreURI)))
	}
	h.Put(eval.KeyPcoreVersion, WrapSemVer(t.pcoreVersion))
	if t.nameAuthority != `` {
		h.Put(KeyNameAuthority, WrapURI2(string(t.nameAuthority)))
	}
	h.Put(keyName, stringValue(t.name))
	if t.version != nil {
		h.Put(KeyVersion, WrapSemVer(t.version))
	}
	if !t.types.IsEmpty() {
		h.Put(KeyTypes, t.types)
	}
	if len(t.references) > 0 {
		h.Put(KeyReferences, t.referencesHash())
	}
	return h
}

func (t *typeSet) referencesHash() *HashValue {
	if len(t.references) == 0 {
		return emptyMap
	}
	entries := make([]*HashEntry, len(t.references))
	idx := 0
	for key, tr := range t.references {
		entries[idx] = WrapHashEntry2(key, WrapStringPValue(tr.initHash()))
		idx++
	}
	return WrapHash(entries)
}

func (t *typeSet) resolveLiteralHash(c eval.EvaluationContext, lh *parser.LiteralHash) *HashValue {
	entries := make([]*HashEntry, 0)
	types := map[string]parser.Expression{}

	var typesHash *HashValue

	for _, ex := range lh.Entries() {
		entry := ex.(*parser.KeyedEntry)
		key := eval.Evaluate(c, entry.Key()).String()
		if key == KeyTypes || key == KeyReferences {
			if key == KeyTypes {
				typesHash = emptyMap
			}

			// Avoid resolving qualified references into types
			if vh, ok := entry.Value().(*parser.LiteralHash); ok {
				xes := make([]*HashEntry, 0)
				for _, hex := range vh.Entries() {
					he := hex.(*parser.KeyedEntry)
					var name string
					if qr, ok := he.Key().(*parser.QualifiedReference); ok {
						name = qr.Name()
					} else {
						name = eval.Evaluate(c, he.Key()).String()
					}
					if key == KeyTypes {
						// Defer type resolution until all types are known
						types[name] = he.Value()
					} else {
						xes = append(xes, WrapHashEntry2(name, eval.Evaluate(c, he.Value())))
					}
				}
				if key == KeyReferences {
					entries = append(entries, WrapHashEntry2(key, WrapHash(xes)))
				}
			} else {
				// Probably a bogus entry, will cause type error further on
				entries = append(entries, WrapHashEntry2(key, eval.Evaluate(c, entry.Value())))
				if key == KeyTypes {
					typesHash = nil
				}
			}
		} else {
			entries = append(entries, WrapHashEntry2(key, eval.Evaluate(c, entry.Value())))
		}
	}

	result := WrapHash(entries)
	nameAuth := t.resolveNameAuthority(result, c, lh)
	if len(types) > 0 {
		factory := parser.DefaultFactory()
		typesMap := make(map[string]eval.Value, len(types))
		for typeName, value := range types {
			fullName := fmt.Sprintf(`%s::%s`, t.name, typeName)
			if rde, ok := value.(*parser.ResourceDefaultsExpression); ok {
				// This is actually a <Parent> { <key-value entries> } notation. Convert to a literal
				// hash that contains the parent
				attrs := make([]parser.Expression, 0)
				if qr, ok := rde.TypeRef().(*parser.QualifiedReference); ok {
					name := qr.Name()
					if !(name == `Object` || name == `TypeSet`) {
						// the name `parent` is not allowed here
						for _, op := range rde.Operations() {
							if op.(issue.Named).Name() == keyParent {
								panic(eval.Error2(op, eval.DuplicateKey, issue.H{`key`: keyParent}))
							}
						}
						attrs = append(attrs, factory.KeyedEntry(
							factory.QualifiedName(keyParent, qr.Locator(), qr.ByteOffset(), 0),
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
			typesMap[typeName] = createTypeDefinition(nameAuth, fullName, value)
		}
		typesHash = WrapStringToValueMap(typesMap)
	}
	if typesHash != nil {
		result = WrapHash(append(entries, WrapHashEntry2(KeyTypes, typesHash)))
	}
	return result
}

func (t *typeSet) resolveDeferred(c eval.Context, lh eval.OrderedMap) *HashValue {
	entries := make([]*HashEntry, 0)
	types := hash.NewStringHash(16)

	var typesHash *HashValue

	lh.Each(func(v eval.Value) {
		le := v.(eval.MapEntry)
		key := le.Key().String()
		if key == KeyTypes || key == KeyReferences {
			if key == KeyTypes {
				typesHash = emptyMap
			}

			// Avoid resolving qualified references into types
			if vh, ok := le.Value().(eval.OrderedMap); ok {
				xes := make([]*HashEntry, 0)
				vh.Each(func(v eval.Value) {
					he := v.(eval.MapEntry)
					var name string
					if qr, ok := he.Key().(*DeferredType); ok && len(qr.Parameters()) == 0 {
						name = qr.Name()
					} else if tr, ok := he.Key().(*TypeReferenceType); ok {
						name = tr.typeString
					} else {
						name = resolveValue(c, he.Key()).String()
					}
					if key == KeyTypes {
						// Defer type resolution until all types are known
						types.Put(name, he.Value())
					} else {
						xes = append(xes, WrapHashEntry2(name, resolveValue(c, he.Value())))
					}
				})
				if key == KeyReferences {
					entries = append(entries, WrapHashEntry2(key, WrapHash(xes)))
				}
			} else {
				// Probably a bogus entry, will cause type error further on
				entries = append(entries, resolveEntry(c, le).(*HashEntry))
				if key == KeyTypes {
					typesHash = nil
				}
			}
		} else {
			entries = append(entries, resolveEntry(c, le).(*HashEntry))
		}
	})

	result := WrapHash(entries)
	nameAuth := t.resolveNameAuthority(result, c, nil)
	if !types.IsEmpty() {
		es := make([]*HashEntry, 0, types.Len())
		types.EachPair(func(typeName string, value interface{}) {
			fullName := fmt.Sprintf(`%s::%s`, t.name, typeName)
			if rde, ok := value.(*DeferredType); ok && len(rde.params) == 1 {
				if ih, ok := rde.params[0].(eval.OrderedMap); ok {
					name := rde.Name()
					if !(name == `Object` || name == `TypeSet`) {
						// the name `parent` is not allowed here
						if pn, ok := ih.Get4(keyParent); ok {
							if name != pn.String() {
								panic(eval.Error(eval.DuplicateKey, issue.H{`key`: keyParent}))
							}
						} else {
							rde.params[0] = ih.Merge(SingletonHash2(keyParent, WrapString(name)))
						}
					}
				}
				es = append(es, WrapHashEntry2(typeName, NamedType(nameAuth, fullName, rde)))
			} else if tc, ok := value.(eval.Type); ok {
				es = append(es, WrapHashEntry2(typeName, tc))
			}
		})
		typesHash = WrapHash(es)
	}
	if typesHash != nil {
		result = WrapHash(append(entries, WrapHashEntry2(KeyTypes, typesHash)))
	}
	return result
}

func (t *typeSet) resolveNameAuthority(hash eval.OrderedMap, c eval.Context, location issue.Location) eval.URI {
	nameAuth := t.nameAuthority
	if nameAuth == `` {
		nameAuth = uriArg(hash, KeyNameAuthority, ``)
		if nameAuth == `` {
			nameAuth = c.Loader().NameAuthority()
		}
	}
	if nameAuth == `` {
		n := t.name
		if n == `` {
			n = stringArg(hash, keyName, ``)
		}
		var err error
		if location != nil {
			err = eval.Error2(location, eval.TypesetMissingNameAuthority, issue.H{`name`: n})
		} else {
			err = eval.Error(eval.TypesetMissingNameAuthority, issue.H{`name`: n})
		}
		panic(err)
	}
	return nameAuth
}

func newTypeSet(name, typeDecl string) eval.TypeSet {
	p := parser.CreateParser()
	_, fileName, fileLine, _ := runtime.Caller(1)
	expr, err := p.Parse(fileName, fmt.Sprintf(`type %s = TypeSet%s`, name, typeDecl), true)
	if err != nil {
		err = convertReported(err, fileName, fileLine)
		panic(err)
	}

	if ta, ok := expr.(*parser.TypeAlias); ok {
		rt, _ := CreateTypeDefinition(ta, eval.RuntimeNameAuthority)
		registerResolvableType(rt.(eval.ResolvableType))
		return rt.(eval.TypeSet)
	}
	panic(convertReported(eval.Error2(expr, eval.NoDefinition, issue.H{`source`: ``, `type`: eval.NsType, `name`: name}), fileName, fileLine))
}
