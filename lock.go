/* See the file "LICENSE.txt" for the full license governing this code. */

// Simple lock file mechanism to prevent multiple instances to run

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

type pidLocker struct {
	pid int
	f   string
}

func newPidLocker(lockfile string) *pidLocker {
	return &pidLocker{
		pid: os.Getpid(),
		f:   lockfile,
	}
}

func (pl *pidLocker) Lock() error {
	_, err := os.Stat(pl.f)
	if err == nil {
		return fmt.Errorf("pid file %s already exists. Is snaprd running already?", pl.f)
	}
	debugf("write pid %d to pidfile %s", pl.pid, pl.f)
	err = ioutil.WriteFile(pl.f, []byte(strconv.Itoa(pl.pid)), 0666)
	if err != nil {
		return fmt.Errorf("could not write pid file %s: %s", pl.f, err)
	}
	return nil
}

func (pl *pidLocker) Unlock() {
	debugf("delete pidfile %s", pl.f)
	err := os.Remove(pl.f)
	if err != nil {
		log.Printf("could not remove pid file %s: %s", pl.f, err)
	}
}
