package serialization

type (
	Extension byte

	// Symbol represents a symbolic string. Not used by this runtime but
	// type needed to retain the type in serailization
	Symbol string
)

const (
	// 0x00 - 0x0F are reserved for low-level serialization / tabulation extensions

	// Tabulation internal to the low level protocol reader/writer

	ExInnerTabulation = Extension(0x00)

	// Tabulation managed by the serializer / deserializer

	ExTabulation = Extension(0x01)

	// 0x10 - 0x1F are reserved for structural extensions

	ExArrayStart       = Extension(0x10)
	ExMapStart         = Extension(0x11)
	ExPcoreObjectStart = Extension(0x12)
	ExObjectStart      = Extension(0x13)
	ExSensitiveStart   = Extension(0x14)

	// 0x20 - 0x2f reserved for special extension objects

	ExDefault = Extension(0x20)
	ExComment = Extension(0x21)

	// 0x30 - 0x7f reserved for mapping of specific runtime classes

	ExRegexp        = Extension(0x30)
	ExTypeReference = Extension(0x31)
	ExSymbol        = Extension(0x32)
	ExTime          = Extension(0x33)
	ExTimespan      = Extension(0x34)
	ExVersion       = Extension(0x35)
	ExVersionRange  = Extension(0x36)
	ExBinary        = Extension(0x37)
	ExBase64        = Extension(0x38)
	ExUri           = Extension(0x39)

	// PcoreTypeKey is the key used to signify the type of a serialized value
	PcoreTypeKey = `__ptype`

	// PcoreValueKey is used when the value can be represented as, and recreated from, a single string that can
	// be passed to a `from_string` method or an array of values that can be passed to the default
	// initializer method.
	PcoreValueKey = `__pvalue`

	// PCORE_REF_KEY is the key used to signify the ordinal number of a previously serialized value. The
	// value is always an integer
	PCORE_REF_KEY = `__pref`

	// PCORE_TYPE_BINARY is used for binaries serialized using base64
	PCORE_TYPE_BINARY = `Binary`

	// PcoreTypeHash is used for hashes that contain keys that are not of type String
	PcoreTypeHash = `Hash`

	// PcoreTypeSensitive is the type key used for sensitive values
	PcoreTypeSensitive = `Sensitive`

	// PCORE_TYPE_SYMBOL is the type key used for symbols
	PCORE_TYPE_SYMBOL = `Symbol`

	// PcoreTypeDefault is the type key used for Default
	PcoreTypeDefault = `Default`

	// PCORE_LOCAL_REF_SYMBOL is the type key used for document local references
	PCORE_LOCAL_REF_SYMBOL = `LocalRef`
)
