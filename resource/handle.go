package resource

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"io"
	"github.com/puppetlabs/go-parser/issue"
)

type(
	Handle interface {
		eval.PuppetObject
	}

	handle struct {
		value eval.PuppetObject
		location issue.Location
	}
)

func (h *handle) Get(key string) (value eval.PValue, ok bool) {
	return h.value.Get(key)
}

func (h *handle) InitHash() eval.KeyedValue {
	return h.value.InitHash()
}

func (h *handle) String() string {
	return h.value.String()
}

func (h *handle) Equals(other interface{}, guard eval.Guard) bool {
	return h.value.Equals(other, guard)
}

func (h *handle) ToString(bld io.Writer, format eval.FormatContext, g eval.RDetect) {
	h.value.ToString(bld, format, g)
}

func (h *handle) Type() eval.PType {
	return h.value.Type()
}

func (h *handle) setLocation(location issue.Location) {
	h.location = location
}

func (h *handle) setValue(value eval.PuppetObject) {
	h.value = value
}
