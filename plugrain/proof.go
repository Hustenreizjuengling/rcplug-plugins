package main

import (
	"os"
	"os/user"
	"strconv"
	"strings"
)

// The proof of root on the page: what a shell on the device would print,
// read by this process itself, so it is as true as the page is (a plugin
// that does not run as root shows "$" and its own id).

// system is what the proof is made of; tests fill it in.
type system struct {
	uid, gid int
	groups   []int
	host     string
	uname    string // kernel name, release and machine, as uname -srm
	model    string // /proc/device-tree/model
	rcplug   string
}

func thisSystem() system {
	s := system{uid: os.Getuid(), gid: os.Getgid(), uname: unameSRM(), rcplug: os.Getenv("RCPLUG_VERSION")}
	s.groups, _ = os.Getgroups()
	s.host, _ = os.Hostname()
	if b, err := os.ReadFile("/proc/device-tree/model"); err == nil {
		s.model = strings.TrimSpace(strings.TrimRight(string(b), "\x00"))
	}
	return s
}

// proof is the terminal's title, prompt and lines.
func (s system) proof() (title, prompt string, lines []TermLine) {
	name := func(id int, lookup func(string) (string, error)) string {
		n, err := lookup(strconv.Itoa(id))
		if err != nil || n == "" {
			return strconv.Itoa(id)
		}
		return strconv.Itoa(id) + "(" + n + ")"
	}
	userName := func(id string) (string, error) {
		u, err := user.LookupId(id)
		if err != nil {
			return "", err
		}
		return u.Username, nil
	}
	groupName := func(id string) (string, error) {
		g, err := user.LookupGroupId(id)
		if err != nil {
			return "", err
		}
		return g.Name, nil
	}
	if s.uid == 0 {
		userName = func(string) (string, error) { return "root", nil }
	}
	if s.gid == 0 {
		g := groupName
		groupName = func(id string) (string, error) {
			if id == "0" {
				return "root", nil
			}
			return g(id)
		}
	}
	groups := s.groups
	if len(groups) == 0 {
		groups = []int{s.gid}
	}
	var gs []string
	for _, g := range groups {
		gs = append(gs, name(g, groupName))
	}
	who, _ := userName(strconv.Itoa(s.uid))
	if who == "" {
		who = strconv.Itoa(s.uid)
	}
	title = who + "@" + s.host
	prompt = "$ "
	if s.uid == 0 {
		prompt = "# "
	}
	lines = []TermLine{
		{Cmd: true, Text: "id"},
		{Text: "uid=" + name(s.uid, userName) + " gid=" + name(s.gid, groupName) + " groups=" + strings.Join(gs, ",")},
	}
	if s.uname != "" {
		lines = append(lines, TermLine{Cmd: true, Text: "uname -srm"}, TermLine{Text: s.uname})
	}
	if s.model != "" {
		lines = append(lines, TermLine{Cmd: true, Text: "cat /proc/device-tree/model"}, TermLine{Text: s.model})
	}
	if s.rcplug != "" {
		lines = append(lines, TermLine{Cmd: true, Text: "rcplugd --version"}, TermLine{Text: s.rcplug})
	}
	return title, prompt, lines
}
