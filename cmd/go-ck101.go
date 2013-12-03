package main

import (
	"flag"
	"fmt"
	"log"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	ck101 "github.com/wuman/go-ck101"
)

var url string

func init() {
	flag.StringVar(&url, "ck101.url", "", "url to grab images from. should have pattern http://ck101.com/thread-2593278-1-1.html")
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
	threadId := regexp.MustCompile("thread-(\\d+)-.*").FindStringSubmatch(path.Base(url))[1]

	b, err := ck101.GrabPage(url)
	if err != nil {
		log.Fatalf("Failed to grab the page: %v", err)
	}

	targetDir := filepath.Join(basedir, fmt.Sprintf("%s - %s", threadId, b.Title))
	ck101.GrabImages(b, targetDir)
}
