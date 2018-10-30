package types

import "github.com/puppetlabs/go-evaluator/eval"

type attributesInfo struct {
	nameToPos                map[string]int
	posToName                map[int]string
	attributes               []eval.Attribute
	equalityAttributeIndexes []int
	requiredCount            int
}

func newAttributesInfo(attributes []eval.Attribute, requiredCount int, equality []string) *attributesInfo {
	nameToPos := make(map[string]int, len(attributes))
	posToName := make(map[int]string, len(attributes))
	for ix, at := range attributes {
		nameToPos[at.Name()] = ix
		posToName[ix] = at.Name()
	}

	ei := make([]int, len(equality))
	for ix, e := range equality {
		ei[ix] = nameToPos[e]
	}

	return &attributesInfo{attributes: attributes, nameToPos: nameToPos, posToName: posToName, equalityAttributeIndexes: ei, requiredCount: requiredCount}
}

func (ai *attributesInfo) NameToPos() map[string]int {
	return ai.nameToPos
}

func (ai *attributesInfo) PosToName() map[int]string {
	return ai.posToName
}

func (pi *attributesInfo) Attributes() []eval.Attribute {
	return pi.attributes
}

func (ai *attributesInfo) EqualityAttributeIndex() []int {
	return ai.equalityAttributeIndexes
}

func (ai *attributesInfo) RequiredCount() int {
	return ai.requiredCount
}

func (ai *attributesInfo) PositionalFromHash(hash eval.OrderedMap) []eval.Value {
	nameToPos := ai.NameToPos()
	va := make([]eval.Value, len(nameToPos))

	hash.EachPair(func(k eval.Value, v eval.Value) {
		if ix, ok := nameToPos[k.String()]; ok {
			va[ix] = v
		}
	})
	attrs := ai.Attributes()
	fillValueSlice(va, attrs)
	for i := len(va) - 1; i >= ai.RequiredCount(); i-- {
		if !attrs[i].Default(va[i]) {
			break
		}
		va = va[:i]
	}
	return va
}
