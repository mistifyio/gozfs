package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"

	"github.com/stretchr/testify/suite"
)

var (
	einval = syscall.EINVAL.Error()
	enoent = syscall.ENOENT.Error()
)

type internal struct {
	pool  string
	files []string
	suite.Suite
}

func TestSuiteInternal(t *testing.T) {
	suite.Run(t, &internal{})
}

func (s *internal) create(pool string) {
	s.pool = pool
	files := make([]string, 5)
	for i := range files {
		f, err := ioutil.TempFile("", "gozfs-test-temp")
		if err != nil {
			panic(err)
		}
		files[i] = f.Name()
		f.Close()
	}
	s.files = files

	script := []byte(`
	set -e
	pool=$1
	shift
	files=($@)
	for f in ${files[*]}; do
		truncate -s1G $f
	done
	zpool create $pool ${files[*]}

	zfs create $pool/a
	zfs create $pool/a/1
	zfs create $pool/a/2
	zfs snapshot $pool/a/1@snap1
	zfs clone $pool/a/1@snap1 $pool/a/3
	zfs snapshot $pool/a/2@snap1
	zfs hold hold1 $pool/a/2@snap1
	zfs snapshot $pool/a/2@snap2
	zfs hold hold2 $pool/a/2@snap2
	zfs create $pool/a/4
	zfs unmount $pool/a/4

	zfs create $pool/b
	zfs create -V 8192 $pool/b/1
	zfs create -b 1024 -V 2048 $pool/b/2
	zfs snapshot $pool/b/1@snap1
	zfs clone $pool/b/1@snap1 $pool/b/3
	zfs snapshot $pool/b/2@snap1
	zfs hold hold1 $pool/b/2@snap1
	zfs snapshot $pool/b/2@snap2
	zfs hold hold2 $pool/b/2@snap2
	zfs create -V 8192 $pool/b/4
	exit 0
	`)

	args := make([]string, 3, 3+len(files))
	args[0] = "bash"
	args[1] = "/dev/stdin"
	args[2] = s.pool
	args = append(args, files...)
	cmd := exec.Command("sudo", args...)
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		s.Require().NoError(err)
	}
	go func() {
		_, err := stdin.Write([]byte(script))
		if err != nil {
			s.Require().NoError(err)
		}
	}()

	s.Require().NoError(cmd.Run())
}

func (s *internal) SetupTest() {
	s.create("gozfs-test")
}

func (s *internal) destroy() {
	cmd := exec.Command("sudo", "zpool", "destroy", s.pool)
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	for i := range s.files {
		os.Remove(s.files[i])
	}
	s.Require().NoError(err)
}

func (s *internal) TearDownTest() {
	s.destroy()
}

func (s *internal) TestListEmpty() {
	s.destroy()

	m, err := list("", nil, true, 0)
	s.NoError(err, "m: %v", m)
	s.Assert().Len(m, 0)

	s.create(s.pool)
}

func (s *internal) TestList() {
	type t map[string]bool
	tests := []struct {
		name    string
		types   t
		recurse bool
		depth   uint64
		expNum  int
		err     string
	}{
		{name: "blah", err: enoent},
		{name: strings.Repeat("a", 257), err: einval},
		{name: s.pool, expNum: 1},
		{name: s.pool, recurse: true, expNum: 7},
		{name: s.pool, recurse: true, depth: 1, expNum: 3},
		{name: s.pool + "/a", recurse: true, expNum: 5},
		{name: s.pool + "/a", recurse: true, depth: 1, expNum: 5},
		{name: s.pool, types: t{"volume": true}, recurse: true, expNum: 4},
		{name: s.pool, types: t{"volume": true}, recurse: true, depth: 1, expNum: 0},
		{name: s.pool + "/a", types: t{"volume": true}, recurse: true, expNum: 0},
		{name: s.pool + "/b", types: t{"volume": true}, recurse: true, expNum: 4},
		{name: s.pool + "/b", types: t{"volume": true}, recurse: true, depth: 1, expNum: 4},
		{name: s.pool, types: t{"snapshot": true}, recurse: true, expNum: 6},
		{name: s.pool, types: t{"snapshot": true}, recurse: true, depth: 1, expNum: 0},
		{name: s.pool + "/a", types: t{"snapshot": true}, recurse: true, expNum: 3},
		{name: s.pool + "/a/1", types: t{"snapshot": true}, recurse: true, expNum: 1},
		{name: s.pool + "/b", types: t{"snapshot": true}, recurse: true, expNum: 3},
		{name: s.pool + "/b/1", types: t{"snapshot": true}, recurse: true, expNum: 1},
	}
	for i, test := range tests {
		m, err := list(test.name, test.types, test.recurse, test.depth)
		if test.err != "" {
			s.EqualError(err, test.err, "test num:%d", i)
			s.Nil(m, "test:%d", i)
		} else {
			s.NoError(err, "test:%d", i)
			s.Len(m, test.expNum, "test num:%d", i)
		}
	}
}
