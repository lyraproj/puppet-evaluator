# Puppet Evaluator

This is an evaluator for the AST produced by the [Puppet Language Parser](/puppetlabs/go-parser)

## Unit Testing

Unit tests are deliberately very scarse in this project. Tests for virtually everything is written
in the [Puppet Specification Language](https://docs.google.com/document/d/1VySubOiw8pD99OucVjk0ueyw2xlq4ky4WIWkV6Qoq64)
and then implemented in the [Puppet PSpec Evaluator](/puppetlabs/go-pspec). The evaluator for the
specification language is a specialization of this Puppet evaluator so it's a bit of a chicken and egg
problem.

### Expression evaluator status:

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
* [ ] class definition statements
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

#### Catalog production
* [ ] -> operator
* [ ] ~> operator
* [ ] <- operator
* [ ] <~ operator
* [ ] defined type statements
* [ ] node definition statements
* [ ] resource expressions
* [ ] resource metaparameters
* [ ] virtual resource expressions
* [ ] exported resource expressions
* [ ] resource defaults expressions (see note below)
* [ ] resource override expressions
* [ ] resource collection statements
* [ ] exported resource collection expressions (NYI: importing resources)

Note: resource default expressions use "static scoping" instead of "dynamic scoping"

Type system implemented:

* [x] Any
* [x] Array
* [x] Binary
* [x] Boolean
* [x] Callable
* [ ] Class
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
* [ ] Resource

Puppet functions implemented:

* [x] all
* [x] any
* [x] alert
* [x] assert_type
* [x] break
* [x] call
* [x] crit
* [x] debug
* [x] each
* [x] emerg
* [x] err
* [x] fail
* [x] filter
* [x] info
* [x] map
* [x] new
* [x] next
* [x] notice
* [x] reduce
* [x] return
* [x] type_of
* [x] warning
* [x] with
