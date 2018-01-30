package serialization

type (
	Extension byte

	// RichDataKey constants used in rich data hashes
	RichDataKey string

	// Symbol represents a symbolic string. Not used by this runtime but
	// type needed to retain the type in serailization
	Symbol string
)

const (
	// 0x00 - 0x0F are reserved for low-level serialization / tabulation extensions

	// Tabulation internal to the low level protocol reader/writer

	INNER_TABULATION = Extension(0x00)

	// Tabulation managed by the serializer / deserializer

	TABULATION = Extension(0x01)

	// 0x10 - 0x1F are reserved for structural extensions

	ARRAY_START        = Extension(0x10)
	MAP_START          = Extension(0x11)
	PCORE_OBJECT_START = Extension(0x12)
	OBJECT_START       = Extension(0x13)
	SENSITIVE_START    = Extension(0x14)

	// 0x20 - 0x2f reserved for special extension objects

	DEFAULT = Extension(0x20)
	COMMENT = Extension(0x21)

	// 0x30 - 0x7f reserved for mapping of specific runtime classes

	REGEXP         = Extension(0x30)
	TYPE_REFERENCE = Extension(0x31)
	SYMBOL         = Extension(0x32)
	TIME           = Extension(0x33)
	TIMESPAN       = Extension(0x34)
	VERSION        = Extension(0x35)
	VERSION_RANGE  = Extension(0x36)
	BINARY         = Extension(0x37)
	BASE64         = Extension(0x38)
	URI            = Extension(0x39)

	// PCORE_TYPE_KEY is the key used to signify the type of a serialized value
	PCORE_TYPE_KEY = RichDataKey(`__pcore_type__`)

  // PCORE_VALUE_KEY is used when the value can be represented as, and recreated from, a single string that can
  // be passed to a `from_string` method or an array of values that can be passed to the default
  // initializer method.
  PCORE_VALUE_KEY = RichDataKey(`__pcore_value__`)

  // PCORE_TYPE_HASH is used for hashes that contain keys that are not of type String
  PCORE_TYPE_HASH = RichDataKey(`Hash`)

  // PCORE_TYPE_SENSITIVE is the type key used for sensitive values
  PCORE_TYPE_SENSITIVE = RichDataKey(`Sensitive`)

  // PCORE_TYPE_SYMBOL is the type key used for symbols
  PCORE_TYPE_SYMBOL = RichDataKey(`Symbol`)

  // PCORE_TYPE_DEFAULT is the type key used for Default
  PCORE_TYPE_DEFAULT = RichDataKey(`Default`)

  // PCORE_LOCAL_REF_SYMBOL is the type key used for document local references
  PCORE_LOCAL_REF_SYMBOL = RichDataKey(`LocalRef`)
)
