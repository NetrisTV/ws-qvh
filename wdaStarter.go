package main

import (
	"bufio"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"os/exec"
	"strings"
	"unicode"
)

const (
	Begin = "ServerURLHere->"
	End   = "<-ServerURLHere"
)

type WdaProcess struct {
	udid   string
	result *chan *MessageRunWda
	exit   *chan interface{}
	str    []rune
	pos    int
	old    []byte
	step   int
	value  string
	found  bool
}

func (w *WdaProcess) Write(p []byte) (n int, err error) {
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
					w.pos++
				} else {
					if w.step == 1 {
						if w.pos != 0 {
							w.value += string(w.str[0:w.pos])
							w.pos = 0
						}
						if w.str[w.pos] == c {
							w.pos++
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
							msg := NewMessageRunWda(w.udid, 0, w.value)
							*w.result <- &msg
							w.result = nil
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

func (w *WdaProcess) Start() {
	cmd := exec.Command("xcodebuild", "test-without-building",
		"-project",
		"./WebDriverAgent/WebDriverAgent.xcodeproj",
		"-scheme",
		"WebDriverAgentRunner",
		"-destination",
		"id="+strings.Trim(w.udid, "\x00"),
		"-xcconfig",
		"./wda-build.xcconfig")
	cmd.Stdout = w

	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
		if w.result != nil {
			*w.result <- nil
			w.result = nil
		}
		return
	}
	go func() {
		rr := cmd.Wait()
		select {
		case *w.exit <- nil:
			break
		default:
			break
		}
		if !w.found {
			if exitError, ok := rr.(*exec.ExitError); ok {
				waitStatus := exitError.ExitCode()
				if w.result != nil {
					msg := NewMessageRunWda(w.udid, waitStatus, "failed")
					*w.result <- &msg
					w.result = nil
				}
			}
			log.Debug("Finished: ", rr)
			if w.result != nil {
				*w.result <- nil
				w.result = nil
			}
		}
	}()
}

func NewWdaProcess(udid string, ch *chan *MessageRunWda, exit *chan interface{}) *WdaProcess {
	return &WdaProcess{
		udid:   udid,
		str:    []rune(Begin),
		pos:    0,
		value:  "",
		result: ch,
		exit:   exit,
	}
}
