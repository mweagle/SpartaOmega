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
		size:    1931,
		modtime: 1465484544,
		compressed: `
H4sIAAAJbogA/7RV227bRhO+36eYyLn4f7QkJdsRIAEsIAdOaxQ+QHIKFEEgjLgrcmFyd7EHWWrid++Q
oh3SldKbRhBEznG/nflmdPImWUmVrNAVEG0FW9zN5vez5e315a+z5cXVzWz+5/Judv9bmhS6EklYBeVD
8uULxAthNzITN1gJeHqKS6xWHGOs+PicsZP/+MNOgL73wnnBQSv42OCA0TgenrPaOru+mkKLTlaYC5cU
mypyjreY6XJKYhk1IVEDM3J0BWGj0+FoPHxHhhH8DysZDceryfl4PP4/k2v4BG8gWsOgd/9AkRw9xq4Y
wGfmC6EYQBZsCdHGQeG9mSbJaDyJT9+dx+0zKdHTBZrgqI6GSMOxtHW6otIcftp+zweNj3LhQSrnsSzB
BVO3xWkL0Y6tJXUCrlqboLvufCFVzty+d11357WBr1//JeOzNRjCIEgDHVVukfd1B5Lgo8tKCUH9JQ3k
0tdZfwRdboMF/aiA2I2WgD86cGeQGfql1jQEPrsI2QOhfHpq5d/FjoR+vdGYUmbopVYxQWZ74K8798oL
It6zs5devj02YD+kCIuPd5fzP64Wt3MS5pcfps/U/NYQHmubJ2R+XwpUOnhwItOKU9GASysyr6l81QO9
Q2Q2kAifdcIT8l3HnKjYXx3fTl6+v735kA4+GauJINXUGbQeqTY5fmaZripUPD1algbwoplTpkJFSTKX
jtgLsjTxlWHGSm2l36WTyYRh8JpoZ33qbRCNaMVeEZTYGgoUnDUKuiplG+4FK7yVwqVnTGylzzSn9+HP
p6yeDCdzhWV6fzm/buRHlJ1gbdDlVgeTrrF0gj3Isuxr6qlNWy44z6nMy1Lna1mKNNmgTUhIOoWJSX7l
t6xwu9rRAklH1xevbSvMHoJp0TSWjEYw2MNRtAiUd0uhcFUK3iK0Yl/TJTkJa1vtXnjB2oVI+meYHZd/
HNi19WHWlmMwa9tBmAMmskLD4O332DaAX47wtFfkWlUvyDv0WUH7C2iPdzhP59D+iObt38p0/+jPdT1j
NXcoePFy1qH12jKw3rAHly/Z/g4AAP//8HiiQYsHAAA=
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
