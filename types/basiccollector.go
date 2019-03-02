package types

import "github.com/lyraproj/puppet-evaluator/eval"

// A BasicCollector is an extendable basic implementation of the Collector interface
type BasicCollector struct {
	values []eval.Value
	stack  [][]eval.Value
}

// NewCollector returns a new Collector instance
func NewCollector() eval.Collector {
	hm := &BasicCollector{}
	hm.Init()
	return hm
}

func (hm *BasicCollector) Init() {
	hm.values = make([]eval.Value, 0, 64)
	hm.stack = make([][]eval.Value, 1, 8)
	hm.stack[0] = make([]eval.Value, 0, 1)
}

func (hm *BasicCollector) AddArray(cap int, doer eval.Doer) {
	BuildArray(cap, func(ar *ArrayValue, elements []eval.Value) []eval.Value {
		hm.Add(ar)
		top := len(hm.stack)
		hm.stack = append(hm.stack, elements)
		doer()
		st := hm.stack[top]
		hm.stack = hm.stack[0:top]
		return st
	})
}

func (hm *BasicCollector) AddHash(cap int, doer eval.Doer) {
	BuildHash(cap, func(ar *HashValue, entries []*HashEntry) []*HashEntry {
		hm.Add(ar)
		top := len(hm.stack)
		hm.stack = append(hm.stack, make([]eval.Value, 0, cap*2))
		doer()
		st := hm.stack[top]
		hm.stack = hm.stack[0:top]

		top = len(st)
		for i := 0; i < top; i += 2 {
			entries = append(entries, WrapHashEntry(st[i], st[i+1]))
		}
		return entries
	})
}

func (hm *BasicCollector) Add(element eval.Value) {
	top := len(hm.stack) - 1
	hm.stack[top] = append(hm.stack[top], element)
	hm.values = append(hm.values, element)
}

func (hm *BasicCollector) AddRef(ref int) {
	top := len(hm.stack) - 1
	hm.stack[top] = append(hm.stack[top], hm.values[ref])
}

func (hm *BasicCollector) CanDoBinary() bool {
	return true
}

func (hm *BasicCollector) CanDoComplexKeys() bool {
	return true
}

func (hm *BasicCollector) PopLast() eval.Value {
	top := len(hm.stack) - 1
	st := hm.stack[top]
	l := len(st) - 1
	if l >= 0 {
		v := st[l]
		hm.stack[top] = st[:l]
		hm.values = hm.values[:len(hm.values)-1]
		return v
	}
	return nil
}

func (hm *BasicCollector) StringDedupThreshold() int {
	return 0
}

func (hm *BasicCollector) Value() eval.Value {
	return hm.stack[0][0]
}
