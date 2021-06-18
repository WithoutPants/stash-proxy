package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
)

var streamRE = regexp.MustCompile(`\/stream\..+`)

func openLogFile(fn string) (*os.File, error) {
	return os.OpenFile("log.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
}

func main() {
	c, err := loadConfig()
	if err != nil {
		panic(err)
	}

	if c.LogFile != "" {
		f, err := openLogFile(c.LogFile)
		if err != nil {
			panic(err)
		}

		defer f.Close()
		log.SetOutput(f)
	}

	address := c.Host + ":" + strconv.Itoa(c.Port)

	http.HandleFunc("/", handleRequestAndRedirect(c))
	go func() {
		fmt.Printf("Running stash proxy on %s\n", address)
		err := http.ListenAndServe(address, nil)
		if err != nil {
			log.Println(err.Error())
		}
	}()

	if c.ChromePath != "" {
		err = openChromeAndWait(c)
		if err != nil {
			log.Println(err.Error())
		}
	} else {
		// just block forever
		select {}
	}
}

func openChromeAndWait(c *config) error {
	// make temporary user-data-dir
	dataDir, err := os.MkdirTemp(os.TempDir(), "stash-proxy*")
	if err != nil {
		return err
	}

	if dataDir == "" {
		return errors.New("could not create temp directory")
	}

	defer func() {
		if err := os.RemoveAll(dataDir); err != nil {
			log.Printf("error deleting temporary directory %s: %s", dataDir, err.Error())
		}
	}()

	url := c.Host
	if url == "" {
		url = "localhost"
	}
	address := url + ":" + strconv.Itoa(c.Port)

	args := []string{
		"--incognito",
		"--new-window",
		`--user-data-dir=` + dataDir,
		"http://" + address,
	}
	p := exec.Command(c.ChromePath, args...)

	err = p.Start()
	if err != nil {
		return err
	}

	p.Process.Wait()

	return nil
}

func serveReverseProxy(target string, res http.ResponseWriter, req *http.Request) {
	// parse the url
	url, _ := url.Parse(target)

	// create the reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(url)

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)
}

// Given a request send it to the appropriate url
func handleRequestAndRedirect(c *config) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		if streamRE.MatchString(req.URL.Path) {
			localStream(c, res, req)
			return
		}

		serveReverseProxy(c.ServerURL, res, req)
	}
}

func localStream(c *config, res http.ResponseWriter, req *http.Request) {
	req.URL.Path = streamRE.ReplaceAllString(req.URL.Path, "/stream")

	query := req.URL.Query()
	if query.Get("apikey") == "" && c.ApiKey != "" {
		query.Add("apikey", c.ApiKey)
	}

	// TODO - handle resolution
	// resolution := query.Get("resolution")
	// query.Del("resolution")

	start := query.Get("start")
	query.Del("start")

	req.URL.RawQuery = query.Encode()

	remoteURL, _ := url.Parse(c.ServerURL)
	req.URL.Scheme = remoteURL.Scheme
	req.URL.Host = remoteURL.Host

	s, _ := stream(c, req.URL.String(), start)
	s.Serve(res, req)
}
