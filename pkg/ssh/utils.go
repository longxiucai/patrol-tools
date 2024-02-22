package ssh

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/golang/glog"
	"golang.org/x/sync/errgroup"
)

func fileExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func ReadPipe(stdout, stderr io.Reader, isStdout bool) {
	var combineSlice []string
	var combineLock sync.Mutex
	doneout := make(chan error, 1)
	doneerr := make(chan error, 1)
	go func() {
		doneerr <- readPipe(stderr, &combineSlice, &combineLock, isStdout)
	}()
	go func() {
		doneout <- readPipe(stdout, &combineSlice, &combineLock, isStdout)
	}()
	<-doneerr
	<-doneout
}

func readPipe(pipe io.Reader, combineSlice *[]string, combineLock *sync.Mutex, isStdout bool) error {
	r := bufio.NewReader(pipe)
	for {
		line, _, err := r.ReadLine()
		if err != nil {
			return err
		}

		combineLock.Lock()
		*combineSlice = append(*combineSlice, string(line))
		glog.Infof("command execution result is: %s", line)
		if isStdout {
			fmt.Println(string(line))
		}
		combineLock.Unlock()
	}
}

func WaitSSHReady(ssh Interface, tryTimes int, hosts ...net.IP) error {
	var err error
	eg, _ := errgroup.WithContext(context.Background())
	for _, h := range hosts {
		host := h
		eg.Go(func() error {
			for i := 0; i < tryTimes; i++ {
				err = ssh.Ping(host)
				if err == nil {
					return nil
				}
				time.Sleep(time.Duration(i) * time.Second)
			}
			return fmt.Errorf("wait for [%s] ssh ready timeout: %v, ensure that the IP address or password is correct", host, err)
		})
	}
	return eg.Wait()
}
