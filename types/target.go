package types

func init() {
	newType(`Target`, `{
	attributes => {
	  host => String[1],
	  options => { type => Hash[String[1], Data], value => {} }
	}
}`)
}
