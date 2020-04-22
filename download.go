package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

var goproxy string

type ModVersion struct {
	Version string
	Time    time.Time
}

func getLatestVer(modpath string) (*ModVersion, error) {
	url1 := fmt.Sprintf("%s/%s/@latest", goproxy, modpath)
	resp, err := http.Get(url1)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	res := new(ModVersion)
	buf := bytes.NewBufferString("")
	for {
		n, err := buf.ReadFrom(resp.Body)
		if n <= 0 || err != nil {
			break
		}
	}
	err = json.Unmarshal(buf.Bytes(), res)
	return res, err
}

func downloadZipMod(modpath, ver string) (string, error) {
	url1 := fmt.Sprintf("%s/%s/@v/%s.zip", goproxy, modpath, ver)
	resp, err := http.Get(url1)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	fname := path.Base(modpath) + ".zip"
	fp, err := os.Create(fname)
	if err != nil {
		return "", err
	}
	defer fp.Close()
	for {
		n, err := io.Copy(fp, resp.Body)
		if n <= 0 || err != nil {
			break
		}
	}
	return fname, nil
}

func main() {
	var myproxy = flag.String("goproxy", "https://goproxy.io", "set GOPROXY url")
	flag.Parse()
	goproxy = *myproxy
	var gopath = os.Getenv("GOPATH")
	mod := flag.Arg(0)
	if mod == "" {
		fmt.Println("usage: gomget <module path>")
	}
	ver, err := getLatestVer(mod)
	if err != nil {
		log.Fatalf("getLatestVer:%s\n", err.Error())
	}
	fmt.Printf("Version:%s\nTime:%s\n", ver.Version, ver.Time.Format("2006-01-02 15:04:05"))

	fname, err := downloadZipMod(mod, ver.Version)
	fmt.Printf("Saved: %s\nUnzipping...\n", fname)
	rd, err := zip.OpenReader(fname)
	if err != nil {
		panic(err)
	}
	pkgPath, err := UnzipAll(rd, gopath, ver.Version)
	if err != nil {
		panic(err)
	}
	fmt.Printf("PackagePath: %s\n", pkgPath)
	nName := strings.TrimSuffix(pkgPath, "@"+ver.Version)
	os.RemoveAll(nName)
	err = os.Rename(pkgPath, nName)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Rename: %s -> %s\n", pkgPath, nName)
}
