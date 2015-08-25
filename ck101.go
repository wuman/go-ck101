package ck101

import (
	"crypto/md5"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/opesun/goquery"
)

const (
	ErrorImageTooSmall = "image too small"
)

var verbose bool

func init() {
	flag.BoolVar(&verbose, "v", false, "verbose output")
}

type CK101Page struct {
	Title string
	Imgs  []string
}

type CK101Lover struct {
	username string
	pwdhash  string
	client   *http.Client
}

func NewCK101Lover(username, password string) *CK101Lover {
	l := new(CK101Lover)
	if username != "" && password != "" {
		h := md5.New()
		io.WriteString(h, password)
		l.username = username
		l.pwdhash = fmt.Sprintf("%x", h.Sum(nil))
	}
	return l
}

func (l *CK101Lover) authenticate() error {
	if l.client != nil {
		return nil
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}

	l.client = &http.Client{nil, nil, jar, time.Second}

	if l.username != "" && l.pwdhash != "" {
		resp, err := l.client.PostForm("http://ck101.com/member.php?mod=logging&action=login&loginsubmit=yes&infloat=yes&lssubmit=yes&inajax=1",
			url.Values{
				"username":     {l.username},
				"password":     {l.pwdhash},
				"quickforward": {"yes"},
				"handlekey":    {"ls"},
			})
		if err != nil {
			l.username = ""
			l.pwdhash = ""
			return err
		}
		defer resp.Body.Close()
	}

	return nil
}

func (l *CK101Lover) get(url string) (string, error) {
	// authenticate if possible
	l.authenticate()

	resp, err := l.client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (l *CK101Lover) GrabPage(url string) (*CK101Page, error) {
	if url == "" {
		return nil, errors.New("supplied url is empty or invalid")
	}

	page, err := l.get(url)
	if err != nil {
		return nil, fmt.Errorf("could not fetch content, check your network connection. (%v)", err)
	}
	doc, err := goquery.ParseString(page)
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
	imgs := doc.Find("img['file']")
	for _, img := range imgs {
		for _, attr := range img.Attr {
			if attr.Key != "file" {
				continue
			}
			url := attr.Val
			if !strings.HasPrefix(url, "http") {
				continue
			}
			result.Imgs = append(result.Imgs, url)
		}
	}

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
	if page == nil || len(page.Imgs) == 0 {
		return errors.New("No images to fetch")
	}

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
