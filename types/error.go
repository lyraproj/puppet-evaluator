package types

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-parser/issue"
)

func init() {
	newObjectType(`Error`, `{
	type_parameters => {
	  kind => Optional[Variant[String,Regexp,Type[Enum],Type[Pattern],Type[NotUndef],Type[Undef]]],
	  issue_code => Optional[Variant[String,Regexp,Type[Enum],Type[Pattern],Type[NotUndef],Type[Undef]]]
	},
	attributes => {
	  message => String[1],
	  kind => { type => Optional[String[1]], value => undef },
	  issue_code => { type => Optional[String[1]], value => undef },
	  partial_result => { type => Data, value => undef },
	  details => { type => Optional[Hash[String[1],Data]], value => undef },
	}
}`)
}

func NewError(c eval.Context, args...eval.PValue) eval.PuppetObject {
	if ctor, ok := eval.Load(c, eval.NewTypedName(eval.CONSTRUCTOR, `Error`)); ok {
		return ctor.(eval.Function).Call(c, nil, args...).(eval.PuppetObject)
	}
	panic(eval.Error(c, eval.EVAL_FAILURE, issue.H{`message`: `Error.new not found`}))
}
