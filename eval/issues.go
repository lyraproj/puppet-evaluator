package eval

import . "github.com/puppetlabs/go-parser/issue"

const (
  EVAL_LHS_MUST_BE_QREF = `EVAL_LHS_MUST_BE_QREF`
  EVAL_UNHANDLED_EXPRESSION = `EVAL_UNHANDLED_EXPRESSION`
)

func init() {
  HardIssue(EVAL_LHS_MUST_BE_QREF, `LHS of [] expression must be a Type name`)
  HardIssue(EVAL_UNHANDLED_EXPRESSION, `Evaluator cannot handle an expression of type %T`)
}