package semver

import (
	"bytes"
	"fmt"
	"io"
	. "regexp"
	"strconv"
)

type (
	abstractRange interface {
		asLowerBound() abstractRange
		asUpperBound() abstractRange
		equals(or abstractRange) bool
		includes(v *Version) bool
		isAbove(v *Version) bool
		isBelow(v *Version) bool
		isExcludeStart() bool
		isExcludeEnd() bool
		isLowerBound() bool
		isUpperBound() bool
		start() *Version
		end() *Version
		testPrerelease(v *Version) bool
		ToString(bld io.Writer)
	}

	simpleRange struct {
		Version
	}

	startEndRange struct {
		startCompare abstractRange
		endCompare   abstractRange
	}

	eqRange struct {
		simpleRange
	}

	gtRange struct {
		simpleRange
	}

	gtEqRange struct {
		simpleRange
	}

	ltRange struct {
		simpleRange
	}

	ltEqRange struct {
		simpleRange
	}

	VersionRange struct {
		originalString string
		ranges         []abstractRange
	}
)

var NR = `0|[1-9][0-9]*`
var XR = `(x|X|\*|` + NR + `)`

var PART = `(?:[0-9A-Za-z-]+)`
var PARTS = PART + `(?:\.` + PART + `)*`
var QUALIFIER = `(?:-(` + PARTS + `))?(?:\+(` + PARTS + `))?`

var PARTIAL = XR + `(?:\.` + XR + `(?:\.` + XR + QUALIFIER + `)?)?`

// The ~> isn`t in the spec but allowed
var SIMPLE = `([<>=~^]|<=|>=|~>|~=)?(?:` + PARTIAL + `)`
var SIMPLE_PATTERN = MustCompile(`\A` + SIMPLE + `\z`)

var OR_SPLIT = MustCompile(`\s*\|\|\s*`)
var SIMPLE_SPLIT = MustCompile(`\s+`)

var OP_WS_PATTERN = MustCompile(`([><=~^])(?:\s+|\s*v)`)

var HYPHEN = `(?:` + PARTIAL + `)\s+-\s+(?:` + PARTIAL + `)`
var HYPHEN_PATTERN = MustCompile(`\A` + HYPHEN + `\z`)

var _HIGHEST_LB = &gtRange{simpleRange{*MAX}}
var _LOWEST_LB = &gtEqRange{simpleRange{*MIN}}
var _LOWEST_UB = &ltRange{simpleRange{*MIN}}

var MATCH_ALL = &VersionRange{`*`, []abstractRange{_LOWEST_LB}}
var MATCH_NONE = &VersionRange{`<0.0.0`, []abstractRange{_LOWEST_UB}}

func ExactVersionRange(v *Version) *VersionRange {
	return &VersionRange{``, []abstractRange{&eqRange{simpleRange{*v}}}}
}

func FromVersions(start *Version, excludeStart bool, end *Version, excludeEnd bool) *VersionRange {
	var as abstractRange
	if excludeStart {
		as = &gtRange{simpleRange{*start}}
	} else {
		as = &gtEqRange{simpleRange{*start}}
	}
	var ae abstractRange
	if excludeEnd {
		ae = &ltRange{simpleRange{*end}}
	} else {
		ae = &ltEqRange{simpleRange{*end}}
	}
	return newVersionRange(``, []abstractRange{as, ae})
}

func ParseVersionRange(vr string) (result *VersionRange, err error) {
	if vr == `` {
		return nil, nil
	}

	defer func() {
		if err := recover(); err != nil {
			if ba, ok := err.(*badArgument); ok {
				err = ba
			} else {
				panic(err)
			}
		}
	}()

	vr = OP_WS_PATTERN.ReplaceAllString(vr, `$1`)
	rangeStrings := OR_SPLIT.Split(vr, -1)
	ranges := make([]abstractRange, 0, len(rangeStrings))
	for _, rangeStr := range rangeStrings {
		if rangeStr == `` {
			ranges = append(ranges, _LOWEST_LB)
			continue
		}

		if m := HYPHEN_PATTERN.FindStringSubmatch(rangeStr); m != nil {
			ranges = append(ranges, intersection(createGtEqRange(m, 1), createGtEqRange(m, 6)))
			continue
		}

		var simpleRange abstractRange
		for _, simple := range SIMPLE_SPLIT.Split(rangeStr, -1) {
			m := SIMPLE_PATTERN.FindStringSubmatch(simple)
			if m == nil {
				panic(&badArgument{fmt.Sprintf(`'%s' is not a valid version range`, simple)})
			}
			var rng abstractRange
			switch m[1] {
			case `~`, `~>`:
				rng = createTildeRange(m, 2)
			case `^`:
				rng = createCaretRange(m, 2)
			case `>`:
				rng = createGtRange(m, 2)
			case `>=`:
				rng = createGtEqRange(m, 2)
			case `<`:
				rng = createLtRange(m, 2)
			case `<=`:
				rng = createLtEqRange(m, 2)
			default:
				rng = createXRange(m, 2)
			}
			if simpleRange == nil {
				simpleRange = rng
			} else {
				simpleRange = intersection(simpleRange, rng)
			}
		}
		if simpleRange != nil {
			ranges = append(ranges, simpleRange)
		}
	}
	return newVersionRange(vr, ranges), nil
}

