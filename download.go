package main

import (
	"archive/zip"
	"bytes"
	"container/list"
	"encoding/json"
	"flag"
	"fmt"
	"go/build"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"runtime"
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

func downloadPath(gopath, modpath string) error {
	mod := modpath
	var ver *ModVersion
	var err error
	for {
		ver, err = getLatestVer(mod)
		if err != nil {
			fmt.Printf("NotFound:%s (%s)\n", mod, err.Error())
			mod = path.Dir(mod)
			if strings.Contains(mod, "/") == false {
				return fmt.Errorf("getLatestVer:%s\n", err.Error())
			}
		} else {
			fmt.Printf("Found: %s\n", mod)
			break
		}
	}
	fmt.Printf("Version:%s\nTime:%s\n", ver.Version, ver.Time.Format("2006-01-02 15:04:05"))

	fname, err := downloadZipMod(mod, ver.Version)
	fmt.Printf("Saved: %s\nUnzipping...\n", fname)
	rd, err := zip.OpenReader(fname)
	if err != nil {
		return err
	}
	pkgPath, err := UnzipAll(rd, gopath, ver.Version)
	if err != nil {
		return err
	}
	fmt.Printf("PackagePath: %s\n", pkgPath)
	nName := strings.TrimSuffix(pkgPath, "@"+ver.Version)
	os.RemoveAll(nName)
	err = os.Rename(pkgPath, nName)
	if err != nil {
		return err
	}
	fmt.Printf("Rename: %s -> %s\n", pkgPath, nName)
	return nil
}

func tryGetPackage(goroot, gopath, modpath string) (p *build.Package, err error) {
	p, err = build.Import(modpath, path.Join(goroot, "src"), 0)
	if err != nil {
		p, err = build.Import(modpath, path.Join(gopath, "src"), 0)
	}
	return
}

func arrayToList(array []string) *list.List {
	res := list.New()
	for _, v := range array {
		if strings.Contains(v, ".") == false {
			continue
		}
		res.PushBack(v)
	}
	return res
}

func downPathWithDeps(goroot, gopath, modpath string) error {
	err := downloadPath(gopath, modpath)
	if err != nil {
		return err
	}
	p, err := tryGetPackage(goroot, gopath, modpath)
	if err != nil {
		return err
	}
	record := make(map[string]bool)
	imports := arrayToList(p.Imports)
	//list
	for {
		if imports.Len() == 0 {
			break
		}
		v1 := imports.Front()
		p1 := v1.Value.(string)
		imports.Remove(v1)

		if strings.Contains(p1, ".") == false {
			continue
		}

		if _, ok := record[p1]; ok {
			continue
		}
		record[p1] = true

		p, err = tryGetPackage(goroot, gopath, p1)
		if err != nil {
			fmt.Printf("Download:%s\n", p1)
			err = downloadPath(gopath, p1)
			if err != nil {
				return err
			}
			p, err = tryGetPackage(goroot, gopath, p1)
			if err != nil {
				return err
			}
		}
		for _, v := range p.Imports {
			if strings.Contains(v, ".") == false {
				continue
			}
			if _, ok := record[v]; ok {
				continue
			}
			imports.PushBack(v)
		}
	}
	return nil
}

func main() {
	var myproxy = flag.String("goproxy", "https://goproxy.io", "set GOPROXY url")
	flag.Parse()
	goproxy = *myproxy
	var gopath string
	var err error
	gopath, err = getGOPATH()
	if err != nil {
		log.Fatalf("getGOPATH:%s\n", err.Error())
	}
	fmt.Printf("GOPATH: %s\n", gopath)
	mod := flag.Arg(0)
	if mod == "" {
		fmt.Println("usage: gomget <module path>")
	}

	goroot := runtime.GOROOT()

	err = downPathWithDeps(goroot, gopath, mod)
	fmt.Println(err)
}
