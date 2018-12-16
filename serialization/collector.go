package serialization

import (
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
)

// A Collector receives streaming events and produces an eval.Value
type Collector interface {
	ValueConsumer

	// Value returns the created value. Must not be called until the consumption
	// of values is complete.
	Value() eval.Value
}

type collector struct {
	values []eval.Value
	stack  [][]eval.Value
}

// NewCollector returns a new Collector instance
func NewCollector() Collector {
	hm := &collector{}
	hm.Init()
	return hm
}

func (hm *collector) Init() {
	hm.values = make([]eval.Value, 0, 64)
	hm.stack = make([][]eval.Value, 1, 8)
	hm.stack[0] = make([]eval.Value, 0, 1)
}

func (hm *collector) AddArray(cap int, doer eval.Doer) {
	types.BuildArray(cap, func(ar *types.ArrayValue, elements []eval.Value) []eval.Value {
		hm.Add(ar)
		top := len(hm.stack)
		hm.stack = append(hm.stack, elements)
		doer()
		st := hm.stack[top]
		hm.stack = hm.stack[0:top]
		return st
	})
}

func (hm *collector) AddHash(cap int, doer eval.Doer) {
	types.BuildHash(cap, func(ar *types.HashValue, entries []*types.HashEntry) []*types.HashEntry {
		hm.Add(ar)
		top := len(hm.stack)
		hm.stack = append(hm.stack, make([]eval.Value, 0, cap*2))
		doer()
		st := hm.stack[top]
		hm.stack = hm.stack[0:top]

		top = len(st)
		for i := 0; i < top; i += 2 {
			entries = append(entries, types.WrapHashEntry(st[i], st[i+1]))
		}
		return entries
	})
}

func (hm *collector) Add(element eval.Value) {
	top := len(hm.stack) - 1
	hm.stack[top] = append(hm.stack[top], element)
	hm.values = append(hm.values, element)
}

func (hm *collector) AddRef(ref int) {
	top := len(hm.stack) - 1
	hm.stack[top] = append(hm.stack[top], hm.values[ref])
}

func (hm *collector) CanDoBinary() bool {
	return true
}

func (hm *collector) CanDoComplexKeys() bool {
	return true
}

func (hm *collector) StringDedupThreshold() int {
	return 0
}

func (hm *collector) Value() eval.Value {
	return hm.stack[0][0]
}
