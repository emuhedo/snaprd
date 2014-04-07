package main

import (
    "io/ioutil"
    "log"
    "os"
    "path/filepath"
    "time"
    "strconv"
)

type SnapshotState int

const (
    STATE_INCOMPLETE SnapshotState = 1 << iota
    STATE_COMPLETE
    STATE_OBSOLETE
    STATE_INDELETION
)

func (st SnapshotState) String() string {
    s := ""
    if st & STATE_INCOMPLETE == STATE_INCOMPLETE {
        s += ":Incomplete"
    }
    if st & STATE_COMPLETE == STATE_COMPLETE {
        s += ":Complete"
    }
    if st & STATE_OBSOLETE == STATE_OBSOLETE {
        s += ":Obsolete"
    }
    if st & STATE_INDELETION == STATE_INDELETION {
        s += ":Indeletion"
    }
    return s
}

type Snapshot struct {
    startTime int64
    endTime int64
    state SnapshotState
    dirname string
}

func newSnapshot(startTime int64, endTime int64, state SnapshotState, dirname string) *Snapshot {
    return &Snapshot{startTime, endTime, state, dirname}
}

func (s *Snapshot) String() string {
    stime := strconv.FormatInt(s.startTime, 10)
    etime := strconv.FormatInt(s.endTime, 10)
    return stime + "-" + etime + " S" + s.state.String() + "(" + s.dirname + ")"
}

type SnapshotList []*Snapshot

func unixTimestamp() string {
    return strconv.FormatInt(time.Now().Unix(), 10)
}

func newSnapshotName() string {
    ts := unixTimestamp()
    return ts
}

func finishedSnapshotName(incompleteName string) string {
    return incompleteName + "-" + unixTimestamp()
}

func isSnapshot(f os.FileInfo) bool {
    if !f.IsDir() {
        return false
    }
    // number-number OR
    // number-
    return true
}

func FindSnapshots() SnapshotList {
    snapshots := make(SnapshotList, 0, 256)
    files, err := ioutil.ReadDir(filepath.Join(config.dstPath, ""))
    if err != nil {
        log.Panic(err)
    }
    for _, f := range files {
        if isSnapshot(f) {
            s := newSnapshot(12345, 23456, STATE_COMPLETE | STATE_OBSOLETE | STATE_INDELETION, f.Name())
            snapshots = append(snapshots, s)
        } else {
            log.Println(f.Name() + " is not a snapshot")
        }
    }
    return snapshots
}
