package main

import (
    "errors"
    "fmt"
    "io/ioutil"
    "log"
    "os"
    "path/filepath"
    "sort"
    "strconv"
    "strings"
    "time"
)

type SnapshotState uint

const NONE SnapshotState = 0
const (
    STATE_INCOMPLETE SnapshotState = 1 << iota
    STATE_COMPLETE
    STATE_OBSOLETE
    STATE_PURGING
    ANY = (1 << iota) - 1
)

func (st SnapshotState) String() string {
    switch st {
    case STATE_INCOMPLETE:
        return "Incomplete"
    case STATE_COMPLETE:
        return "Complete"
    case STATE_OBSOLETE:
        return "Obsolete"
    case STATE_PURGING:
        return "Purging"
    }
    return "Unknown"
}

type Snapshot struct {
    startTime time.Time
    endTime   time.Time
    state     SnapshotState
}

func newSnapshot(startTime, endTime time.Time, state SnapshotState) *Snapshot {
    return &Snapshot{startTime, endTime, state}
}

func newIncompleteSnapshot() *Snapshot {
    return &Snapshot{time.Now(), time.Time{}, STATE_INCOMPLETE}
}

func (s *Snapshot) String() string {
    stime := s.startTime.Unix()
    etime := s.endTime.Unix()
    return fmt.Sprintf("%d-%d %s", stime, etime, s.state.String())
}

func (s *Snapshot) Name() string {
    stime := s.startTime.Unix()
    etime := s.endTime.Unix()
    switch s.state {
    case STATE_INCOMPLETE:
        return fmt.Sprintf("%d-0-incomplete", stime)
    case STATE_COMPLETE:
        return fmt.Sprintf("%d-%d-complete", stime, etime)
    case STATE_OBSOLETE:
        return fmt.Sprintf("%d-%d-obsolete", stime, etime)
    case STATE_PURGING:
        return fmt.Sprintf("%d-%d-purging", stime, etime)
    }
    return fmt.Sprintf("%d-%d-unknown", stime, etime)
}

func (s *Snapshot) FullName() string {
    return filepath.Join(config.repository, DATA_SUBDIR, s.Name())
}

// Mark the latest snapshot for easy access.
// Do not fail if not possible since it is more important
// to continue creating new snapshots.
func tryLink(target string) {
    linkName := filepath.Join(config.repository, "latest")
    fi, err := os.Lstat(linkName)
    if err != nil {
        // link does not exist or can not be read
        log.Println(err)
    }
    if fi != nil {
        // link exists
        if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
            // link is indeed a symlink
            err = os.Remove(linkName)
            if err != nil {
                // link can not be removed
                log.Println(err)
            }
        }
    }
    err = os.Symlink(target, linkName)
    if err != nil {
        log.Println(err)
    } else {
        //log.Println("symlink latest snapshot")
    }
}

func (s *Snapshot) transComplete() {
    oldName := s.FullName()
    etime := time.Now()
    if etime.Before(s.startTime) {
        log.Fatal("endTime before startTime!")
    }
    // make all snapshots at least 1 second long
    if etime.Sub(s.startTime).Seconds() < 1 {
        etime = etime.Add(time.Second)
    }
    s.endTime = etime
    s.state = STATE_COMPLETE
    err := os.Rename(oldName, s.FullName())
    if err != nil {
        log.Fatal(err)
    }
    tryLink(filepath.Join(DATA_SUBDIR, s.Name()))
}

func (s *Snapshot) transObsolete() {
    oldName := s.FullName()
    s.state = STATE_OBSOLETE
    newName := s.FullName()
    err := os.Rename(oldName, newName)
    if err != nil {
        log.Fatal(err)
    }
}

func (s *Snapshot) transPurging() {
    oldName := s.FullName()
    s.state = STATE_PURGING
    newName := s.FullName()
    err := os.Rename(oldName, newName)
    if err != nil {
        log.Fatal(err)
    }
}

func (s *Snapshot) purge() {
    s.transPurging()
    path := s.FullName()
    log.Println("purging", s.Name())
    os.RemoveAll(path)
    log.Println("finished purging", s.Name())
}

