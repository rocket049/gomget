package main

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
)

var decoder = simplifiedchinese.GB18030.NewDecoder()

func getName(v *zip.File) (name string, err error) {
	if v.NonUTF8 {
		tmp := make([]byte, len([]byte(v.Name)))
		_, _, err = encoding.UTF8Validator.Transform(tmp, []byte(v.Name), true)
		if err == nil {
			name = v.Name
		} else {
			name, err = decoder.String(v.Name)
			if err != nil {
				return
			}
		}
	} else {
		name = v.Name
	}
	return
}

func UnzipAll(fp *zip.ReadCloser, gopath, ver string) (vdir string, err error) {
	rname := filepath.Join(gopath, "src")

	os.MkdirAll(rname, os.ModePerm)
	os.Chdir(rname)
	var name string
	for _, v := range fp.File {
		name, err = getName(v)
		if err != nil {
			return
		}
		if v.FileInfo().IsDir() {
			//dir
			os.MkdirAll(name, v.FileInfo().Mode())
			if strings.HasSuffix(name, ver) {
				vdir = filepath.Join(rname, name)
			}
			continue
		}
		dir1 := filepath.Dir(name)
		//dir
		os.MkdirAll(dir1, os.ModePerm)
		if strings.HasSuffix(dir1, ver) {
			vdir = filepath.Join(rname, dir1)
		}
		//fmt.Println(i, v.Mode(), name, v.UncompressedSize64)
	}
	var i int = 1
	for _, v := range fp.File {
		if v.FileInfo().IsDir() {
			continue
		}
		name, err = getName(v)
		if err != nil {
			return
		}
		var fpr io.ReadCloser
		fpr, err = v.Open()
		if err != nil {
			return
		}
		var fpw io.WriteCloser
		fpw, err = os.Create(name)
		if err != nil {
			fpr.Close()
			return
		}
		io.Copy(fpw, fpr)
		fpw.Close()
		fpr.Close()
		os.Chtimes(name, v.Modified, v.Modified)
		os.Chmod(name, v.Mode())
		//fmt.Println(i, v.Mode(), name, v.UncompressedSize64)
		i++
	}
	return
}
