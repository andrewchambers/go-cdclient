package cdclient

import (
	"bufio"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// AuthFile implements the collectd password file format..
// The file has a very simple syntax with one username / password mapping per
// line, separated by a colon. For example:
//
//   alice: w0nderl4nd
//   bob:   bu1|der
type AuthFile struct {
	path string
	last time.Time
	tab  map[string]string
	lock sync.RWMutex
}

// NewAuthFile initializes and returns a new AuthFile.
// A valid (but empty) authfile is still returned even if there is an error.
func NewAuthFile(path string) (*AuthFile, error) {
	a := &AuthFile{
		path: path,
		lock: sync.RWMutex{},
		tab:  make(map[string]string),
	}
	_, err := a.Refresh()
	if err != nil {
		return a, err
	}
	return a, nil
}

// Password looks up a user in the file and returns the associated password.
func (a *AuthFile) Password(user string) (string, bool) {
	a.lock.RLock()
	defer a.lock.RUnlock()
	pwd, ok := a.tab[user]
	return pwd, ok
}

// Reload the auth file, returns (true, nil) when
// there was no error.
func (a *AuthFile) Refresh() (bool, error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	fi, err := os.Stat(a.path)
	if err != nil {
		return false, err
	}

	if !(len(a.tab) == 0) && !fi.ModTime().After(a.last) {
		return false, nil
	}

	f, err := os.Open(a.path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	newTab := make(map[string]string)

	r := bufio.NewReader(f)
	for {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return false, err
		}

		line = strings.Trim(line, " \r\n\t\v")
		fields := strings.SplitN(line, ":", 2)
		if len(fields) != 2 {
			continue
		}

		user := strings.TrimSpace(fields[0])
		pass := strings.TrimSpace(fields[1])
		if strings.HasPrefix(user, "#") {
			continue
		}
		newTab[user] = pass
	}

	a.tab = newTab
	a.last = fi.ModTime()
	return true, nil
}
