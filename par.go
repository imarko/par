package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type job struct {
	name string
	cmd  []string
	out  string
	err  error
}

var runners = flag.Int("j", 20, "number of concurrent jobs")
var retries = flag.Int("r", 1, "try failing jobs this many times")
var replace = flag.String("i", "", "arg pattern to be replaced with inputs")
var bare = flag.Bool("b", false, "show bare output without prefix")
var verbose = flag.Bool("v", false, "verbose output, show task start/finish and exit status")

func runner(in chan *job, out chan *job) {
	for j := range in {
		for try := 0; try < *retries; try++ {
			if *verbose {
				fmt.Printf("%s: STARTING\n", j.name)
			}
			c := exec.Command(j.cmd[0], j.cmd[1:]...)
			o, err := c.CombinedOutput()
			j.err = err
			j.out = string(o)
			if *verbose {
				if err == nil {
					fmt.Printf("%s: SUCCEES\n", j.name)
				} else {
					fmt.Printf("%s: FAILED: %s\n", j.name, err)
				}
			}
			if err == nil {
				break
			}
		}
		out <- j
	}
}

func printer(in <-chan *job, wg *sync.WaitGroup) {
	for j := range in {
		if j.err != nil {
			fmt.Printf("%s: %s\n", j.name, j.err.Error())
		}
		lines := strings.Split(j.out, "\n")
		for _, l := range lines {
			if l != "" {
				if *bare {
					fmt.Println(l)
				} else {
					fmt.Printf("%s: %s\n", j.name, l)
				}
			}
		}
		wg.Done()
	}
}
func main() {
	flag.Parse()
	preargs := []string{"/bin/echo"}
	if flag.NArg() > 0 {
		preargs = flag.Args()
	}

	submitter := make(chan *job)
	results := make(chan *job, 1000)
	wg := &sync.WaitGroup{}

	go printer(results, wg)

	for i := 0; i < *runners; i++ {
		go runner(submitter, results)
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		l := scanner.Text()
		cmd := make([]string, 0)
		if *replace == "" {
			cmd = append(preargs, l)
		} else {
			for _, a := range preargs {
				cmd = append(cmd, strings.Replace(a, *replace, l, -1))
			}
		}
		j := &job{
			name: l,
			cmd:  cmd,
		}
		wg.Add(1)
		submitter <- j
	}
	close(submitter)
	wg.Wait()
}