func (r *VersionRange) EndVersion() *Version {
	if len(r.ranges) == 1 {
		return r.ranges[0].end()
	}
	return nil
}

func (r *VersionRange) Equals(or *VersionRange) bool {
	top := len(r.ranges)
	if top != len(or.ranges) {
		return false
	}
	for idx, ar := range r.ranges {
		if !ar.equals(or.ranges[idx]) {
			return false
		}
	}
	return true
}

func (r *VersionRange) Includes(v *Version) bool {
	if v != nil {
		for _, ar := range r.ranges {
			if ar.includes(v) && (v.IsStable() || ar.testPrerelease(v)) {
				return true
			}
		}
	}
	return false
}

func (r *VersionRange) IsAsRestrictiveAs(o *VersionRange) bool {
arNext:
	for _, ar := range r.ranges {
		for _, ao := range o.ranges {
			is := intersection(ar, ao)
			if is != nil && asRestrictedAs(ar, ao) {
				continue arNext
			}
		}
		return false
	}
	return true
}

func (r *VersionRange) IsExcludeEnd() bool {
	if len(r.ranges) == 1 {
		return r.ranges[0].isExcludeEnd()
	}
	return false
}

func (r *VersionRange) IsExcludeStart() bool {
	if len(r.ranges) == 1 {
		return r.ranges[0].isExcludeStart()
	}
	return false
}

func (r *VersionRange) Merge(or *VersionRange) *VersionRange {
	return newVersionRange(``, append(r.ranges, or.ranges...))
}

func (r *VersionRange) NormalizedString() string {
	bld := bytes.NewBufferString(``)
	r.ToNormalizedString(bld)
	return bld.String()
}

func (r *VersionRange) StartVersion() *Version {
	if len(r.ranges) == 1 {
		return r.ranges[0].start()
	}
	return nil
}

func (r *VersionRange) String() string {
	bld := bytes.NewBufferString(``)
	r.ToString(bld)
	return r.String()
}

func (r *VersionRange) ToNormalizedString(bld io.Writer) {
	top := len(r.ranges)
	r.ranges[0].ToString(bld)
	for idx := 1; idx < top; idx++ {
		io.WriteString(bld, ` || `)
		r.ranges[idx].ToString(bld)
	}
}

func (r *VersionRange) ToString(bld io.Writer) {
	if r.originalString == `` {
		r.ToNormalizedString(bld)
	} else {
		io.WriteString(bld, r.originalString)
	}
}

func newVersionRange(vr string, ranges []abstractRange) *VersionRange {
	mergeHappened := true
	for len(ranges) > 1 && mergeHappened {
		mergeHappened = false
		result := make([]abstractRange, 0)
		for len(ranges) > 1 {
			unmerged := make([]abstractRange, 0)
			ln := len(ranges) - 1
			x := ranges[ln]
			ranges = ranges[:ln]
			for _, y := range ranges {
				merged := union(x, y)
				if merged == nil {
					unmerged = append(unmerged, y)
				} else {
					mergeHappened = true
					x = merged
				}
			}
			result = append([]abstractRange{x}, result...)
			ranges = unmerged
		}
		if len(ranges) > 0 {
			result = append(ranges, result...)
		}
		ranges = result
	}
	if len(ranges) == 0 {
		return MATCH_NONE
	}
	return &VersionRange{vr, ranges}
}

