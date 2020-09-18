package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"strings"
	"unicode"
)

const (
	Begin = "ServerURLHere->"
	End = "<-ServerURLHere"
)

type Writer struct {
	result chan []byte
	str []rune
	pos int
	old []byte
	step int
	value string
	found bool
}

func (w *Writer) Write(p []byte) (n int, err error) {
	if w.found {
		return len(p), nil
	}
	var b strings.Builder
	if w.old != nil && len(w.old) > 0 {
		fmt.Printf("Old [%v]\n", w.old)
		b.Write(w.old)
	}
	b.Write(p)
	r := bufio.NewReader(strings.NewReader(b.String()))
	n = 0
	for {
		if c, sz, err := r.ReadRune(); err != nil {
			if err == io.EOF {
				break
			} else {
				log.Fatal(err)
				return 0, err
			}
		} else {
			if c != unicode.ReplacementChar {
				fmt.Print(string(c))
				n += sz
				if w.str[w.pos] == c {
					w.pos ++
				} else {
					if w.step == 1 {
						if w.pos != 0 {
							w.value += string(w.str[0: w.pos])
							w.pos = 0
						}
						if w.str[w.pos] == c {
							w.pos ++
						} else {
							w.value += string(c)
						}
					}
					if w.step == 0 && w.pos != 0 {
						w.pos = 0
					}
				}
				if w.pos == len(w.str) {
					if w.step == 0 {
						w.step = 1
						w.str = []rune(End)
						w.pos = 0
					} else {
						if w.step == 1 {
							w.found = true
							fmt.Println("\nFound", w.value)
							w.result <- []byte(w.value)
							return len(p), nil
						}
					}
				}
			} else {

			}
		}
	}
	return n, nil
}

func (w *Writer) Start(udid string, c chan []byte) {
	cmd := exec.Command("xcodebuild", "test-without-building",
		"-project",
		"./WebDriverAgent/WebDriverAgent.xcodeproj",
		"-scheme",
		"WebDriverAgentRunner",
		"-destination",
		"id=" + string(bytes.Trim([]byte (udid), "\x00")),
		"-xcconfig",
		"./wda-build.xcconfig")
	//cmd := exec.Command("cat", "sample.log")
	//cmd := exec.Command("./test.sh")
	var out Writer
	out.str = []rune(Begin)
	out.pos = 0
	out.value = ""
	out.result = c

	cmd.Stdout = &out

	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
		w.result <- nil
		return
	}
	go func() {
		err = cmd.Wait()
		if !w.found {
			log.Info("Finished: ", err)
			w.result <- nil
		}
	}()
}
