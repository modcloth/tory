package maybestatic

import (
	"bytes"
	"os"
	"time"
)

var (
	bootTime = time.Now()
)

type fileAsset struct {
	*bytes.Reader
	name string
	fi   os.FileInfo
}

func newFileAsset(name string, b []byte) *fileAsset {
	return &fileAsset{
		Reader: bytes.NewReader(b),
		name:   name,
		fi: &assetFileInfo{
			name:    name,
			size:    int64(len(b)),
			mode:    os.FileMode(0644),
			modTime: bootTime,
		},
	}
}

func (fa *fileAsset) Readdir(count int) ([]os.FileInfo, error) {
	return []os.FileInfo{}, nil
}

func (fa *fileAsset) Stat() (os.FileInfo, error) {
	return fa.fi, nil
}

func (fa *fileAsset) Close() error {
	return nil
}