func createGtEqRange(rxGroup []string, startInMatcher int) abstractRange {
	major, ok := xDigit(rxGroup[startInMatcher])
	if !ok {
		return _LOWEST_LB
	}
	startInMatcher++
	minor, ok := xDigit(rxGroup[startInMatcher])
	if !ok {
		minor = 0
	}
	startInMatcher++
	patch, ok := xDigit(rxGroup[startInMatcher])
	if !ok {
		patch = 0
	}
	startInMatcher++
	preRelease := rxGroup[startInMatcher]
	startInMatcher++
	build := rxGroup[startInMatcher]
	return &gtEqRange{simpleRange{*NewVersion4(major, minor, patch, preRelease, build)}}
}

func createGtRange(rxGroup []string, startInMatcher int) abstractRange {
	major, ok := xDigit(rxGroup[startInMatcher])
	if !ok {
		return _LOWEST_LB
	}
	startInMatcher++
	minor, ok := xDigit(rxGroup[startInMatcher])
	if !ok {
		return &gtEqRange{simpleRange{Version{major + 1, 0, 0, nil, nil}}}
	}
	startInMatcher++
	patch, ok := xDigit(rxGroup[startInMatcher])
	if !ok {
		return &gtEqRange{simpleRange{Version{major, minor + 1, 0, nil, nil}}}
	}
	startInMatcher++
	preRelease := rxGroup[startInMatcher]
	startInMatcher++
	build := rxGroup[startInMatcher]
	return &gtRange{simpleRange{*NewVersion4(major, minor, patch, preRelease, build)}}
}

func createLtEqRange(rxGroup []string, startInMatcher int) abstractRange {
	major, ok := xDigit(rxGroup[startInMatcher])
	if !ok {
		return _LOWEST_UB
	}
	startInMatcher++
	minor, ok := xDigit(rxGroup[startInMatcher])
	if !ok {
		return &ltRange{simpleRange{Version{major + 1, 0, 0, nil, nil}}}
	}
	startInMatcher++
	patch, ok := xDigit(rxGroup[startInMatcher])
	if !ok {
		return &ltRange{simpleRange{Version{major, minor + 1, 0, nil, nil}}}
	}
	startInMatcher++
	preRelease := rxGroup[startInMatcher]
	startInMatcher++
	build := rxGroup[startInMatcher]
	return &ltEqRange{simpleRange{*NewVersion4(major, minor, patch, preRelease, build)}}
}

func createLtRange(rxGroup []string, startInMatcher int) abstractRange {
	major, ok := xDigit(rxGroup[startInMatcher])
	if !ok {
		return _LOWEST_UB
	}
	startInMatcher++
	minor, ok := xDigit(rxGroup[startInMatcher])
	if !ok {
		minor = 0
	}
	startInMatcher++
	patch, ok := xDigit(rxGroup[startInMatcher])
	if !ok {
		patch = 0
	}
	startInMatcher++
	preRelease := rxGroup[startInMatcher]
	startInMatcher++
	build := rxGroup[startInMatcher]
	return &ltRange{simpleRange{*NewVersion4(major, minor, patch, preRelease, build)}}
}

func createTildeRange(rxGroup []string, startInMatcher int) abstractRange {
	return allowPatchUpdates(rxGroup, startInMatcher, true)
}

func createCaretRange(rxGroup []string, startInMatcher int) abstractRange {
	major, ok := xDigit(rxGroup[startInMatcher])
	if !ok {
		return _LOWEST_LB
	}
	if major == 0 {
		return allowPatchUpdates(rxGroup, startInMatcher, true)
	}
	startInMatcher++
	return allowMinorUpdates(rxGroup, major, startInMatcher)
}

func createXRange(rxGroup []string, startInMatcher int) abstractRange {
	return allowPatchUpdates(rxGroup, startInMatcher, false)
}

func allowPatchUpdates(rxGroup []string, startInMatcher int, tildeOrCaret bool) abstractRange {
	major, ok := xDigit(rxGroup[startInMatcher])
	if !ok {
		return _LOWEST_LB
	}
	startInMatcher++
	minor, ok := xDigit(rxGroup[startInMatcher])
	if !ok {
		return &startEndRange{
			&gtEqRange{simpleRange{Version{major, 0, 0, nil, nil}}},
			&ltRange{simpleRange{Version{major + 1, 0, 0, nil, nil}}}}
	}
	startInMatcher++
	patch, ok := xDigit(rxGroup[startInMatcher])
	if !ok {
		return &startEndRange{
			&gtEqRange{simpleRange{Version{major, minor, 0, nil, nil}}},
			&ltRange{simpleRange{Version{major, minor + 1, 0, nil, nil}}}}
	}
	startInMatcher++
	preRelease := rxGroup[startInMatcher]
	startInMatcher++
	build := rxGroup[startInMatcher]
	if tildeOrCaret {
		return &startEndRange{
			&gtEqRange{simpleRange{*NewVersion4(major, minor, patch, preRelease, build)}},
			&ltRange{simpleRange{Version{major, minor + 1, 0, nil, nil}}}}
	}
	return &eqRange{simpleRange{*NewVersion4(major, minor, patch, preRelease, build)}}
}

