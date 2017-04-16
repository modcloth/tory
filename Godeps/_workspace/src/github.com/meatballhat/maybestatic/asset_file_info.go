package maybestatic

import (
	"os"
	"time"
)

type assetFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

func (afi *assetFileInfo) Name() string {
	return afi.name
}

func (afi *assetFileInfo) Size() int64 {
	return afi.size
}

func (afi *assetFileInfo) Mode() os.FileMode {
	return afi.mode
}

func (afi *assetFileInfo) ModTime() time.Time {
	return afi.modTime
}

func (afi *assetFileInfo) IsDir() bool {
	return false
}

func (afi *assetFileInfo) Sys() interface{} {
	return nil
}
