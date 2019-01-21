package serialization

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/puppet-evaluator/eval"
	"github.com/lyraproj/puppet-evaluator/types"
)

// NewJsonStreamer creates a new streamer that will produce JSON when
// receiving values
func NewJsonStreamer(out io.Writer) ValueConsumer {
	return &jsonStreamer{out, firstInArray}
}

type jsonStreamer struct {
	out   io.Writer
	state int
}

// DataToJson streams the given value to a Json ValueConsumer using a
// Serializer. This function is deprecated Use a Serializer directly with
// NewJsonStreamer
func DataToJson(value eval.Value, out io.Writer) {
	he := make([]*types.HashEntry, 0, 2)
	he = append(he, types.WrapHashEntry2(`rich_data`, types.BooleanFalse))
	NewSerializer(eval.Puppet.RootContext(), types.WrapHash(he)).Convert(value, NewJsonStreamer(out))
	assertOk(out.Write([]byte("\n")))
}

func (j *jsonStreamer) AddArray(len int, doer eval.Doer) {
	j.delimit(func() {
		j.state = firstInArray
		assertOk(j.out.Write([]byte{'['}))
		doer()
		assertOk(j.out.Write([]byte{']'}))
	})
}

func (j *jsonStreamer) AddHash(len int, doer eval.Doer) {
	j.delimit(func() {
		assertOk(j.out.Write([]byte{'{'}))
		j.state = firstInObject
		doer()
		assertOk(j.out.Write([]byte{'}'}))
	})
}

func (j *jsonStreamer) Add(element eval.Value) {
	j.delimit(func() {
		j.write(element)
	})
}

func (j *jsonStreamer) AddRef(ref int) {
	j.delimit(func() {
		assertOk(fmt.Fprintf(j.out, `{"%s":%d}`, PCORE_REF_KEY, ref))
	})
}

func (j *jsonStreamer) CanDoBinary() bool {
	return false
}

func (j *jsonStreamer) CanDoComplexKeys() bool {
	return false
}

func (j *jsonStreamer) StringDedupThreshold() int {
	return 20
}

func (j *jsonStreamer) delimit(doer eval.Doer) {
	switch j.state {
	case firstInArray:
		doer()
		j.state = afterElement
	case firstInObject:
		doer()
		j.state = afterKey
	case afterKey:
		assertOk(j.out.Write([]byte{':'}))
		doer()
		j.state = afterValue
	case afterValue:
		assertOk(j.out.Write([]byte{','}))
		doer()
		j.state = afterKey
	default: // Element
		assertOk(j.out.Write([]byte{','}))
		doer()
	}
}

func (j *jsonStreamer) write(element eval.Value) {
	var v []byte
	var err error
	switch element.(type) {
	case eval.StringValue:
		v, err = json.Marshal(element.String())
	case eval.FloatValue:
		v, err = json.Marshal(element.(eval.FloatValue).Float())
	case eval.IntegerValue:
		v, err = json.Marshal(element.(eval.IntegerValue).Int())
	case eval.BooleanValue:
		v, err = json.Marshal(element.(eval.BooleanValue).Bool())
	default:
		v = []byte(`null`)
	}
	assertOk(0, err)
	assertOk(j.out.Write(v))
}

func assertOk(_ int, err error) {
	if err != nil {
		panic(eval.Error(eval.EVAL_FAILURE, issue.H{`message`: err}))
	}
}