func (s *Snapshot) matchFilter(f SnapshotState) bool {
    //log.Println("filter:", strconv.FormatInt(int64(s.state), 2), strconv.FormatInt(int64(f), 2), strconv.FormatBool(s.state & f == s.state))
    //log.Println(strconv.FormatInt(int64(ANY), 2))
    return (s.state & f) == s.state
}

type SnapshotList []*Snapshot

// find the last snapshot to use as a basis for the next one
func (sl SnapshotList) lastGood() *Snapshot {
    var t time.Time
    var ix int = -1
    for i, sn := range sl {
        if (sn.startTime.After(t)) && (sn.state == STATE_COMPLETE) {
            t = sn.startTime
            ix = i
        }
    }
    if ix == -1 {
        return nil
    }
    return sl[ix]
}

func parseSnapshotName(s string) (time.Time, time.Time, SnapshotState, error) {
    sa := strings.Split(s, "-")
    var zero time.Time
    if len(sa) != 3 {
        return zero, zero, 0, errors.New("malformed snapshot name: " + s)
    }
    stime, err := strconv.ParseInt(sa[0], 10, 64)
    if err != nil {
        return zero, zero, 0, err
    }
    etime, err := strconv.ParseInt(sa[1], 10, 64)
    if err != nil {
        return zero, zero, 0, err
    }
    var state SnapshotState = 0
    switch sa[2] {
    case "complete":
        state = STATE_COMPLETE
    case "incomplete":
        state = STATE_INCOMPLETE
    case "obsolete":
        state = STATE_OBSOLETE
    case "purging":
        state = STATE_PURGING
    }
    if state == 0 {
        return time.Unix(stime, 0), time.Unix(etime, 0), state, errors.New("could not parse state: " + s)
    }
    return time.Unix(stime, 0), time.Unix(etime, 0), state, nil
}

type SnapshotListByStartTime SnapshotList

func (sl SnapshotListByStartTime) Len() int {
    return len(sl)
}
func (sl SnapshotListByStartTime) Swap(i, j int) {
    sl[i], sl[j] = sl[j], sl[i]
}
func (sl SnapshotListByStartTime) Less(i, j int) bool {
    return sl[i].startTime.Before(sl[j].startTime)
}

func FindSnapshots() (SnapshotList, error) {
    snapshots := make(SnapshotList, 0, 256)
    dataPath := filepath.Join(config.repository, DATA_SUBDIR, "")
    files, err := ioutil.ReadDir(dataPath)
    if err != nil {
        return nil, errors.New("repository " + dataPath + " does not exist")
    }
    for _, f := range files {
        if !f.IsDir() {
            continue
        }
        stime, etime, state, err := parseSnapshotName(f.Name())
        if err != nil {
            log.Println(err)
            continue
        }
        if stime.After(time.Now()) {
            log.Println("ignoring snapshot with startTime in future:", f.Name())
            continue
        }
        sn := newSnapshot(stime, etime, state)
        snapshots = append(snapshots, sn)
    }
    sort.Sort(SnapshotListByStartTime(snapshots))
    return snapshots, nil
}

// return a new list of snapshots that fall into the given time period
func (sl SnapshotList) period(after, before time.Time) SnapshotList {
    slNew := make(SnapshotList, 0, len(sl))
    for _, sn := range sl {
        if sn.startTime.After(after) && sn.startTime.Before(before) {
            slNew = append(slNew, sn)
        }
    }
    Debugf("period(): %s", slNew)
    return slNew
}

// return a list of snapshots within the given interval
func (sl SnapshotList) interval(intervals intervalList, i int) SnapshotList {
    t := time.Now()
    from := t.Add(-intervals.offset(i+1))
    to := t.Add(-intervals.offset(i))
    Debugf("interval from: %s, to: %s", from, to)
    return sl.period(from, to)
}

func (sl SnapshotList) state(include, exclude SnapshotState) SnapshotList {
    slNew := make(SnapshotList, 0, len(sl))
    for _, sn := range sl {
        if sn.matchFilter(include) && sn.matchFilter(^exclude) {
            slNew = append(slNew, sn)
        }
    }
    return slNew
}
