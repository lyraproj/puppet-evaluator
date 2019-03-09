package evaluator

import "github.com/lyraproj/pcore/px"

func init() {
	px.NewObjectType(`Task`, `{
    attributes => {
      # Fully qualified name of the task
      name => { type => Pattern[/\A[a-z][a-z0-9_]*(?:::[a-z][a-z0-9_]*)*\z/] },

      # Full path to executable
      executable => { type => String },

      # Task description
      description => { type => Optional[String], value => undef },

      # Puppet Task version
      puppet_task_version => { type => Integer, value => 1 },

      # Type, description, and sensitive property of each parameter 
      parameters => {
        type => Optional[Hash[
          Pattern[/\A[a-z][a-z0-9_]*\z/],
          Struct[
            Optional[description] => String,
            Optional[sensitive] => Boolean,
            type => Type]]],
        value => undef
      },

       # Type, description, and sensitive property of each output 
      output => {
        type => Optional[Hash[
          Pattern[/\A[a-z][a-z0-9_]*\z/],
          Struct[
            Optional[description] => String,
            Optional[sensitive] => Boolean,
            type => Type]]],
        value => undef
      },

      supports_noop => { type => Boolean, value => false },
      input_method => { type => String, value => 'both' },
    }
 }`)
}
