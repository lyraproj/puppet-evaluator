package functions

import (
	"github.com/puppetlabs/go-evaluator/eval"
	"github.com/puppetlabs/go-evaluator/types"
	"io/ioutil"
	"os"
	"github.com/puppetlabs/go-issues/issue"
)

func init() {
	eval.NewGoFunction(`binary_file`,
		func(d eval.Dispatch) {
			d.Param(`String`)
			d.Function(func(c eval.Context, args []eval.PValue) eval.PValue {
				path := args[0].String()
				bytes, err := ioutil.ReadFile(path)
				if err != nil {
					stat, serr := os.Stat(path)
					if serr != nil {
						if os.IsNotExist(serr) {
							panic(eval.Error(c, eval.EVAL_FILE_NOT_FOUND, issue.H{`path`: path}))
						}
						if os.IsPermission(serr) {
							panic(eval.Error(c, eval.EVAL_FILE_READ_DENIED, issue.H{`path`: path}))
						}
					} else {
						if stat.IsDir() {
							panic(eval.Error(c, eval.EVAL_IS_DIRECTORY, issue.H{`path`: path}))
						}
					}
					panic(eval.Error(c, eval.EVAL_FAILURE, issue.H{`message`: err.Error()}))
				}
				return types.WrapBinary(bytes)
			})
		})
}
