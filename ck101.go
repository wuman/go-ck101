package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

const (
	ErrorImageTooSmall = "image too small"
)

var verbose bool
var url string

func init() {
	flag.BoolVar(&verbose, "ck101.verbose", false, "verbose output")
	flag.StringVar(&url, "ck101.url", "", "url to grab images from. should have pattern http://ck101.com/thread-2593278-1-1.html")
}

type CK101Page struct {
	title string
	imgs  []string
}

func GrabPage(url string) (*CK101Page, error) {
	if url == "" {
		return nil, errors.New("supplied url is empty or invalid")
	}

	doc, err := goquery.NewDocument(url)
	if err != nil {
		return nil, fmt.Errorf("could not fetch content, check your network connection. (%v)", err)
	}

	result := &CK101Page{}

	// find the title
	result.title = doc.Find("title").Text()
	result.title = strings.Split(result.title, " - ")[0]
	result.title = strings.Replace(result.title, "/", "", -1)
	result.title = strings.TrimSpace(result.title)

	// find the images
	doc.Find("img[file]").Each(func(i int, s *goquery.Selection) {
		url, _ := s.Attr("file")
		if !strings.HasPrefix(url, "http") {
			return
		}
		result.imgs = append(result.imgs, url)
	})

	return result, nil
}

func mkdir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, os.ModePerm)
		return err
	}
	return nil
}

func GrabImages(page *CK101Page, targetDir string) error {
	if err := mkdir(targetDir); err != nil {
		log.Fatalf("Unable to create directory: %s\n", targetDir)
		return err
	}
	if verbose {
		log.Printf("Target saving directory: %s\n", targetDir)
	}

	var wg sync.WaitGroup
	for _, img := range page.imgs {
		wg.Add(1)
		go func(img string) {
			defer wg.Done()
			err := grabImage(img, filepath.Join(targetDir, path.Base(img)))
			if verbose {
				if err == nil {
					log.Printf("%s [ok]\n", img)
				} else if err.Error() == ErrorImageTooSmall {
					log.Printf("%s [skip]: %v\n", img, err)
				} else {
					log.Printf("%s [fail]: %v\n", img, err)
				}
			}
		}(img)
	}
	wg.Wait()
	return nil
}

func grabImage(url, savepath string) error {
	resp, err := http.Get(url)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	r := resp.Body

	m, _, err := image.Decode(r)
	if err != nil {
		return err
	}
	bounds := m.Bounds()
	if bounds.Max.X < 400 || bounds.Max.Y < 400 {
		return errors.New(ErrorImageTooSmall)
	}

	w, err := os.Create(savepath)
	defer w.Close()

	err = jpeg.Encode(w, m, &jpeg.Options{jpeg.DefaultQuality})

	return err
}

func main() {
	flag.Parse()

	if url == "" || !strings.HasPrefix(url, "http") {
		log.Fatalf("URL should be in the form of http://ck101.com/thread-2593278-1-1.html")
	}

	u, err := user.Current()
	if err != nil {
		log.Fatalf("Failed to get current user: %v", err)
	}
	basedir := filepath.Join(u.HomeDir, "Pictures/go-ck101/")
	if err := mkdir(basedir); err != nil {
		log.Fatalf("Unable to create directory: %s\n", basedir)
	}
	threadId := regexp.MustCompile("thread-(\\d+)-.*").FindStringSubmatch(path.Base(url))[1]

	b, err := GrabPage(url)
	if err != nil {
		log.Fatalf("Failed to grab the page: %v", err)
	}
	if verbose {
		log.Printf("title: %s\n", b.title)
	}

	targetDir := filepath.Join(basedir, fmt.Sprintf("%s - %s", threadId, b.title))
	GrabImages(b, targetDir)
}
