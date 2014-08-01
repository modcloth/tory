package tory

import (
	"time"
)

var (
	VersionString   = "?"
	RevisionString  = "?"
	BranchString    = "?"
	GeneratedString = "?"
	CompiledTime    time.Time

	compiledTimeLayout = "2006-01-02T03:04:05Z"
)

func init() {
	var err error
	CompiledTime, err = time.Parse(compiledTimeLayout, GeneratedString)
	if err != nil {
		CompiledTime = time.Date(1955, time.November, 11, 5, 0, 0, 0, time.UTC)
	}
}
