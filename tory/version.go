package tory

import (
	"os"
	"time"
)

var (
	BranchString    = "?"
	CompiledTime    time.Time
	GeneratedString = "?"
	RevisionString  = "?"
	VersionString   = "?"

	compiledTimeLayout = "2006-01-02T03:04:05Z"
)

func init() {
	var err error
	CompiledTime, err = time.Parse(compiledTimeLayout, GeneratedString)
	if err != nil {
		CompiledTime = time.Date(1955, time.November, 11, 5, 0, 0, 0, time.UTC)
	}

	os.Setenv("TORY_BRANCH", BranchString)
	os.Setenv("TORY_GENERATED", GeneratedString)
	os.Setenv("TORY_REVISION", RevisionString)
	os.Setenv("TORY_VERSION", VersionString)
}
