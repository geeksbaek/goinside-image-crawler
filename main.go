package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	vision "cloud.google.com/go/vision/apiv1"
	"github.com/geeksbaek/goinside"
	"github.com/sirupsen/logrus"
	pb "google.golang.org/genproto/googleapis/cloud/vision/v1"
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

	errFileSizeTooSmall    = errors.New("File size is too small")
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
	URL, gallID := mustGetID(*flagURL, *flagGallID)

	imageSubdirectory = fmt.Sprintf(`%s/%s`, defaultImageDirectory, gallID)
	mkdir(imageSubdirectory + "/ADULT_0_UNKNOWN")
	mkdir(imageSubdirectory + "/ADULT_1_VERY_UNLIKELY")
	mkdir(imageSubdirectory + "/ADULT_2_UNLIKELY")
	mkdir(imageSubdirectory + "/ADULT_3_POSSIBLE")
	mkdir(imageSubdirectory + "/ADULT_4_LIKELY")
	mkdir(imageSubdirectory + "/ADULT_5_VERY_LIKELY")
	hashingExistImages(imageSubdirectory)

	logrus.Infof("target is %s, crawl start.", gallID)
	// get first list of *flagGall every tick.
	// and iterate all articles.
	ticker := time.Tick(duration)
	for range ticker {
		logrus.Infof("Fetching First Page of %v...", gallID)
		if list, err := goinside.FetchList(gallID, 1); err != nil {
			logrus.Errorf("%v: %v", URL, err)
		} else {
			go iterate(list.Items)
		}
	}
}

func detectSafeSearch(f []byte) (pb.Likelihood, error) {
	ctx := context.Background()

	client, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return 0, err
	}

	image, err := vision.NewImageFromReader(bytes.NewReader(f))
	if err != nil {
		return 0, err
	}
	props, err := client.DetectSafeSearch(ctx, image, nil)
	if err != nil {
		return 0, err
	}
	return props.Adult, nil
}

func hashingExistImages(path string) {
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
	// if not, passing the images to process()
	imageCount := len(imageURLs)
	successAll := true
	wg := new(sync.WaitGroup)
	wg.Add(len(imageURLs))
	for i, imageURL := range imageURLs {
		i, imageURL := i, imageURL
		go func() {
			defer wg.Done()
			err := process(imageURL)
			switch err {
			case errDuplicateImage:
				logrus.Infof("%v (%v/%v) Dup.", item.Subject, i+1, imageCount)
			case nil:
				logrus.Infof("%v (%v/%v) OK.", item.Subject, i+1, imageCount)
			default:
				logrus.Infof("%v (%v/%v) Failed. %v", item.Subject, i+1, imageCount, err)
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
func process(URL goinside.ImageURLType) (err error) {
	image, filename, err := URL.Fetch()
	if err != nil {
		return
	}
	hash := hashingBytes(image)
	if history.image.get(hash) == true {
		err = errDuplicateImage
		return
	}
	defer history.image.set(hash, true)

	_, extension := splitPath(filename)
	filename = strings.Join([]string{hash, extension}, ".")

	if len(image) < 40000 {
		return errFileSizeTooSmall
	}

	adultLikelihood, err := detectSafeSearch(image)
	if err != nil {
		return
	}

	path := fmt.Sprintf(`%s/ADULT_%d_%s/%s`,
		imageSubdirectory, adultLikelihood, adultLikelihood.String(), filename)
	if err = saveImage(image, path); err != nil {
		return
	}
	return
}
