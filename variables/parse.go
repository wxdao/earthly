package variables

import (
	"os"
	"strings"

	"github.com/earthly/earthly/variables/reserved"

	"github.com/pkg/errors"
)

// ProcessNonConstantVariableFunc is a function which takes in an expression and
// turns it into a state, target intput and arg index.
type ProcessNonConstantVariableFunc func(name string, expression string) (value string, argIndex int, err error)

// ParseCommandLineArgs parses a slice of old build args
// (the ones passed via --build-arg) and returns a new scope.
func ParseCommandLineArgs(args []string) (*Scope, error) {
	ret := NewScope()
	for _, arg := range args {
		splitArg := strings.SplitN(arg, "=", 2)
		if len(splitArg) < 1 {
			return nil, errors.Errorf("invalid build arg %s", splitArg)
		}
		key := splitArg[0]
		value := ""
		hasValue := false
		if len(splitArg) == 2 {
			value = splitArg[1]
			hasValue = true
		}
		if reserved.IsBuiltIn(key) {
			return nil, errors.Errorf("built-in arg %s cannot be passed on the command line", key)
		}
		if !hasValue {
			var found bool
			value, found = os.LookupEnv(key)
			if !found {
				return nil, errors.Errorf("env var %s not set", key)
			}
		}
		ret.AddInactive(key, value)
	}
	return ret, nil
}

// ParseArgs parses args passed as --build-arg to an Earthly command, such as BUILD or FROM.
func ParseArgs(args []string, pncvf ProcessNonConstantVariableFunc, current *Collection) (*Scope, error) {
	ret := NewScope()
	for _, arg := range args {
		name, variable, err := parseArg(arg, pncvf, current)
		if err != nil {
			return nil, errors.Wrapf(err, "parse build arg %s", arg)
		}
		ret.AddInactive(name, variable)
	}
	return ret, nil
}

func parseArg(arg string, pncvf ProcessNonConstantVariableFunc, current *Collection) (string, string, error) {
	var name string
	splitArg := strings.SplitN(arg, "=", 2)
	if len(splitArg) < 1 {
		return "", "", errors.Errorf("invalid build arg %s", splitArg)
	}
	name = splitArg[0]
	value := ""
	hasValue := false
	if len(splitArg) == 2 {
		value = splitArg[1]
		hasValue = true
	}
	if hasValue {
		if reserved.IsBuiltIn(name) {
			return "", "", errors.Errorf("value cannot be specified for built-in build arg %s", name)
		}
		return name, value, nil
	}
	v, ok := current.GetActive(name)
	if !ok {
		return "", "", errors.Errorf("value not specified for build arg %s and no value can be inferred", name)
	}
	return name, v, nil
}

// ContainsShell returns true for strings containing $(
// except for cases where escaped: e.g. \$(
// or cases with singlquotes: '$(...'
func ContainsShell(s string) bool {
	var escaped bool
	var singlequoted bool
	var last rune
	for _, c := range s {
		//fmt.Printf("got %s\n", string(c))
		if escaped {
			escaped = false
			last = 0
			continue
		}
		if c == '\\' {
			escaped = true
			last = 0
			continue
		}
		if c == '\'' {
			singlequoted = !singlequoted
			last = 0
			continue
		}
		if last == '$' && c == '(' && !singlequoted {
			return true
		}
		last = c
	}
	return false
}

// ParseEnvVars parses env vars from a slice of strings of the form "key=value".
func ParseEnvVars(envVars []string) *Scope {
	ret := NewScope()
	for _, envVar := range envVars {
		k, v, _ := ParseKeyValue(envVar)
		ret.AddActive(k, v)
	}
	return ret
}
