package semver

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type (
	badArgument struct {
		msg string
	}

	Version struct {
		major      int
		minor      int
		patch      int
		preRelease []interface{}
		build      []interface{}
	}
)

var vMIN_PRERELEASE []interface{}

var vPR_PART = `(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-]+[0-9A-Za-z-]*)`
var vPR_PARTS = vPR_PART + `(?:\.` + vPR_PART + `)*`
var vPART = `[0-9A-Za-z-]+`
var vPARTS = vPART + `(?:\.` + vPART + `)*`
var vPRERELEASE = `(?:-(` + vPR_PARTS + `))?`
var vBUILD = `(?:\+(` + vPARTS + `))?`
var vQUALIFIER = vPRERELEASE + vBUILD
var vNR = `(0|[1-9][0-9]*)`

var vPR_PARTS_PATTERN = regexp.MustCompile(`\A` + vPR_PARTS + `\z`)
var vPARTS_PATTERN = regexp.MustCompile(`\A` + vPARTS + `\z`)

var MAX = &Version{math.MaxInt64, math.MaxInt64, math.MaxInt64, nil, nil}
var MIN = &Version{0, 0, 0, vMIN_PRERELEASE, nil}
var VERSION_PATTERN = regexp.MustCompile(`\A` + vNR + `\.` + vNR + `\.` + vNR + vQUALIFIER + `\z`)

func NewVersion(major, minor, patch int) (version *Version, err error) {
	return NewVersion3(major, minor, patch, ``, ``)
}

func NewVersion2(major, minor, patch int, preRelease string) (version *Version, err error) {
	return NewVersion3(major, minor, patch, preRelease, ``)
}

func NewVersion3(major, minor, patch int, preRelease string, build string) (version *Version, err error) {
	if major < 0 || minor < 0 || patch < 0 {
		return nil, &badArgument{`Negative numbers not accepted in version`}
	}
	ps, err := splitParts(`pre-release`, preRelease, true)
	if err != nil {
		return nil, err
	}
	bs, err := splitParts(`build`, build, false)
	if err != nil {
		return nil, err
	}
	return &Version{major, minor, patch, ps, bs}, nil
}

// NewVersion4 is like NewVersion3 but panics rather than returning an error. Primarilly intended to be used internally
// by the VersionRange parser which in turn recovers the panic and returns error
func NewVersion4(major, minor, patch int, preRelease string, build string) *Version {
	if major < 0 || minor < 0 || patch < 0 {
		panic(&badArgument{`Negative numbers not accepted in version`})
	}
	ps, err := splitParts(`pre-release`, preRelease, true)
	if err != nil {
		panic(err)
	}
	bs, err := splitParts(`build`, build, false)
	if err != nil {
		panic(err)
	}
	return &Version{major, minor, patch, ps, bs}
}

func ParseVersion(str string) (version *Version, err error) {
	if group := VERSION_PATTERN.FindStringSubmatch(str); group != nil {
		major, _ := strconv.Atoi(group[1])
		minor, _ := strconv.Atoi(group[2])
		patch, _ := strconv.Atoi(group[3])
		return NewVersion3(major, minor, patch, group[4], group[5])
	}
	return nil, &badArgument{`The string '` + str + `' does not represent a valid semantic version`}
}

func (v *Version) CompareTo(o *Version) int {
	cmp := v.major - o.major
	if cmp == 0 {
		cmp = v.minor - o.minor
		if cmp == 0 {
			cmp = v.patch - o.patch
			if cmp == 0 {
				cmp = comparePreReleases(v.preRelease, o.preRelease)
			}
		}
	}
	return cmp
}

func (v *Version) Equals(ov *Version) bool {
	return v.TripletEquals(ov) && equalSegments(v.preRelease, ov.preRelease) && equalSegments(v.build, ov.build)
}

func (v *Version) IsStable() bool {
	return v.preRelease == nil
}

func (v *Version) NextPatch() *Version {
	return &Version{v.major, v.minor, v.patch + 1, nil, nil}
}

func (v *Version) String() string {
	bld := bytes.NewBufferString(``)
	v.ToString(bld)
	return bld.String()
}

func (v *Version) ToString(bld io.Writer) {
	fmt.Fprintf(bld, `%d.%d.%d`, v.major, v.minor, v.patch)
	if v.preRelease != nil {
		bld.Write([]byte(`-`))
		writeParts(v.preRelease, bld)
	}
	if v.build != nil {
		bld.Write([]byte(`+`))
		writeParts(v.build, bld)
	}
}

func (v *Version) TripletEquals(ov *Version) bool {
	return v.major == ov.major && v.minor == ov.minor && v.patch == ov.patch
}

func writeParts(parts []interface{}, bld io.Writer) {
	top := len(parts)
	if top > 0 {
		fmt.Fprintf(bld, `%v`, parts[0])
		for idx := 1; idx < top; idx++ {
			bld.Write([]byte(`.`))
			fmt.Fprintf(bld, `%v`, parts[idx])
		}
	}
}

func comparePreReleases(p1, p2 []interface{}) int {
	if p1 == nil {
		if p2 == nil {
			return 0
		}
		return 1
	}
	if p2 == nil {
		return -1
	}

	p1Size := len(p1)
	p2Size := len(p2)
	commonMax := p1Size
	if p1Size > p2Size {
		commonMax = p2Size
	}
	for idx := 0; idx < commonMax; idx++ {
		v1 := p1[idx]
		v2 := p2[idx]
		if i1, ok := v1.(int); ok {
			if i2, ok := v2.(int); ok {
				cmp := i1 - i2
				if cmp != 0 {
					return cmp
				}
				continue
			}
			return -1
		}

		if _, ok := v2.(int); ok {
			return 1
		}

		cmp := strings.Compare(v1.(string), v2.(string))
		if cmp != 0 {
			return cmp
		}
	}
	return p1Size - p2Size
}

func equalSegments(a, b []interface{}) bool {
	if a == nil {
		if b == nil {
			return true
		}
		return false
	}
	top := len(a)
	if b == nil || top != len(b) {
		return false
	}
	for idx := 0; idx < top; idx++ {
		if a[idx] != b[idx] {
			return false
		}
	}
	return true
}

func (v *Version) ToStable() *Version {
	return &Version{v.major, v.minor, v.patch, nil, v.build}
}

func mungePart(part string) interface{} {
	if i, err := strconv.ParseInt(part, 10, 64); err == nil {
		return int(i)
	}
	return part
}

func splitParts(tag, str string, stringToInt bool) ([]interface{}, error) {
	if str == `` {
		return nil, nil
	}

	pattern := vPARTS_PATTERN
	if stringToInt {
		pattern = vPR_PARTS_PATTERN
	}
	if !pattern.MatchString(str) {
		return nil, &badArgument{`Illegal characters in ` + tag}
	}

	parts := strings.Split(str, `.`)
	result := make([]interface{}, len(parts))
	for idx, sp := range parts {
		if stringToInt {
			result[idx] = mungePart(sp)
		} else {
			result[idx] = sp
		}
	}
	return result, nil
}
