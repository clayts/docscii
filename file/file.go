package file

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

func Read(filename string) string {
	if filename == "" {
		return ""
	}
	b, _ := ioutil.ReadFile(filename)
	if string(b) != "" {
		fmt.Println("Processing\t", filename)
	}
	return string(b)
}

func Write(filename, contents string) {
	fmt.Println("Creating\t", filename)
	err := os.MkdirAll(filepath.Dir(filename), 0777)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(filename, []byte(contents), 0644)
	if err != nil {
		panic(err)
	}
}

func Copy(src, dst string) error {
	if !Exists(src) {
		panic("file not found --" + src + " " + dst)
	}
	fmt.Println("Copying\t\t", dst)
	err := os.MkdirAll(filepath.Dir(dst), 0777)
	if err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	err = out.Sync()
	return err
}

func Exists(filename string) bool {
	if _, err := os.Stat(filename); err == nil {
		return true
	}
	return false
}

func StripExt(filename string) string {
	ext := filepath.Ext(filename)
	return filename[:len(filename)-len(ext)]
}