func allowMinorUpdates(rxGroup []string, major int, startInMatcher int) abstractRange {
	minor, ok := xDigit(rxGroup[startInMatcher])
	if !ok {
		minor = 0
	}
	startInMatcher++
	patch, ok := xDigit(rxGroup[startInMatcher])
	if !ok {
		patch = 0
	}
	startInMatcher++
	preRelease := rxGroup[startInMatcher]
	startInMatcher++
	build := rxGroup[startInMatcher]
	return &startEndRange{
		&gtEqRange{simpleRange{*NewVersion4(major, minor, patch, preRelease, build)}},
		&ltRange{simpleRange{Version{major + 1, 0, 0, nil, nil}}}}
}

func xDigit(str string) (int, bool) {
	if str == `` || str == `x` || str == `X` || str == `*` {
		return 0, false
	}
	if str[0] != '0' {
		if i, err := strconv.ParseInt(str, 10, 64); err == nil {
			return int(i), true
		}
	}
	panic(&badArgument{`Illegal version triplet`})
}

func isOverlap(ra, rb abstractRange) bool {
	cmp := ra.start().CompareTo(rb.end())
	if cmp < 0 || cmp == 0 && !(ra.isExcludeStart() || rb.isExcludeEnd()) {
		cmp := rb.start().CompareTo(ra.end())
		return cmp < 0 || cmp == 0 && !(rb.isExcludeStart() || ra.isExcludeEnd())
	}
	return false
}

func asRestrictedAs(ra, vr abstractRange) bool {
	cmp := vr.start().CompareTo(ra.start())
	if cmp > 0 || (cmp == 0 && !ra.isExcludeStart() && vr.isExcludeStart()) {
		return false
	}

	cmp = vr.end().CompareTo(ra.end())
	return !(cmp < 0 || (cmp == 0 && !ra.isExcludeEnd() && vr.isExcludeEnd()))
}

func intersection(ra, rb abstractRange) abstractRange {
	cmp := ra.start().CompareTo(rb.end())
	if cmp > 0 {
		return nil
	}

	if cmp == 0 {
		if ra.isExcludeStart() || rb.isExcludeEnd() {
			return nil
		}
		return &eqRange{simpleRange{*ra.start()}}
	}

	cmp = rb.start().CompareTo(ra.end())
	if cmp > 0 {
		return nil
	}

	if cmp == 0 {
		if rb.isExcludeStart() || ra.isExcludeEnd() {
			return nil
		}
		return &eqRange{simpleRange{*rb.start()}}
	}

	cmp = ra.start().CompareTo(rb.start())
	var start abstractRange
	if cmp < 0 {
		start = rb
	} else if cmp > 0 {
		start = ra
	} else if ra.isExcludeStart() {
		start = ra
	} else {
		start = rb
	}

	cmp = ra.end().CompareTo(rb.end())
	var end abstractRange
	if cmp > 0 {
		end = rb
	} else if cmp < 0 {
		end = ra
	} else if ra.isExcludeEnd() {
		end = ra
	} else {
		end = rb
	}

	if !end.isUpperBound() {
		return start
	}

	if !start.isLowerBound() {
		return end
	}

	return &startEndRange{start.asLowerBound(), end.asUpperBound()}
}

func fromTo(ra, rb abstractRange) abstractRange {
	var startR abstractRange
	if ra.isExcludeStart() {
		startR = &gtRange{simpleRange{*ra.start()}}
	} else {
		startR = &gtEqRange{simpleRange{*ra.start()}}
	}
	var endR abstractRange
	if rb.isExcludeEnd() {
		endR = &ltRange{simpleRange{*rb.end()}}
	} else {
		endR = &ltEqRange{simpleRange{*rb.end()}}
	}
	return &startEndRange{startR, endR}
}

