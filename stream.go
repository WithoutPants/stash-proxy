package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

func getStreamArgs(input string, start string) []string {
	args := []string{
		"-hide_banner",
		"-v", "error",
	}

	if start != "" {
		args = append(args, "-ss", start)
	}

	args = append(args,
		"-i", input,
	)

	args = append(args,
		"-c:v", "libvpx-vp9",
	)

	args = append(args, "-deadline", "realtime",
		"-cpu-used", "5",
		"-row-mt", "1",
		"-crf", "30",
		"-b:v", "0")

	args = append(args,
		// this is needed for 5-channel ac3 files
		"-ac", "2",
		"-f", "webm",
		"pipe:",
	)

	return args
}

func stream(c *config, input string, start string) (*Stream, error) {
	ffmpegPath := c.FFmpegPath
	args := getStreamArgs(input, start)
	cmd := exec.Command(ffmpegPath, args...)
	setSysProcAttr(cmd)

	if c.LogDebug {
		fmt.Printf("Streaming via: %s\n", strings.Join(cmd.Args, " "))
	}

	stdout, err := cmd.StdoutPipe()
	if nil != err {
		log.Println("FFMPEG stdout not available: " + err.Error())
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if nil != err {
		log.Println("FFMPEG stderr not available: " + err.Error())
		return nil, err
	}

	if err = cmd.Start(); err != nil {
		return nil, err
	}

	// stderr must be consumed or the process deadlocks
	go func() {
		stderrData, _ := ioutil.ReadAll(stderr)
		stderrString := string(stderrData)
		if len(stderrString) > 0 {
			log.Printf("[stream] ffmpeg stderr: %s\n", stderrString)
		}
	}()

	ret := &Stream{
		Stdout:  stdout,
		Process: cmd.Process,
	}
	return ret, nil
}

type Stream struct {
	Stdout  io.ReadCloser
	Process *os.Process
}

func (s *Stream) Serve(w http.ResponseWriter, r *http.Request) {
	const mimeType = "video/webm"
	w.Header().Set("Content-Type", mimeType)
	w.WriteHeader(http.StatusOK)

	// handle if client closes the connection
	notify := r.Context().Done()
	go func() {
		<-notify
		if s.Process != nil {
			s.Process.Kill()
		}
	}()

	io.Copy(w, s.Stdout)
}
