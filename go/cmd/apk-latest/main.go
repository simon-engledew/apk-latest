package main // import "github.com/simon-engledew/apk-latest/go/cmd/apk-latest"

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

func packages(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if i := bytes.Index(data, []byte{'\n', '\n'}); i >= 0 {
		return i + 2, data[0 : i-2], nil
	}

	if atEOF {
		return len(data), data, nil
	}

	return 0, nil, nil
}

// see https://wiki.alpinelinux.org/wiki/Apk_spec
type apk struct {
	Architecture         string `apk:"A"`
	PullChecksum         string `apk:"C"`
	PullDependencies     string `apk:"D"`
	PackageInstalledSize string `apk:"I"`
	Licence              string `apk:"L"`
	Name                 string `apk:"P"`
	Version              string `apk:"V"`
	Size                 int    `apk:"S"`
	Description          string `apk:"T"`
	URL                  string `apk:"U"`
	GitCommit            string `apk:"c"`
	Maintainer           string `apk:"m"`
	Origin               string `apk:"o"`
	ProviderPriority     string `apk:"k"`
	Provides             string `apk:"p"`
	BuildTimestamp       int    `apk:"t"`
}

var packageSetter = (func() func(apk *apk, key string, value string) error {
	t := reflect.TypeOf(apk{})

	mapping := make(map[string]int, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		mapping[t.Field(i).Tag.Get("apk")] = i
	}

	return func(apk *apk, key string, value string) error {
		if i, ok := mapping[key]; ok {
			f := reflect.ValueOf(apk).Elem().Field(i)

			if f.IsValid() && f.CanSet() {
				switch f.Kind() {
				case reflect.Int:
					n, err := strconv.ParseInt(value, 10, 64)
					if err != nil {
						return err
					}
					if f.OverflowInt(n) {
						return fmt.Errorf("int %d would overflow field", n)
					}
					f.SetInt(n)
				default:
					f.SetString(value)
				}
			}

			return nil
		}

		return fmt.Errorf("unknown key: %s", key)
	}
})()

var (
	indexes = kingpin.Flag("repository", "Use packages from REPO").Short('X').Default("http://dl-cdn.alpinelinux.org/alpine/v3.10/main/x86_64/APKINDEX.tar.gz", "http://dl-cdn.alpinelinux.org/alpine/v3.10/community/x86_64/APKINDEX.tar.gz").URLList()
	args    = kingpin.Arg("PACKAGE", "Print the latest version of PACKAGEs").Required().Strings()
)

func scanPackage(reader io.Reader, fn func(apk *apk) error) error {
	scanner := bufio.NewScanner(reader)
	scanner.Split(packages)

	for scanner.Scan() {
		buffer := scanner.Bytes()

		i := 0

		current := new(apk)

		for i < len(buffer) {
			key := string(buffer[i])
			i += 1
			if buffer[i] != ':' {
				return errors.New("parse error")
			}
			i += 1
			end := bytes.IndexByte(buffer[i:], '\n')
			if end == -1 {
				break
			}
			value := string(buffer[i : end+i])
			i += end + 1

			if err := packageSetter(current, key, value); err != nil {
				return err
			}
		}
		if err := fn(current); err != nil {
			return err
		}
	}

	return nil
}

func scanIndex(r io.Reader, fn func(apk *apk) error) error {
	gzipReader, err := gzip.NewReader(r)
	if err != nil {
		return err
	}

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if header.Name == "APKINDEX" {
			if err := scanPackage(tarReader, fn); err != nil {
				return err
			}
		}
	}

	return nil
}

func main() {
	kingpin.Parse()

	find := make(map[string]*apk, len(*args))

	for _, arg := range *args {
		find[arg] = nil
	}

	for _, target := range *indexes {
		resp, err := http.Get(target.String())
		if err != nil {
			panic(err)
		}

		if err := scanIndex(resp.Body, func(apk *apk) error {
			// TO-DO: implement version check and select newest
			if _, ok := find[apk.Name]; ok {
				find[apk.Name] = apk
			}

			return nil
		}); err != nil {
			panic(err)
		}
	}

	found := make([]string, len(*args))

	var missing []string

	for n, name := range *args {
		apk := find[name]

		if apk == nil {
			missing = append(missing, name)
			continue
		}

		found[n] = fmt.Sprintf("%s==%s", apk.Name, apk.Version)
	}

	if len(missing) > 0 {
		panic(fmt.Errorf("missing packages: %s", strings.Join(missing, ", ")))
	}

	fmt.Println(strings.Join(found, " "))
}