func union(ra, rb abstractRange) abstractRange {
	if ra.includes(rb.start()) || rb.includes(ra.start()) {
		var start *Version
		var excludeStart bool
		cmp := ra.start().CompareTo(rb.start())
		if cmp < 0 {
			start = ra.start()
			excludeStart = ra.isExcludeStart()
		} else if cmp > 0 {
			start = rb.start()
			excludeStart = rb.isExcludeStart()
		} else {
			start = ra.start()
			excludeStart = ra.isExcludeStart() && rb.isExcludeStart()
		}

		var end *Version
		var excludeEnd bool
		cmp = ra.end().CompareTo(rb.end())
		if cmp > 0 {
			end = ra.end()
			excludeEnd = ra.isExcludeEnd()
		} else if cmp < 0 {
			end = rb.end()
			excludeEnd = rb.isExcludeEnd()
		} else {
			end = ra.end()
			excludeEnd = ra.isExcludeEnd() && rb.isExcludeEnd()
		}

		var startR abstractRange
		if excludeStart {
			startR = &gtRange{simpleRange{*start}}
		} else {
			startR = &gtEqRange{simpleRange{*start}}
		}
		var endR abstractRange
		if excludeEnd {
			endR = &ltRange{simpleRange{*end}}
		} else {
			endR = &ltEqRange{simpleRange{*end}}
		}
		return &startEndRange{startR, endR}
	}
	if ra.isExcludeStart() && rb.isExcludeStart() && ra.start().CompareTo(rb.start()) == 0 {
		return fromTo(ra, rb)
	}
	if ra.isExcludeEnd() && !rb.isExcludeStart() && ra.end().CompareTo(rb.start()) == 0 {
		return fromTo(ra, rb)
	}
	if rb.isExcludeEnd() && !ra.isExcludeStart() && rb.end().CompareTo(ra.start()) == 0 {
		return fromTo(rb, ra)
	}
	if !ra.isExcludeEnd() && !rb.isExcludeStart() && ra.end().NextPatch().CompareTo(rb.start()) == 0 {
		return fromTo(ra, rb)
	}
	if !rb.isExcludeEnd() && !ra.isExcludeStart() && rb.end().NextPatch().CompareTo(ra.start()) == 0 {
		return fromTo(rb, ra)
	}
	return nil
}

func (r *startEndRange) asLowerBound() abstractRange {
	return r.startCompare
}

func (r *startEndRange) asUpperBound() abstractRange {
	return r.endCompare
}

func (r *startEndRange) equals(o abstractRange) bool {
	if or, ok := o.(*startEndRange); ok {
		return r.startCompare.equals(or.startCompare) && r.endCompare.equals(or.endCompare)
	}
	return false
}

func (r *startEndRange) includes(v *Version) bool {
	return r.startCompare.includes(v) && r.endCompare.includes(v)
}

func (r *startEndRange) isAbove(v *Version) bool {
	return r.startCompare.isAbove(v)
}

func (r *startEndRange) isBelow(v *Version) bool {
	return r.endCompare.isBelow(v)
}

func (r *startEndRange) isExcludeStart() bool {
	return r.startCompare.isExcludeStart()
}

func (r *startEndRange) isExcludeEnd() bool {
	return r.endCompare.isExcludeEnd()
}

func (r *startEndRange) isLowerBound() bool {
	return r.startCompare.isLowerBound()
}

func (r *startEndRange) isUpperBound() bool {
	return r.endCompare.isUpperBound()
}

func (r *startEndRange) start() *Version {
	return r.startCompare.start()
}

func (r *startEndRange) end() *Version {
	return r.endCompare.end()
}

func (r *startEndRange) testPrerelease(v *Version) bool {
	return r.startCompare.testPrerelease(v) || r.endCompare.testPrerelease(v)
}

func (r *startEndRange) ToString(bld io.Writer) {
	r.startCompare.ToString(bld)
	bld.Write([]byte(` `))
	r.endCompare.ToString(bld)
}

func (r *simpleRange) asLowerBound() abstractRange {
	return _HIGHEST_LB
}

func (r *simpleRange) asUpperBound() abstractRange {
	return _LOWEST_UB
}

func (r *simpleRange) isAbove(v *Version) bool {
	return false
}

func (r *simpleRange) isBelow(v *Version) bool {
	return false
}

func (r *simpleRange) isExcludeStart() bool {
	return false
}

func (r *simpleRange) isExcludeEnd() bool {
	return false
}

func (r *simpleRange) isLowerBound() bool {
	return false
}

func (r *simpleRange) isUpperBound() bool {
	return false
}

func (r *simpleRange) start() *Version {
	return MIN
}

