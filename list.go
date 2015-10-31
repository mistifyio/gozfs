package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"syscall"

	"github.com/mistifyio/gozfs/nv"
)

type header struct {
	Size     uint32
	ExtSpace uint8
	Error    uint8
	Endian   uint8
	Reserved uint8
}

func getSize(b []byte) (int64, error) {
	h := header{}
	buf := bytes.NewBuffer(b)
	err := binary.Read(buf, binary.LittleEndian, &h)
	if err != nil {
		return 0, err
	}

	if h.Endian != 1 {
		buf := bytes.NewBuffer(b)
		err := binary.Read(buf, binary.BigEndian, &h)
		if err != nil {
			return 0, err
		}
	}

	size := uint(h.Size + uint32(h.ExtSpace))
	if h.Error != 0 {
		err = syscall.Errno(h.Error)
	}
	return int64(size), err
}

func list(name string, types map[string]bool, recurse bool, depth uint64) ([]map[string]interface{}, error) {
	var reader io.Reader
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer reader.(*os.File).Close()
	defer writer.Close()

	opts := map[string]interface{}{
		"fd": int32(writer.Fd()),
	}
	if types != nil {
		opts["type"] = types
	}
	if recurse != false {
		if depth != 0 {
			opts["recurse"] = depth
		} else {
			opts["recurse"] = true
		}
	}
	args := map[string]interface{}{
		"cmd":     "zfs_list",
		"innvl":   map[string]interface{}{},
		"opts":    opts,
		"version": uint64(0),
	}

	enc, err := nv.Encode(args)
	if err != nil {
		return nil, err
	}

	err = ioctl(zfs, name, enc, nil)
	if err != nil {
		return nil, err
	}

	var buf []byte
	reader = bufio.NewReader(reader)

	ret := []map[string]interface{}{}
	for {
		header := make([]byte, 8)
		_, err := io.ReadFull(reader, header)
		if err != nil {
			return nil, err
		}

		size, err := getSize(header)
		if err != nil {
			panic(err)
		}
		if size == 0 {
			break
		}

		if len(buf) < int(size) {
			l := (size + 1023) & ^1023
			buf = make([]byte, l)
		}
		buf = buf[:size]

		_, err = io.ReadFull(reader, buf)
		if err != nil {
			return nil, err
		}

		m := map[string]interface{}{}
		err = nv.Decode(buf, &m)
		if err != nil {
			panic(err)
		}
		ret = append(ret, m)
	}
	return ret, nil
}