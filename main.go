package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
)

var counter Counter

type UrlSet struct {
	XMLName xml.Name `xml:"urlset"`
	Urls    []Url    `xml:"url"`
}
type Url struct {
	Loc    string `xml:"loc"`
	NewLoc string
}

func makeRequest(url string, wg *sync.WaitGroup, bar *progressbar.ProgressBar) {
	client := http.Client{}
	response, err := client.Get(url)
	if err != nil {
		fmt.Println(err)
	}
	defer func() {
		respBodyClose := response.Body.Close()
		if respBodyClose != nil {
			fmt.Print(respBodyClose.Error())

			return
		}
	}()

	barError := bar.Add(1)
	if barError != nil {
		fmt.Printf(barError.Error())
	}

	defer wg.Done()
	counter.Add(response.Status, 1)
}

func getXmlFromUrl(xmlUrl string) []byte {
	resp, getErr := http.Get(xmlUrl)
	if getErr != nil {
		fmt.Printf(getErr.Error())
	}

	body, err := io.ReadAll(resp.Body)

	defer func() {
		respErr := resp.Body.Close()
		if respErr != nil {
			fmt.Printf(respErr.Error())
		}
	}()

	if err != nil {
		fmt.Printf(err.Error())
	}
	return body
}

func printSyncMap(counter *Counter) {
	i := 0
	counter.m.Range(func(key, value interface{}) bool {
		int64val, _ := counter.Get(key.(string))

		fmt.Printf("\t[%d] key: %v, value: %d\n", i, key, int64val)
		i++
		return true
	})
}

func start(url string) {
	bar := progressbar.NewOptions(-1,
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionShowIts(),
		progressbar.OptionShowCount(),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetDescription("[yellow]Trying redirects...[reset]"))

	data := getXmlFromUrl(url)
	var urlSet UrlSet
	var wg sync.WaitGroup

	defer func() {
		finishError := bar.Finish()
		println(fmt.Errorf(finishError.Error()))
	}()

	unmarshalErr := xml.Unmarshal(data, &urlSet)
	if unmarshalErr != nil {
		fmt.Printf(unmarshalErr.Error())
	}

	for _, v := range urlSet.Urls {
		r := rand.Intn(1000000)
		time.Sleep(time.Duration(r) * time.Microsecond)

		url := strings.Replace(v.Loc, "watch", "oroloi", -1)
		v.NewLoc = url
		wg.Add(1)

		go makeRequest(url, &wg, bar)
	}

	wg.Wait()

	printSyncMap(&counter)
}

func main() {
	app := &cli.App{
		Name:  "redtest",
		Usage: "testing redirects",
		Action: func(cCtx *cli.Context) error {
			fmt.Printf("Running for %s\n", cCtx.Args().Get(0))
			start(cCtx.Args().Get(0))
			return nil
		},
	}

	err := app.Run(os.Args)

	if err != nil {
		fmt.Printf(err.Error())
	}
}
