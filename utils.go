package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
)

func mustGetID(URL, gallID string) (retURL, retGallID string) {
	switch {
	case URL != "" && gallID == "":
		matched := idRe.FindStringSubmatch(URL)
		if len(matched) == 2 {
			retURL = URL
			retGallID = matched[1]
			return
		}
	case URL == "" && gallID != "":
		retURL = fmt.Sprintf("http://m.dcinside.com/list.php?id=%v", gallID)
		retGallID = gallID
		return
	}
	panic(errInvalidArgs)
}

func mkdir(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, 0700)
		if err != nil {
			panic(err)
		}
		return
	}
}

func splitPath(fullname string) (filename, extension string) {
	splitedName := strings.Split(fullname, ".")
	filename = strings.Join(splitedName[:len(splitedName)-1], ".")
	extension = splitedName[len(splitedName)-1]
	return
}

func hashingBytes(data []byte) (hash string) {
	hasher := sha1.New()
	hasher.Write(data)
	hash = hex.EncodeToString(hasher.Sum(nil))
	return
}

func hashingFile(path string) (hash string, err error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	hash = hashingBytes(data)
	return
}

func fileRenameToHash(path, extension string) (err error) {
	newpath, err := hashingFile(path)
	if err != nil {
		return
	}
	newpath = fmt.Sprintf(`%s/%s`, path, newpath)
	newfilename := strings.Join([]string{newpath, extension}, ".")
	err = os.Rename(path, newfilename)
	if err != nil {
		return
	}
	return
}

func getFilename(resp *http.Response) (filename string, err error) {
	filenameRe := regexp.MustCompile(`filename=(.*)`)
	contentDisposition := resp.Header.Get("Content-Disposition")
	matched := filenameRe.FindStringSubmatch(contentDisposition)
	if len(matched) != 2 {
		err = errCannotFoundFilename
		return
	}
	filename = strings.ToLower(matched[1])
	return
}

func saveImage(data []byte, path string) (err error) {
	file, err := os.Create(path)
	if err != nil {
		return
	}
	_, err = io.Copy(file, bytes.NewReader(data))
	if err != nil {
		return
	}
	file.Close()
	return
}
