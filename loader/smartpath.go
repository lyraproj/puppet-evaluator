package loader

import (
	. "github.com/puppetlabs/go-evaluator/evaluator"
	"path/filepath"
	"regexp"
	"strings"
)

type (
	SmartPath interface {
		Loader() Loader
		GenericPath() string
		EffectivePath(name TypedName) string
		Extension() string
		RelativePath() string
		Namespace() Namespace
		IsMatchMany() bool
		PreferredOrigin(i []string) string
		TypedName(nameAuthority URI, relativePath string) TypedName
		Instantiator() Instantiator
		Indexed() bool
		SetIndexed()
	}

	smartPath struct {
		relativePath string
		loader       *fileBasedLoader
		namespace    Namespace
		extension    string

		// Paths are not supposed to contain module name
		moduleNameRelative bool
		matchMany          bool
		instantiator       Instantiator
		indexed            bool
	}
)

func (p *smartPath) Indexed() bool {
	return p.indexed
}

func (p *smartPath) SetIndexed() {
	p.indexed = true
}

func (p *smartPath) Loader() Loader {
	return p.loader
}

func (p *smartPath) EffectivePath(name TypedName) string {
	nameParts := name.NameParts()
	if p.moduleNameRelative {
		if len(nameParts) < 2 || nameParts[0] != p.loader.moduleName {
			return ``
		}
		nameParts = nameParts[1:]
	}

	parts := make([]string, 0, len(nameParts)+2)
	parts = append(parts, p.loader.path) // system, environment, or module root
	if p.relativePath != `` {
		parts = append(parts, p.relativePath)
	}
	parts = append(parts, nameParts...)
	return filepath.Join(parts...) + p.extension
}

func (p *smartPath) GenericPath() string {
	parts := make([]string, 0)
	parts = append(parts, p.loader.path) // system, environment, or module root
	if p.relativePath != `` {
		parts = append(parts, p.relativePath)
	}
	return filepath.Join(parts...)
}

func (p *smartPath) Namespace() Namespace {
	return p.namespace
}

func (p *smartPath) Extension() string {
	return p.extension
}

func (p *smartPath) RelativePath() string {
	return p.relativePath
}

func (p *smartPath) IsMatchMany() bool {
	return p.matchMany
}

func (p *smartPath) PreferredOrigin(origins []string) string {
	if len(origins) == 1 {
		return origins[0]
	}
	if p.namespace == TASK {
		// Prefer .json file if present
		for _, origin := range origins {
			if strings.HasSuffix(origin, `.json`) {
				return origin
			}
		}
	}
	return origins[0]
}

var dropExtension = regexp.MustCompile(`\.[^\\/]*\z`)

func (p *smartPath) TypedName(nameAuthority URI, relativePath string) TypedName {
	parts := strings.Split(relativePath, `/`)
	l := len(parts) - 1
	s := parts[l]
	if p.extension == `` {
		s = dropExtension.ReplaceAllLiteralString(s, ``)
	} else {
		s = s[:len(s)-len(p.extension)]
	}
	parts[l] = s

	if p.moduleNameRelative && !(len(parts) == 1 && (s == `init` || s == `init_typeset`)) {
		parts = append([]string{p.loader.moduleName}, parts...)
	}
	return NewTypedName2(p.namespace, strings.Join(parts, `::`), nameAuthority)
}

func (p *smartPath) Instantiator() Instantiator {
	return p.instantiator
}
