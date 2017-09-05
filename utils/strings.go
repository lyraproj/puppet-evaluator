package utils

import "regexp"

func AllStrings(strings []string, predicate func(str string) bool) bool {
  for _, v := range strings {
    if !predicate(v) {
      return false
    }
  }
  return true
}

// Returns true if strings contains str
func ContainsString(strings []string, str string) bool {
  if str != `` {
    for _, v := range strings {
      if v == str {
        return true
      }
    }
  }
  return false
}

// Returns true if strings contains all entries in other
func ContainsAllStrings(strings []string, other []string) bool {
  for _, str := range other {
    if !ContainsString(strings, str) {
      return false
    }
  }
  return true
}

// Returns true if at least one of the regexps matches str
func MatchesString(regexps []*regexp.Regexp, str string) bool {
  if str != `` {
    for _, v := range regexps {
      if v.MatchString(str) {
        return true
      }
    }
  }
  return false
}

// Returns true if all strings are matched by at least one of the regexps
func MatchesAllStrings(regexps []*regexp.Regexp, strings []string) bool {
  for _, str := range strings {
    if !MatchesString(regexps, str) {
      return false
    }
  }
  return true
}
