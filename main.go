package main

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/aybabtme/rgbterm"
)

var reErr = regexp.MustCompile(`choose one of: \[(.*)\]`)

func string2color(in string) (r, g, b uint8) {
	md := md5.Sum([]byte(in))
	return md[0], md[1], md[2]
}

func getPos(namespace string) []string {
	cmd := fmt.Sprintf("kubectl -n %s get po | grep Running | awk '{ print $1 }'", namespace)
	res := exec.Command("/bin/sh", "-c", cmd)
	b, err := res.Output()
	if err != nil {
		return []string{}
	}
	pos := strings.Fields(string(b))
	return pos
}

func main() {
	namespace := os.Args[1]
	pos := getPos(namespace)
	posCh := make(chan string)
	go func() {
		for _, p := range pos {
			posCh <- p
		}
	}()
	for pod := range posCh {
		go func(pod string) {
			cmd := fmt.Sprintf("kubectl -n %s logs -f %s", namespace, pod)
			c := exec.Command("/bin/sh", "-c", cmd)
			po, err := c.StdoutPipe()
			if err != nil {
				log.Println(err)
				return
			}
			pe, err := c.StderrPipe()
			if err != nil {
				log.Println(err)
				return
			}
			r, g, b := string2color(pod)
			prefix := rgbterm.FgString(fmt.Sprintf("%v:", pod), r, g, b)
			go reader(po, prefix, pod, posCh)
			go reader(pe, prefix, pod, posCh)
			c.Start()
		}(pod)
	}
}

func reader(rd io.Reader, prefix, pod string, ch chan string) {
	reader := bufio.NewReader(rd)
	for {
		l, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Println(prefix, err)
			}
			break
		}
		l = strings.TrimRight(l, "\n")
		if reErr.MatchString(l) {
			p := reErr.FindStringSubmatch(l)[1]
			po := strings.Fields(p)
			for _, v := range po {
				ch <- pod + " " + v
			}
		}
		fmt.Println(prefix, l)

	}
}
