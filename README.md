# Puppet Evaluator

This is an evaluator for the AST produced by the [Puppet Language Parser](/puppetlabs/go-parser)

## Unit Testing

Unit tests are deliberately very scarse in this project. Tests for virtually everything is written
in the [Puppet Specification Language](https://docs.google.com/document/d/1VySubOiw8pD99OucVjk0ueyw2xlq4ky4WIWkV6Qoq64)
and then implemented in the [Puppet PSpec Evaluator](https://github.com/puppetlabs/go-pspec/tree/master/eval_test/testdata). The evaluator for the
specification language is a specialization of this Puppet evaluator so it's a bit of a chicken and egg
problem.

## Implementation status

### Expression evaluator:

* [x] literal expressions
* [x] variable assignment
* [x] + operator
* [x] - operator (binary)
* [x] - operator (unary)
* [x] * operator (binary)
* [x] * operator (unary)
* [x] / operator
* [x] % operator
* [x] << operator
* [x] >> operator
* [x] logical `and`
* [x] logical `or`
* [x] logical `not`
* [x] == operator
* [x] != operator
* [x] =~ operator
* [x] !~ operator
* [x] < operator
* [x] <= operator
* [x] > operator
* [x] >= operator
* [x] `in` operator
* [x] if expressions
* [x] unless expressions
* [x] selector expressions
* [x] case expressions
* [x] method call expressions
* [x] function call expressions
* [x] lambdas
* [x] break statements
* [x] return statements
* [x] next statements
* [x] access expressions
* [x] global scope
* [x] local scope
* [x] node scope
* [x] string interpolation
* [x] Puppet type aliases
* [x] custom functions written in Puppet
* [x] custom functions written in Go
* [ ] custom data types written in Puppet
* [ ] custom data types written in Go
* [ ] external data binding (i.e. hiera)
* [ ] loading functions, plans, data types, and tasks from environment
* [ ] loading functions, plans, data types, and tasks from module
* [ ] ruby regexp (using Oniguruma)
* [x] type mismatch describer

#### Catalog and Resource related:

* [ ] -> operator
* [ ] ~> operator
* [ ] <- operator
* [ ] <~ operator
* [ ] class definition statements
* [ ] defined type statements
* [ ] node definition statements
* [ ] resource expressions
* [ ] resource metaparameters
* [ ] virtual resource expressions
* [ ] exported resource expressions
* [ ] resource defaults expressions
* [ ] resource override expressions
* [ ] resource collection statements
* [ ] exported resource collection expressions (NYI: importing resources)

### Data Type system:

* [x] Any
* [x] Array
* [x] Binary
* [x] Boolean
* [x] Callable
* [x] Collection
* [x] Data
* [x] Default
* [x] Enum
* [ ] Error
* [x] Float
* [x] Hash
* [x] Integer
* [x] Iterable
* [x] Iterator
* [x] NotUndef
* [x] Numeric
* [x] Optional
* [ ] Object
* [x] Pattern
* [x] Regexp
* [x] Runtime
* [x] ScalarData
* [x] Scalar
* [x] SemVer
* [x] SemVerRange
* [x] String
* [x] Struct
* [ ] Task
* [ ] Timespan
* [ ] Timestamp
* [x] Tuple
* [x] Type
* [x] Unit
* [x] Undef
* [ ] URI
* [x] Variant

* [ ] CatalogEntry
* [ ] Class
* [ ] Resource

### Puppet functions:

* [x] all
* [ ] annotate
* [x] any
* [x] alert
* [x] assert_type
* [ ] binary_file
* [x] break
* [x] call
* [x] crit
* [ ] convert_to
* [x] debug
* [ ] dig
* [x] each
* [x] emerg
* [ ] epp
* [x] err
* [ ] eyaml_data
* [x] fail
* [x] filter
* [ ] find_file
* [ ] hocon_data
* [x] info
* [ ] inline_epp
* [ ] json_data
* [ ] lest
* [ ] lookup
* [x] map
* [ ] match
* [x] new
* [x] next
* [x] notice
* [x] reduce
* [ ] regsubst
* [x] return
* [ ] reverse_each
* [ ] scanf
* [ ] slice
* [ ] step
* [x] sprintf
* [ ] strftime
* [ ] then
* [ ] tree_each
* [x] type
* [ ] unique
* [ ] unwrap
* [ ] versioncmp
* [x] warning
* [x] with
* [ ] yaml_data

#### Catalog and Resource related:

* [ ] contain
* [ ] defined
* [ ] include
* [ ] require

#### Concepts
* [x] Settings
* [x] String formatting
* [x] File based loader hierarchies
* [x] Issue based error reporting
* [x] Logging
* [ ] Facts as global variables
* [ ] Pcore serialization
* [ ] Pcore RichData <-> Data transformation
* [ ] Remote calls to other language runtimes
* [ ] Hiera 5
* [ ] Automatic Parameter Lookup
* [ ] CLI
* [ ] Puppet PAL
* [ ] Catalog production
