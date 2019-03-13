package evaluator

import "github.com/lyraproj/pcore/px"

func init() {
	px.NewObjectType(`Target`, `{
	attributes => {
	  host => String[1],
	  options => { type => Hash[String[1], Data], value => {} }
	}
}`)
}
