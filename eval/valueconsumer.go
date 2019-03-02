package eval

// A ValueConsumer is used by a Data streaming mechanism that maintains a reference index
// which is increased by one for each value that it streams. The reference index
// originates from zero.
type ValueConsumer interface {
	// CanDoBinary returns true if the value can handle binary efficiently. This tells
	// the Serializer to pass BinaryValue verbatim to Add
	CanDoBinary() bool

	// CanComplexKeys() returns true if complex values can be used as keys. If this
	// method returns false, all keys must be strings
	CanDoComplexKeys() bool

	// StringDedupThreshold returns the preferred threshold for dedup of strings. Strings
	// shorter than this threshold will not be subjected to de-duplication.
	StringDedupThreshold() int

	// AddArray starts a new array, calls the doer function, and then ends the Array.
	//
	// The callers reference index is increased by one.
	AddArray(len int, doer Doer)

	// AddHash starts a new hash., calls the doer function, and then ends the Hash.
	//
	// The callers reference index is increased by one.
	AddHash(len int, doer Doer)

	// Add adds the next value.
	//
	// Calls following a StartArray will add elements to the Array
	//
	// Calls following a StartHash will first add a key, then a value. This
	// repeats until End or StartArray is called.
	//
	// The callers reference index is increased by one.
	Add(element Value)

	// Add a reference to a previously added afterElement, hash, or array.
	AddRef(ref int)
}
