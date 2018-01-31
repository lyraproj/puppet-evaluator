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

	EX_INNER_TABULATION = Extension(0x00)

	// Tabulation managed by the serializer / deserializer

	EX_TABULATION = Extension(0x01)

	// 0x10 - 0x1F are reserved for structural extensions

	EX_ARRAY_START        = Extension(0x10)
	EX_MAP_START          = Extension(0x11)
	EX_PCORE_OBJECT_START = Extension(0x12)
	EX_OBJECT_START       = Extension(0x13)
	EX_SENSITIVE_START    = Extension(0x14)

	// 0x20 - 0x2f reserved for special extension objects

	EX_DEFAULT = Extension(0x20)
	EX_COMMENT = Extension(0x21)

	// 0x30 - 0x7f reserved for mapping of specific runtime classes

	EX_REGEXP         = Extension(0x30)
	EX_TYPE_REFERENCE = Extension(0x31)
	EX_SYMBOL         = Extension(0x32)
	EX_TIME           = Extension(0x33)
	EX_TIMESPAN       = Extension(0x34)
	EX_VERSION        = Extension(0x35)
	EX_VERSION_RANGE  = Extension(0x36)
	EX_BINARY         = Extension(0x37)
	EX_BASE64         = Extension(0x38)
	EX_URI            = Extension(0x39)

	// PCORE_TYPE_KEY is the key used to signify the type of a serialized value
	PCORE_TYPE_KEY = `__pcore_type__`

	// PCORE_VALUE_KEY is used when the value can be represented as, and recreated from, a single string that can
	// be passed to a `from_string` method or an array of values that can be passed to the default
	// initializer method.
	PCORE_VALUE_KEY = `__pcore_value__`

	// PCORE_TYPE_HASH is used for hashes that contain keys that are not of type String
	PCORE_TYPE_HASH = `Hash`

	// PCORE_TYPE_SENSITIVE is the type key used for sensitive values
	PCORE_TYPE_SENSITIVE = `Sensitive`

	// PCORE_TYPE_SYMBOL is the type key used for symbols
	PCORE_TYPE_SYMBOL = `Symbol`

	// PCORE_TYPE_DEFAULT is the type key used for Default
	PCORE_TYPE_DEFAULT = `Default`

	// PCORE_LOCAL_REF_SYMBOL is the type key used for document local references
	PCORE_LOCAL_REF_SYMBOL = `LocalRef`
)
