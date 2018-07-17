package eval

type Language int
const LangPuppet = Language(1)
const LangYAML = Language(2)
const LangJavaScript = Language(3)

func (l Language) String() string {
	switch l {
	case LangPuppet:
		return `Puppet`
	case LangYAML:
		return `YAML`
	case LangJavaScript:
		return `JavaScript`
	default:
		return `Unknown Language`
	}
}

