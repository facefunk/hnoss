package hnoss

import (
	"bytes"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

type (
	TextFileTimeAdapter struct {
		file string
	}
	TextFileIPAdapter struct {
		file string
	}
	PlainTextIPServiceAdapter struct {
		url string
	}
	RealNowAdapter struct{}
)

func NewTextFileTimeAdapter(file string) *TextFileTimeAdapter {
	return &TextFileTimeAdapter{
		file: file,
	}
}

func (m *TextFileTimeAdapter) Get() (t time.Time, err error) {
	file, closeFile := openFile(m.file, "time", &err)
	if err != nil {
		err = (*Warn)(err.(*Error))
		return
	}
	defer closeFile()
	b := make([]byte, 25)
	if _, err = file.Read(b); err != nil {
		err = ErrorWrap(err, "failed to read from time file")
		return
	}
	s := trim(b)
	t, err = time.Parse(time.RFC3339, s)
	if err != nil {
		err = ErrorWrapf(err, "failed to parse date from time file: %s", s)
	}
	return
}

func (m *TextFileTimeAdapter) Put(t time.Time) (err error) {
	file, closeFile := createFile(m.file, "time", &err)
	if err != nil {
		return
	}
	b := t.Format(time.RFC3339)
	_, err = file.WriteString(b)
	if err != nil {
		err = ErrorWrap(err, "failed to write to time file")
	}
	closeFile()
	return
}

func NewTextFileIPAdapter(file string) *TextFileIPAdapter {
	return &TextFileIPAdapter{
		file: file,
	}
}

func (m *TextFileIPAdapter) Get() (ip netip.Addr, err error) {
	file, closeFile := openFile(m.file, "IP", &err)
	if err != nil {
		err = (*Warn)(err.(*Error))
		return
	}
	defer closeFile()
	return readIP(file)
}

func (m *TextFileIPAdapter) Put(ip netip.Addr) (err error) {
	file, closeFile := createFile(m.file, "time", &err)
	if err != nil {
		return
	}
	_, err = file.WriteString(ip.String())
	if err != nil {
		err = ErrorWrap(err, "failed to write to IP file")
	}
	closeFile()
	return
}

func NewPlainTextIPServiceAdapter(url string) *PlainTextIPServiceAdapter {
	return &PlainTextIPServiceAdapter{
		url: url,
	}
}

func (m *PlainTextIPServiceAdapter) Get() (ip netip.Addr, err error) {
	file, closeFile := fetch(m.url, "IP", &err)
	if err != nil {
		return
	}
	defer closeFile()
	return readIP(file)
}

func readIP(file io.Reader) (ip netip.Addr, err error) {
	b := make([]byte, 39)
	var n int
	if n, err = file.Read(b); err != nil {
		if err != io.EOF || n == 0 {
			err = ErrorWrap(err, "failed to read from IP file")
			return
		}
	}
	s := trim(b)
	ip, err = netip.ParseAddr(s)
	if err != nil {
		err = ErrorWrapf(err, "failed to parse IP address: %s", s)
	}
	return
}

func NewRealNowAdapter() *RealNowAdapter {
	return &RealNowAdapter{}
}

func (m *RealNowAdapter) Now() time.Time {
	return time.Now().UTC()
}

// trim spaces and null characters from byte slice and return string.
func trim(b []byte) string {
	return strings.TrimSpace(string(bytes.Trim(b, "\x00")))
}

// fetch first tries to interpret path as a URL, then as a file path.
func fetch(path, desc string, err *error) (io.ReadCloser, func()) {
	var uri *url.URL
	uri, *err = url.ParseRequestURI(path)
	if *err != nil {
		return openFile(path, desc, err)
	}
	if uri.Scheme == "file" {
		return openFile(uri.Host+uri.Path, desc, err)
	}
	u := uri.String()
	var res *http.Response
	res, *err = http.Get(u)
	if *err != nil {
		*err = ErrorWrapf(*err, "failed to download from %s", u)
		return nil, nil
	}
	file := res.Body
	return file, func() {
		if *err = file.Close(); *err != nil {
			*err = ErrorWrapf(*err, "failed to close %s URL: %s", desc, path)
		}
	}
}
