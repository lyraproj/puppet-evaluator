package types

import (
	"io"

	"github.com/puppetlabs/go-evaluator/eval"
	"strconv"
	"fmt"
	"github.com/puppetlabs/go-evaluator/errors"
)

type NumericType struct{}

var numericType_DEFAULT = &NumericType{}

var Numeric_Type eval.ObjectType

func init() {
	Numeric_Type = newObjectType(`Pcore::NumericType`, `Pcore::ScalarDataType {}`, func(ctx eval.EvalContext, args []eval.PValue) eval.PValue {
		return DefaultNumericType()
	})

	newGoConstructor2(`Numeric`,
		func(t eval.LocalTypes) {
			t.Type(`Convertible`, `Variant[Undef, Integer, Float, Boolean, String, Timespan, Timestamp]`)
			t.Type(`NamedArgs`, `Struct[from => Convertible, Optional[abs] => Boolean]`)
		},

		func(d eval.Dispatch) {
			d.Param(`NamedArgs`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				h := args[0].(*HashValue)
				n := fromConvertible(h.Get5(`from`, eval.UNDEF))
				a := h.Get5(`abs`, nil)
				if a != nil && a.(*BooleanValue).Bool() {
					n = n.Abs()
				}
				return n
			})
		},

		func(d eval.Dispatch) {
			d.Param(`Convertible`)
			d.OptionalParam(`Boolean`)
			d.Function(func(c eval.EvalContext, args []eval.PValue) eval.PValue {
				n := fromConvertible(args[0])
				if len(args) > 1 && args[1].(*BooleanValue).Bool() {
					n = n.Abs()
				}
				return n
			})
		},
	)
}

func DefaultNumericType() *NumericType {
	return numericType_DEFAULT
}

func (t *NumericType) Accept(v eval.Visitor, g eval.Guard) {
	v(t)
}

func (t *NumericType) Equals(o interface{}, g eval.Guard) bool {
	_, ok := o.(*NumericType)
	return ok
}

func (t *NumericType) IsAssignable(o eval.PType, g eval.Guard) bool {
	switch o.(type) {
	case *IntegerType, *FloatType:
		return true
	default:
		return false
	}
}

func (t *NumericType) IsInstance(o eval.PValue, g eval.Guard) bool {
	switch o.Type().(type) {
	case *FloatType, *IntegerType:
		return true
	default:
		return false
	}
}

func (t *NumericType) MetaType() eval.ObjectType {
	return Numeric_Type
}

func (t *NumericType) Name() string {
	return `Numeric`
}

func (t *NumericType) String() string {
	return eval.ToString2(t, NONE)
}

func (t *NumericType) ToString(b io.Writer, s eval.FormatContext, g eval.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *NumericType) Type() eval.PType {
	return &TypeType{t}
}

func fromConvertible(c eval.PValue) eval.NumericValue {
	switch c.(type) {
	case *UndefValue:
		panic(`undefined_value`)
	case eval.NumericValue:
		return c.(eval.NumericValue)
	case *TimestampValue:
		return WrapFloat(c.(*TimestampValue).Float())
	case *TimespanValue:
		return WrapFloat(c.(*TimespanValue).Float())
	case *BooleanValue:
		b := c.(*BooleanValue).Bool()
		if b {
			return WrapInteger(1)
		}
		return WrapInteger(0)
	case *StringValue:
		s := c.String()
		if i, err := strconv.ParseInt(s, 0, 64); err == nil {
			return WrapInteger(i)
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return WrapFloat(f)
		}
		if len(s) > 2 && s[0] == '0' && (s[1] == 'b' || s[1] == 'B') {
			if i, err := strconv.ParseInt(s[2:], 2, 64); err == nil {
				return WrapInteger(i)
			}
		}
	}
	panic(errors.NewArgumentsError(`Numeric`, fmt.Sprintf(`Value of type %s cannot be converted to an Number`, c.Type().String())))
}
