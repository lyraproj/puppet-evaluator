package types

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/hash"
	"github.com/puppetlabs/go-issues/issue"
)

type annotatedMember struct {
	annotatable
	name      string
	container *objectType
	typ       eval.Type
	override  bool
	final     bool
}

func (a *annotatedMember) initialize(c eval.Context, memberType, name string, container *objectType, initHash *HashValue) {
	a.annotatable.initialize(initHash)
	a.name = name
	a.container = container
	typ := initHash.Get5(KEY_TYPE, nil)
	if tn, ok := typ.(*StringValue); ok {
		a.typ = container.parseAttributeType(c, memberType, name, tn)
	} else {
		// Unchecked because type is guaranteed by earlier type assersion on the hash
		a.typ = typ.(eval.Type)
	}
	a.override = boolArg(initHash, KEY_OVERRIDE, false)
	a.final = boolArg(initHash, KEY_FINAL, false)
}

func (a *annotatedMember) Accept(v eval.Visitor, g eval.Guard) {
	a.typ.Accept(v, g)
	visitAnnotations(a.annotations, v, g)
}

func (a *annotatedMember) Annotations() *HashValue {
	return a.annotations
}

func (a *annotatedMember) Call(c eval.Context, receiver eval.Value, block eval.Lambda, args []eval.Value) eval.Value {
	// TODO:
	panic("implement me")
}

func (a *annotatedMember) Name() string {
	return a.name
}

func (a *annotatedMember) Container() eval.ObjectType {
	return a.container
}

func (a *annotatedMember) PType() eval.Type {
	return a.typ
}

func (a *annotatedMember) Override() bool {
	return a.override
}

func (a *annotatedMember) initHash() *hash.StringHash {
	h := a.annotatable.initHash()
	h.Put(KEY_TYPE, a.typ)
	if a.final {
		h.Put(KEY_FINAL, WrapBoolean(true))
	}
	if a.override {
		h.Put(KEY_OVERRIDE, WrapBoolean(true))
	}
	return h
}

func (a *annotatedMember) Final() bool {
	return a.final
}

// Checks if the this _member_ overrides an inherited member, and if so, that this member is declared with
// override = true and that the inherited member accepts to be overridden by this member.
func assertOverride(c eval.Context, a eval.AnnotatedMember, parentMembers *hash.StringHash) {
	parentMember, _ := parentMembers.Get(a.Name(), nil).(eval.AnnotatedMember)
	if parentMember == nil {
		if a.Override() {
			panic(eval.Error(eval.EVAL_OVERRIDDEN_NOT_FOUND, issue.H{`label`: a.Label(), `feature_type`: a.FeatureType()}))
		}
	} else {
		assertCanBeOverridden(c, parentMember, a)
	}
}

func assertCanBeOverridden(c eval.Context, a eval.AnnotatedMember, member eval.AnnotatedMember) {
	if a.FeatureType() != member.FeatureType() {
		panic(eval.Error(eval.EVAL_OVERRIDE_MEMBER_MISMATCH, issue.H{`member`: member.Label(), `label`: a.Label()}))
	}
	if a.Final() {
		aa, ok := a.(eval.Attribute)
		if !(ok && aa.Kind() == CONSTANT && member.(eval.Attribute).Kind() == CONSTANT) {
			panic(eval.Error(eval.EVAL_OVERRIDE_OF_FINAL, issue.H{`member`: member.Label(), `label`: a.Label()}))
		}
	}
	if !member.Override() {
		panic(eval.Error(eval.EVAL_OVERRIDE_IS_MISSING, issue.H{`member`: member.Label(), `label`: a.Label()}))
	}
	if !eval.IsAssignable(a.PType(), member.PType()) {
		panic(eval.Error(eval.EVAL_OVERRIDE_TYPE_MISMATCH, issue.H{`member`: member.Label(), `label`: a.Label()}))
	}
}

// Visit the keys of an annotations map. All keys are known to be types
func visitAnnotations(a *HashValue, v eval.Visitor, g eval.Guard) {
	if a != nil {
		a.EachKey(func(key eval.Value) {
			key.(eval.Type).Accept(v, g)
		})
	}
}
