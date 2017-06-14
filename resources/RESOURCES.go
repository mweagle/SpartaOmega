package resources

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sync"
	"time"
)

type _escLocalFS struct{}

var _escLocal _escLocalFS

type _escStaticFS struct{}

var _escStatic _escStaticFS

type _escDirectory struct {
	fs   http.FileSystem
	name string
}

type _escFile struct {
	compressed string
	size       int64
	modtime    int64
	local      string
	isDir      bool

	once sync.Once
	data []byte
	name string
}

func (_escLocalFS) Open(name string) (http.File, error) {
	f, present := _escData[path.Clean(name)]
	if !present {
		return nil, os.ErrNotExist
	}
	return os.Open(f.local)
}

func (_escStaticFS) prepare(name string) (*_escFile, error) {
	f, present := _escData[path.Clean(name)]
	if !present {
		return nil, os.ErrNotExist
	}
	var err error
	f.once.Do(func() {
		f.name = path.Base(name)
		if f.size == 0 {
			return
		}
		var gr *gzip.Reader
		b64 := base64.NewDecoder(base64.StdEncoding, bytes.NewBufferString(f.compressed))
		gr, err = gzip.NewReader(b64)
		if err != nil {
			return
		}
		f.data, err = ioutil.ReadAll(gr)
	})
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (fs _escStaticFS) Open(name string) (http.File, error) {
	f, err := fs.prepare(name)
	if err != nil {
		return nil, err
	}
	return f.File()
}

func (dir _escDirectory) Open(name string) (http.File, error) {
	return dir.fs.Open(dir.name + name)
}

func (f *_escFile) File() (http.File, error) {
	type httpFile struct {
		*bytes.Reader
		*_escFile
	}
	return &httpFile{
		Reader:   bytes.NewReader(f.data),
		_escFile: f,
	}, nil
}

func (f *_escFile) Close() error {
	return nil
}

func (f *_escFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

func (f *_escFile) Stat() (os.FileInfo, error) {
	return f, nil
}

func (f *_escFile) Name() string {
	return f.name
}

func (f *_escFile) Size() int64 {
	return f.size
}

func (f *_escFile) Mode() os.FileMode {
	return 0
}

func (f *_escFile) ModTime() time.Time {
	return time.Unix(f.modtime, 0)
}

func (f *_escFile) IsDir() bool {
	return f.isDir
}

func (f *_escFile) Sys() interface{} {
	return f
}

// FS returns a http.Filesystem for the embedded assets. If useLocal is true,
// the filesystem's contents are instead used.
func FS(useLocal bool) http.FileSystem {
	if useLocal {
		return _escLocal
	}
	return _escStatic
}

// Dir returns a http.Filesystem for the embedded assets on a given prefix dir.
// If useLocal is true, the filesystem's contents are instead used.
func Dir(useLocal bool, name string) http.FileSystem {
	if useLocal {
		return _escDirectory{fs: _escLocal, name: name}
	}
	return _escDirectory{fs: _escStatic, name: name}
}

// FSByte returns the named file from the embedded assets. If useLocal is
// true, the filesystem's contents are instead used.
func FSByte(useLocal bool, name string) ([]byte, error) {
	if useLocal {
		f, err := _escLocal.Open(name)
		if err != nil {
			return nil, err
		}
		return ioutil.ReadAll(f)
	}
	f, err := _escStatic.prepare(name)
	if err != nil {
		return nil, err
	}
	return f.data, nil
}

// FSMustByte is the same as FSByte, but panics if name is not present.
func FSMustByte(useLocal bool, name string) []byte {
	b, err := FSByte(useLocal, name)
	if err != nil {
		panic(err)
	}
	return b
}

// FSString is the string version of FSByte.
func FSString(useLocal bool, name string) (string, error) {
	b, err := FSByte(useLocal, name)
	return string(b), err
}

// FSMustString is the string version of FSMustByte.
func FSMustString(useLocal bool, name string) string {
	return string(FSMustByte(useLocal, name))
}

var _escData = map[string]*_escFile{

	"/resources/source/userdata.sh": {
		local:   "resources/source/userdata.sh",
		size:    1899,
		modtime: 1465486987,
		compressed: `
H4sIAAAJbogA/7RVbW/bNhD+zl9xdfphwybJTlIDNqABTpFuwZAX2OmAoSiMs0hLRCSS4Itjr81/30lW
Uimz96k1DEv3Rj58+Nz55E2ykipZoSsg2gq2uJvN72fL2+vL32fLi6ub2fzv5d3s/o80KXQlkrAKyofk
yxeIF8JuZCZusBLw9BSXWK04xljx8TljJ9/5w06AvvfCecFBK/jY4IDROB6eszo6u76aQotOVpgLlxSb
KnKOt5jpcEpiGTUlUQMzcnQEYaPT4Wg8fEeBEfyElYyG49XkfDwe/8zkGj7BG4jWMOidP1AlR4+xKwbw
mflCKAaQBVtCtHFQeG+mSTIaT+LTd+dx+0xK9HSApjiqqyHScGzZermi0hx+2R7PWUtiGq6U81iWIOgs
O19IlTO3vxtwwdRvTltwXhv4+hXQ+CgXHmRb1UmJduw5GgztIcgDHVdukfd9BxbBR5eVEoL6RxrIpa9X
/RFyuA0W9KMCUi9aAv7owJ1BZuiXqG8EenYRsgdC+fTU2n+KHRl9PtGYUmbopVYxQWZ74K9v5lUWRLwX
Zy939fZYA/0QEhYf7y7nf10tbudkzC8/TJ+l9+1CeKxtnlD4fSlQ6eDBiUwrTqQBl1ZkXhN91QO9Q2Q2
kAifdcoTyl3HnOTYHw3fdl6+v735kA4+GatJINXUGbQeiZscP7NMVxUqnh6lpQG8aPqQqVDRIplLR+wF
WZr4yjBjpbbS79LJZMIweE2ysz71NojGtGLvCEpsDRUKzhoHHZVWG+4NK7yVwqVnTGylzzSn9+Gvp6zu
DCdzhWV6fzm/buxHlJ1ibdDlVgeTrrF0gj3Isux76q5MWy04z4nmZanztSxFmmzQJmQkHWJisl/lLSvc
rnY0INLR9cXr2Aqzh2BaNE0koxYM9nAVDQLl3VIoXJWCtwit2HO6pCRhbevdGy9YuxDJ/wyzk/KfDbux
Psw6cgxmHTsIc8BEVmgYvP0/tQ3gtyM67ZFcu+oBeYc+K2h+Ac3pjuZpH5of0bz925juH/2+rnus1g4V
L172OjReWwXWE/bg8KXYvwEAAP//UofG0msHAAA=
`,
	},

	"/": {
		isDir: true,
		local: "/",
	},

	"/resources": {
		isDir: true,
		local: "/resources",
	},

	"/resources/source": {
		isDir: true,
		local: "/resources/source",
	},
}
