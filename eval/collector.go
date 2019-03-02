package eval

// A Collector receives streaming events and produces an Value
type Collector interface {
	ValueConsumer

	// PopLast pops the last value from the BasicCollector and returns it
	PopLast() Value

	// Value returns the created value. Must not be called until the consumption
	// of values is complete.
	Value() Value
}

// NewCollector returns a new Collector instance
var NewCollector func() Collector
