package types

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/hash"
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
	typ := initHash.Get5(keyType, nil)
	if tn, ok := typ.(stringValue); ok {
		a.typ = container.parseAttributeType(c, memberType, name, tn)
	} else {
		// Unchecked because type is guaranteed by earlier type assertion on the hash
		a.typ = typ.(eval.Type)
	}
	a.override = boolArg(initHash, keyOverride, false)
	a.final = boolArg(initHash, keyFinal, false)
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

func (a *annotatedMember) Type() eval.Type {
	return a.typ
}

func (a *annotatedMember) Override() bool {
	return a.override
}

func (a *annotatedMember) initHash() *hash.StringHash {
	h := a.annotatable.initHash()
	h.Put(keyType, a.typ)
	if a.final {
		h.Put(keyFinal, BooleanTrue)
	}
	if a.override {
		h.Put(keyOverride, BooleanTrue)
	}
	return h
}

func (a *annotatedMember) Final() bool {
	return a.final
}

// Checks if the this _member_ overrides an inherited member, and if so, that this member is declared with
// override = true and that the inherited member accepts to be overridden by this member.
func assertOverride(a eval.AnnotatedMember, parentMembers *hash.StringHash) {
	parentMember, _ := parentMembers.Get(a.Name(), nil).(eval.AnnotatedMember)
	if parentMember == nil {
		if a.Override() {
			panic(eval.Error(eval.OverriddenNotFound, issue.H{`label`: a.Label(), `feature_type`: a.FeatureType()}))
		}
	} else {
		assertCanBeOverridden(parentMember, a)
	}
}

func assertCanBeOverridden(a eval.AnnotatedMember, member eval.AnnotatedMember) {
	if a.FeatureType() != member.FeatureType() {
		panic(eval.Error(eval.OverrideMemberMismatch, issue.H{`member`: member.Label(), `label`: a.Label()}))
	}
	if a.Final() {
		aa, ok := a.(eval.Attribute)
		if !(ok && aa.Kind() == constant && member.(eval.Attribute).Kind() == constant) {
			panic(eval.Error(eval.OverrideOfFinal, issue.H{`member`: member.Label(), `label`: a.Label()}))
		}
	}
	if !member.Override() {
		panic(eval.Error(eval.OverrideIsMissing, issue.H{`member`: member.Label(), `label`: a.Label()}))
	}
	if !eval.IsAssignable(a.Type(), member.Type()) {
		panic(eval.Error(eval.OverrideTypeMismatch, issue.H{`member`: member.Label(), `label`: a.Label()}))
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
