# Puppet Evaluator

This is an evaluator for the AST produced by the [Puppet Language Parser](/lyraproj/puppet-parser)

## Unit Testing

Tests for virtually everything is written in the
[Puppet Specification Language](https://docs.google.com/document/d/1VySubOiw8pD99OucVjk0ueyw2xlq4ky4WIWkV6Qoq64)
and can be found under the [tests/testdata](https://github.com/lyraproj/puppet-evaluator/tree/master/tests/testdata)
directory.

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
* [x] custom data types written in Puppet
* [x] custom data types written in Go
* [ ] external data binding (i.e. hiera)
* [ ] loading functions, plans, data types, and tasks from environment
* [ ] loading functions, plans, data types, and tasks from module
* [ ] ruby regexp (using Oniguruma)
* [x] type mismatch describer

#### Catalog and Resource related:

* [x] -> operator
* [x] ~> operator
* [x] <- operator
* [x] <~ operator
* [ ] class definition statements
* [ ] defined type statements
* [ ] node definition statements
* [x] resource expressions
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
* [x] Error
* [x] Float
* [x] Hash
* [ ] Init
* [x] Integer
* [x] Iterable
* [x] Iterator
* [x] NotUndef
* [x] Numeric
* [x] Optional
* [x] Object
* [x] Pattern
* [x] Regexp
* [x] Runtime
* [x] ScalarData
* [x] Scalar
* [x] SemVer
* [x] SemVerRange
* [x] Sensitive
* [x] String
* [x] Struct
* [x] Target
* [x] Task
* [x] Timespan
* [x] Timestamp
* [x] Tuple
* [x] Type
* [x] TypeSet
* [x] Unit
* [x] Undef
* [x] URI
* [x] Variant

* [ ] CatalogEntry
* [ ] Class
* [x] Resource

### Puppet functions:

* [x] alert
* [x] all
* [ ] annotate
* [x] any
* [x] assert_type
* [ ] binary_file
* [x] break
* [x] call
* [x] crit
* [x] convert_to
* [x] debug
* [x] dig
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
* [x] lest
* [ ] lookup
* [x] map
* [x] match
* [x] new
* [x] next
* [x] notice
* [x] reduce
* [ ] regsubst
* [x] return
* [ ] reverse_each
* [ ] scanf
* [ ] slice
* [x] split
* [ ] step
* [x] sprintf
* [x] strftime
* [x] then
* [ ] tree_each
* [x] type
* [ ] unique
* [x] unwrap
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
* [x] Pcore RichData <-> Data transformation
* [ ] Remote calls to other language runtimes
* [ ] Hiera 5
* [ ] Automatic Parameter Lookup
* [ ] CLI
* [ ] Puppet PAL
* [ ] Catalog production
