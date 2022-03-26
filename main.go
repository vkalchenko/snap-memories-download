package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type MemoriesHistory struct {
	Media []SavedMedia `json:"Saved Media"`
}

type SavedMedia struct {
	Date         string `json:"Date"`
	MediaType    string `json:"Media Type"`
	DownloadLink string `json:"Download Link"`
}

func main() {

	flag.Usage = func() {
		_, err := fmt.Fprintf(os.Stderr, getHelp())
		if err != nil {
			panic(err)
		}
	}
	flag.Parse()

	//get arguments from the user
	if len(os.Args) < 3 {
		_, err := fmt.Fprintf(os.Stderr, getHelp())
		if err != nil {
			panic(err)
		}
		os.Exit(0)
	}

	myData := os.Args[1]
	dest := os.Args[2]

	dat, err := os.ReadFile(filepath.FromSlash(myData + "/json/memories_history.json"))
	if err != nil {
		fmt.Println("Please verify path specified " + myData)
		os.Exit(2)
	}

	var mediaData = MemoriesHistory{}
	if err := json.Unmarshal(dat, &mediaData); err != nil {
		panic(err)
	}

	filesNum := len(mediaData.Media)
	if filesNum <= 0 {
		fmt.Println("Empty media files list " + myData)
		os.Exit(2)
	}

	if _, err := os.Stat(dest); os.IsNotExist(err) {
		err := os.Mkdir(dest, os.ModeDir)
		if err != nil {
			panic(err)
		}
	}

	var GOROUTINES_NUM = 10
	if filesNum < GOROUTINES_NUM {
		GOROUTINES_NUM = filesNum
	}

	var wg sync.WaitGroup
	ch := make(chan SavedMedia)

	for i := 0; i < GOROUTINES_NUM; i++ {
		wg.Add(1)
		go getMedia(ch, &wg, dest)
	}

	for _, m := range mediaData.Media {
		ch <- m
	}

	close(ch)
	wg.Wait()

	fmt.Printf("Done! %v files were downloaded.", filesNum)
	os.Exit(0)
}

func getHelp() string {
	return "Usage of ./snap-mem-download: \n./snap-mem-download <mydata> <destination>\n - <mydata> - path to mydata folder\n - <destination> - folder for downloaded media files\n"
}

func getMedia(ch <-chan SavedMedia, wg *sync.WaitGroup, path string) error {
	defer wg.Done()

	for m := range ch {
		client := http.Client{
			Timeout: 10 * time.Second,
		}

		// Get the direct download link
		req, _ := http.NewRequest("POST", m.DownloadLink, nil)
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}

		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			fmt.Println("Server responded with code", resp.StatusCode)
			os.Exit(2)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}

		filename := generateFilename(m)

		//Get file
		req, _ = http.NewRequest("GET", string(body), nil)
		resp, err = client.Do(req)
		if err != nil {
			panic(err)
		}

		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			fmt.Println("Server responded with code", resp.StatusCode)
			os.Exit(2)
		}

		err = writeFile(filepath.FromSlash(path+"/"+filename), resp.Body)
		if err != nil {
			return err
		}
	}
	return nil
}

// Create the file
func writeFile(filepath string, body io.Reader) error {

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, body)
	if err != nil {
		return err
	}

	return nil
}

func generateFilename(m SavedMedia) string {
	ext := "dat"
	if m.MediaType == "Image" {
		ext = "jpg"
	}
	if m.MediaType == "Video" {
		ext = "mp4"
	}

	t, _ := time.Parse("2006-01-02 15:04:05 UTC", m.Date)
	filename := fmt.Sprintf("%s_%v.%s", "snapchat", t.Unix(), ext)

	return filename
}