func (r *simpleRange) end() *Version {
	return MAX
}

func (r *simpleRange) testPrerelease(v *Version) bool {
	return !r.IsStable() && r.TripletEquals(v)
}

// Equals
func (r *eqRange) asLowerBound() abstractRange {
	return r
}

func (r *eqRange) asUpperBound() abstractRange {
	return r
}

func (r *eqRange) equals(o abstractRange) bool {
	if or, ok := o.(*eqRange); ok {
		return r.Equals(&or.Version)
	}
	return false
}

func (r *eqRange) includes(v *Version) bool {
	return r.CompareTo(v) == 0
}

func (r *eqRange) isAbove(v *Version) bool {
	return r.CompareTo(v) > 0
}

func (r *eqRange) isBelow(v *Version) bool {
	return r.CompareTo(v) < 0
}

func (r *eqRange) isLowerBound() bool {
	return !r.Equals(MIN)
}

func (r *eqRange) isUpperBound() bool {
	return !r.Equals(MAX)
}

func (r *eqRange) start() *Version {
	return &r.Version
}

func (r *eqRange) end() *Version {
	return &r.Version
}

// GreaterEquals
func (r *gtEqRange) asLowerBound() abstractRange {
	return r
}

func (r *gtEqRange) equals(o abstractRange) bool {
	if or, ok := o.(*gtEqRange); ok {
		return r.Equals(&or.Version)
	}
	return false
}

func (r *gtEqRange) includes(v *Version) bool {
	return r.CompareTo(v) <= 0
}

func (r *gtEqRange) isAbove(v *Version) bool {
	return r.CompareTo(v) > 0
}

func (r *gtEqRange) isLowerBound() bool {
	return !r.Equals(MIN)
}

func (r *gtEqRange) start() *Version {
	return &r.Version
}

func (r *gtEqRange) ToString(bld io.Writer) {
	bld.Write([]byte(`>=`))
	r.Version.ToString(bld)
}

// Greater
func (r *gtRange) asLowerBound() abstractRange {
	return r
}

func (r *gtRange) equals(o abstractRange) bool {
	if or, ok := o.(*gtRange); ok {
		return r.Equals(&or.Version)
	}
	return false
}

func (r *gtRange) includes(v *Version) bool {
	return r.CompareTo(v) < 0
}

func (r *gtRange) isAbove(v *Version) bool {
	if r.IsStable() {
		v = v.ToStable()
	}
	return r.CompareTo(v) >= 0
}

func (r *gtRange) isExcludeStart() bool {
	return true
}

func (r *gtRange) isLowerBound() bool {
	return true
}

func (r *gtRange) start() *Version {
	return &r.Version
}

func (r *gtRange) ToString(bld io.Writer) {
	bld.Write([]byte(`>`))
	r.Version.ToString(bld)
}

// Less Equal
func (r *ltEqRange) asUpperBound() abstractRange {
	return r
}

func (r *ltEqRange) equals(o abstractRange) bool {
	if or, ok := o.(*ltEqRange); ok {
		return r.Equals(&or.Version)
	}
	return false
}

func (r *ltEqRange) includes(v *Version) bool {
	return r.CompareTo(v) >= 0
}

func (r *ltEqRange) isBelow(v *Version) bool {
	return r.CompareTo(v) < 0
}

func (r *ltEqRange) isUpperBound() bool {
	return !r.Equals(MAX)
}

func (r *ltEqRange) end() *Version {
	return &r.Version
}

func (r *ltEqRange) ToString(bld io.Writer) {
	bld.Write([]byte(`<=`))
	r.Version.ToString(bld)
}

// Less
func (r *ltRange) asUpperBound() abstractRange {
	return r
}

func (r *ltRange) equals(o abstractRange) bool {
	if or, ok := o.(*ltRange); ok {
		return r.Equals(&or.Version)
	}
	return false
}

func (r *ltRange) includes(v *Version) bool {
	return r.CompareTo(v) > 0
}

func (r *ltRange) isBelow(v *Version) bool {
	if r.IsStable() {
		v = v.ToStable()
	}
	return r.CompareTo(v) <= 0
}

func (r *ltRange) isUpperBound() bool {
	return true
}

func (r *ltRange) end() *Version {
	return &r.Version
}

func (r *ltRange) ToString(bld io.Writer) {
	bld.Write([]byte(`<`))
	r.Version.ToString(bld)
}

func (b *badArgument) Error() string {
	return b.msg
}
