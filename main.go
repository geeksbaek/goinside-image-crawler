package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/geeksbaek/goinside"
)

type mutexMap struct {
	storage map[string]bool
	mutex   *sync.RWMutex
}

func (m *mutexMap) set(key string, value bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.storage[key] = value
}

func (m *mutexMap) get(key string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.storage[key]
}

var (
	flagGall              = flag.String("gall", "", "http://m.dcinside.com/list.php?id=programming")
	defaultImageDirectory = "./images"
	imageSubdirectory     = ""
	duration              = time.Second * 5

	history = struct {
		article *mutexMap
		image   *mutexMap
	}{
		article: &mutexMap{map[string]bool{}, new(sync.RWMutex)},
		image:   &mutexMap{map[string]bool{}, new(sync.RWMutex)},
	}

	errDuplicateImage      = errors.New("duplicated image")
	errCannotFoundID       = errors.New("cannot found id from url")
	errCannotFoundNo       = errors.New("cannot found no from url")
	errCannotFoundFilename = errors.New("cannot found filename from content-position")

	idRe = regexp.MustCompile(`id=([^&]*)`)
	noRe = regexp.MustCompile(`no=([^&]*)`)
)

func main() {
	flag.Parse()

	id := getID(*flagGall)
	imageSubdirectory = fmt.Sprintf(`%s/%s`, defaultImageDirectory, id)
	mkdir(imageSubdirectory)
	hashingExistImages(imageSubdirectory)

	log.Printf("target is %s, crawl start.\n", *flagGall)
	// get first list of *flagGall every tick.
	// and iterate all articles.
	ticker := time.Tick(duration)
	for _ = range ticker {
		log.Println("Fetching First Page...")
		if list, err := goinside.FetchList(*flagGall, 1); err == nil {
			go iterate(list.Articles)
		}
	}
}

func getID(URL string) (id string) {
	matched := idRe.FindStringSubmatch(URL)
	if len(matched) != 2 {
		panic(errCannotFoundID)
	}
	id = matched[1]
	return
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

// init will crate defalut image directory, and
// find existing images, and hashing, and add to history.
func hashingExistImages(path string) {
	fileRenameToHash := func(path, extension string) (err error) {
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
	forEachImages := func(path string, f os.FileInfo, _ error) (err error) {
		if f.IsDir() {
			return
		}
		filename, extension := splitPath(f.Name())
		// check filename is not hash.
		// if not, hashing and rename.
		if len(filename) != 40 {
			fileRenameToHash(path, extension)
		}
		history.image.set(filename, true)
		return
	}
	filepath.Walk(path, forEachImages)
}

// if find an image included article, fetching it.
func iterate(articles []*goinside.Article) {
	for _, article := range articles {
		if article.HasImage {
			go fetchArticle(article)
		}
	}
}

func fetchArticle(article *goinside.Article) {
	article, err := goinside.FetchArticle(article.URL)
	if err != nil {
		return
	}
	// if you already seen this article, return.
	if history.article.get(article.Number) == true {
		return
	}
	log.Printf("#%v article has an image. process start.\n", article.Number)
	// if not, passing the images to process()
	imageCnt := len(article.Detail.ImageURLs)
	for i, imageURL := range article.Detail.ImageURLs {
		if err := process(imageURL); err == errDuplicateImage {
			log.Printf("#%v (%v/%v) duplicate image.\n",
				article.Number, i+1, imageCnt)
		} else if err != nil {
			log.Printf("#%v (%v/%v) process failed. %v\n",
				article.Number, i+1, imageCnt, err)
			return
		} else {
			log.Printf("#%v (%v/%v) image has been saved successfully.\n",
				article.Number, i+1, imageCnt)
		}
	}
	history.article.set(article.Number, true)
}

// process will fetching the image, and hashing,
// and comparing the history with it.
// if it already exists, return errDuplicateImage.
// if not, save it, and add to the history.
func process(URL string) (err error) {
	resp, err := fetchImage(URL)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	hash := hashingBytes(body)
	if history.image.get(hash) == true {
		err = errDuplicateImage
		return
	}
	filename, err := getFilename(resp)
	if err != nil {
		return
	}
	_, extension := splitPath(filename)
	filename = strings.Join([]string{hash, extension}, ".")
	path := fmt.Sprintf(`%s/%s`, imageSubdirectory, filename)
	err = saveImage(body, path)
	if err != nil {
		return
	}
	history.image.set(hash, true)
	return
}

func fetchImage(URL string) (resp *http.Response, err error) {
	matchedID := idRe.FindStringSubmatch(URL)
	if len(matchedID) != 2 {
		err = errCannotFoundID
		return
	}
	matchedNO := noRe.FindStringSubmatch(URL)
	if len(matchedNO) != 2 {
		err = errCannotFoundNo
		return
	}
	// strangely, dcinside requires these forms to request images.
	form := func(m map[string]string) (reader io.Reader) {
		data := url.Values{}
		for k, v := range m {
			data.Set(k, v)
		}
		reader = strings.NewReader(data.Encode())
		return
	}(map[string]string{
		"id": matchedID[1],
		"no": matchedNO[1],
	})

	req, err := http.NewRequest("GET", URL, form)
	if err != nil {
		return
	}
	client := &http.Client{}
	resp, err = client.Do(req)
	return
}

func hashingBytes(data []byte) (hash string) {
	hasher := sha1.New()
	hasher.Write(data)
	hash = hex.EncodeToString(hasher.Sum(nil))
	return
}

func hashingFile(path string) (hash string, er error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}
	hash = hashingBytes(data)
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

func splitPath(fullname string) (filename, extension string) {
	splitedName := strings.Split(fullname, ".")
	filename = strings.Join(splitedName[:len(splitedName)-1], ".")
	extension = splitedName[len(splitedName)-1]
	return
}
