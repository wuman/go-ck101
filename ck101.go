package ck101

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

const (
	ErrorImageTooSmall = "image too small"
)

var verbose bool

func init() {
	flag.BoolVar(&verbose, "ck101.verbose", false, "verbose output")
}

type CK101Page struct {
	Title string
	Imgs  []string
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
	result.Title = doc.Find("title").Text()
	result.Title = strings.Split(result.Title, " - ")[0]
	result.Title = strings.Replace(result.Title, "/", "", -1)
	result.Title = strings.TrimSpace(result.Title)

	// find the images
	doc.Find("img[file]").Each(func(i int, s *goquery.Selection) {
		url, _ := s.Attr("file")
		if !strings.HasPrefix(url, "http") {
			return
		}
		result.Imgs = append(result.Imgs, url)
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
	for _, img := range page.Imgs {
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
