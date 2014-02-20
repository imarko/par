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
	cmd  string
	args []string
	out  string
	err  error
}

var runners = flag.Int("j", 20, "number of concurrent jobs")
var retries = flag.Int("r", 1, "try failing jobs this many times")
var replace = flag.String("i", "...", "arg pattern to be replaced with inputs")

func runner(in chan *job, out chan *job) {
	for j := range in {
		for try := 0; try < *retries; try++ {
			c := exec.Command(j.cmd, j.args...)
			o, err := c.CombinedOutput()
			j.err = err
			j.out = string(o)
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
				fmt.Printf("%s: %s\n", j.name, l)
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
		args := []string{}
		added := false
		for _, a := range preargs {
			if a == *replace {
				args = append(args, l)
				added = true
			} else {
				args = append(args, a)
			}
		}
		if !added {
			args = append(args, l)
		}
		j := &job{
			name: l,
			cmd:  args[0],
			args: args[1:],
		}
		submitter <- j
		wg.Add(1)
	}
	close(submitter)
	wg.Wait()
}
