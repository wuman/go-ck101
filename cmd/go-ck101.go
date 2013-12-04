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
var username string
var password string

func init() {
	flag.StringVar(&url, "url", "", "url to grab images from. should have pattern http://ck101.com/thread-2593278-1-1.html")
	flag.StringVar(&username, "u", "", "username")
	flag.StringVar(&password, "p", "", "password")
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
	matches := regexp.MustCompile("thread-(\\d+)-.*").FindStringSubmatch(path.Base(url))
	if matches == nil || len(matches) < 2 {
		log.Fatalf("URL should be in the form of http://ck101.com/thread-2593278-1-1.html")
	}
	threadId := matches[1]

	l := ck101.NewCK101Lover(username, password)

	b, err := l.GrabPage(url)
	if err != nil {
		log.Fatalf("Failed to grab the page: %v", err)
	}

	targetDir := filepath.Join(basedir, fmt.Sprintf("%s - %s", threadId, b.Title))
	ck101.GrabImages(b, targetDir)
}
