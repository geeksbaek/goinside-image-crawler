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

// flags
var (
	flagURL    = flag.String("url", "", "http://m.dcinside.com/list.php?id=programming")
	flagGallID = flag.String("gall", "", "programming")
)

var (
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
	errInvalidArgs         = errors.New("invalid args")
	errCannotFoundID       = errors.New("cannot found id from url")
	errCannotFoundNo       = errors.New("cannot found no from url")
	errCannotFoundFilename = errors.New("cannot found filename from content-position")

	idRe = regexp.MustCompile(`id=([^&]*)`)
	noRe = regexp.MustCompile(`no=([^&]*)`)
)

func main() {
	flag.Parse()
	URL, gallID := getID(*flagURL, *flagGallID)

	imageSubdirectory = fmt.Sprintf(`%s/%s`, defaultImageDirectory, gallID)
	mkdir(imageSubdirectory)
	hashingExistImages(imageSubdirectory)

	log.Printf("target is %s, crawl start.\n", gallID)
	// get first list of *flagGall every tick.
	// and iterate all articles.
	ticker := time.Tick(duration)
	for _ = range ticker {
		log.Printf("Fetching First Page of %v...\n", gallID)
		if list, err := goinside.FetchList(URL, 1); err == nil {
			go iterate(list.Items)
		}
	}
}

func getID(URL, gallID string) (retURL, retGallID string) {
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
func iterate(articles []*goinside.ListItem) {
	for _, article := range articles {
		if article.HasImage {
			go fetchArticle(article)
		}
	}
}

func fetchArticle(item *goinside.ListItem) {
	imageURLs, err := item.FetchImageURLs()
	if err != nil {
		return
	}
	// if you already seen this article, return.
	if history.article.get(item.Number) == true {
		return
	}
	log.Printf("#%v article has an image. process start.\n", item.Number)
	// if not, passing the images to process()
	imageCount := len(imageURLs)
	successAll := true
	wg := new(sync.WaitGroup)
	wg.Add(len(imageURLs))
	for i, imageURL := range imageURLs {
		i, imageURL := i, imageURL
		go func() {
			defer wg.Done()
			switch process(imageURL) {
			case errDuplicateImage:
				log.Printf("#%v (%v/%v) duplicate image.\n", item.Number, i+1, imageCount)
			case nil:
				log.Printf("#%v (%v/%v) image has been saved successfully.\n", item.Number, i+1, imageCount)
			default:
				log.Printf("#%v (%v/%v) process failed. %v\n", item.Number, i+1, imageCount, err)
				successAll = false
			}
		}()

	}
	wg.Wait()
	if successAll {
		history.article.set(item.Number, true)
	}
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
	form := formMaker(map[string]string{
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

func formMaker(m map[string]string) (reader io.Reader) {
	data := url.Values{}
	for k, v := range m {
		data.Set(k, v)
	}
	reader = strings.NewReader(data.Encode())
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
