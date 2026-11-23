package cli

import (
	"fmt"
	"runtime/debug"
)

// version describes the build: the module version when installed with
// `go install ...@vX`, otherwise the commit it was built from.
func version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "routine (unknown build)"
	}
	v := info.Main.Version
	if v == "" || v == "(devel)" {
		v = "dev"
	}
	rev, dirty := "", ""
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				dirty = "+dirty"
			}
		}
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}
	if rev != "" {
		return fmt.Sprintf("routine %s (%s%s, %s)", v, rev, dirty, info.GoVersion)
	}
	return fmt.Sprintf("routine %s (%s)", v, info.GoVersion)
}
