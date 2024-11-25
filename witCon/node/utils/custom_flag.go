package utils

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"path"
	"strings"
)

type PathString struct {
	Value string
}

func (ps *PathString) String() string {
	return ps.Value
}

func (ps *PathString) Set(value string) error {
	ps.Value = expandPath(value)
	return nil
}

type PathFlag struct {
	Name  string
	Value PathString
	Usage string
}

func (pf PathFlag) String() string {
	fmtString := "%s %v\t%v"
	if len(pf.Value.Value) > 0 {
		fmtString = "%s \"%v\"\t%v"
	}
	return fmt.Sprintf(fmtString, prefixedNames(pf.Name), pf.Value.Value, pf.Usage)
}

func (pf PathFlag) Apply(set *flag.FlagSet) {
	eachName(pf.Name, func(name string) {
		set.Var(&pf.Value, pf.Name, pf.Usage)
	})
}

func (pf PathFlag) GetName() string {
	return pf.Name
}

func (pf *PathFlag) Set(value string) {
	pf.Value.Value = value
}

func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		if home := homeDir(); home != "" {
			p = home + p[1:]
		}
	}
	return path.Clean(os.ExpandEnv(p))
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}

func eachName(longName string, fn func(string)) {
	parts := strings.Split(longName, ",")
	for _, name := range parts {
		name = strings.Trim(name, " ")
		fn(name)
	}
}

func prefixFor(name string) (prefix string) {
	if len(name) == 1 {
		prefix = "-"
	} else {
		prefix = "--"
	}

	return
}

func prefixedNames(fullName string) (prefixed string) {
	parts := strings.Split(fullName, ",")
	for i, name := range parts {
		name = strings.Trim(name, " ")
		prefixed += prefixFor(name) + name
		if i < len(parts)-1 {
			prefixed += ", "
		}
	}
	return
}
