package serialization

const (
	// PcoreTypeKey is the key used to signify the type of a serialized value
	PcoreTypeKey = `__ptype`

	// PcoreValueKey is used when the value can be represented as, and recreated from, a single string that can
	// be passed to a `from_string` method or an array of values that can be passed to the default
	// initializer method.
	PcoreValueKey = `__pvalue`

	// PcoreRefKey is the key used to signify the ordinal number of a previously serialized value. The
	// value is always an integer
	PcoreRefKey = `__pref`

	// PcoreTypeBinary is used for binaries serialized using base64
	PcoreTypeBinary = `Binary`

	// PcoreTypeHash is used for hashes that contain keys that are not of type String
	PcoreTypeHash = `Hash`

	// PcoreTypeSensitive is the type key used for sensitive values
	PcoreTypeSensitive = `Sensitive`

	// PcoreTypeDefault is the type key used for Default
	PcoreTypeDefault = `Default`
)
