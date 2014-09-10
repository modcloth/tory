package maybestatic

import (
	"net/http"
	"strings"
)

type Dir struct {
	hd     http.Dir
	loader func(string) ([]byte, error)
}

func New(root string, loader func(string) ([]byte, error)) *Dir {
	return &Dir{
		hd:     http.Dir(root),
		loader: loader,
	}
}

func (d *Dir) Open(name string) (http.File, error) {
	f, err := d.hd.Open(name)
	if err == nil {
		return f, err
	}

	return d.openAsset(name)
}

func (d *Dir) openAsset(name string) (http.File, error) {
	name = strings.TrimLeft(name, "/")
	asset, err := d.loader(name)
	if err != nil {
		return nil, err
	}

	return newFileAsset(name, asset), nil
}
