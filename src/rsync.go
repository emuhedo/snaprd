package main

import (
    "os/exec"
    "log"
    "bufio"
    "io"
    "path/filepath"
    "os"
)

func createRsyncCommand(sn *Snapshot, base *Snapshot) *exec.Cmd {
    cmd := exec.Command(config.rsyncPath)
    args := make([]string, 0, 256)
    args = append(args, config.rsyncPath)
    args = append(args, config.rsyncOpts...)
    if base != nil {
        args = append(args, "--link-dest=" + filepath.Join(config.dstPath, base.Name()))
    }
    args = append(args, config.srcPath, filepath.Join(config.dstPath, sn.Name()))
    cmd.Args = args
    cmd.Dir = config.wrkPath
    log.Println("run:", args)
    return cmd
}

func logPipe(cmd *exec.Cmd) error {
    var err error
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return err
    }
    stderr, err := cmd.StderrPipe()
    if err != nil {
        return err
    }
    stdoutReader := bufio.NewReader(stdout)
    stderrReader := bufio.NewReader(stderr)
    err = cmd.Start()
    if err != nil {
        return err
    }
    for {
        str, err := stderrReader.ReadString('\n')
        if err == io.EOF {
            break
        }
        if err != nil && err != io.EOF {
            return err
        }
        log.Print("<rsync stderr> ", str)
    }
    for {
        str, err := stdoutReader.ReadString('\n')
        if err == io.EOF {
            break
        }
        if err != nil && err != io.EOF {
            return err
        }
        log.Print("<rsync stdout> ", str)
    }
    return nil
}


func CreateSnapshot(c chan error, base *Snapshot) {
    // first snapshot
    if base == nil {
        log.Println("creating destination directory for initial snapshot:", config.dstPath)
        err := os.MkdirAll(config.dstPath, 00755)
        if err != nil {
            log.Fatal(err)
        }
    }
    newSn := newIncompleteSnapshot()
    cmd := createRsyncCommand(newSn, base)
    err := logPipe(cmd)
    if err != nil {
        c <- err
        return
    }
    err = cmd.Wait()
    if err != nil {
        c <- err
        return
    }
    newSn.transComplete()
    log.Println("finished:", newSn.Name())
    c <- nil
}
