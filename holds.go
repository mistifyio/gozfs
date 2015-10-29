package main

import (
	"bytes"
	"os"

	"github.com/mistifyio/gozfs/nv"
)

func holds(name string) error {
	m := map[string]interface{}{
		"cmd":     "zfs_get_holds",
		"version": uint64(0),
	}

	encoded, err := nv.Encode(m)
	if err != nil {
		return err
	}

	out := make([]byte, 1024)
	err = ioctl(zfs, name, encoded, out)
	if err != nil {
		return err
	}

	var o bytes.Buffer
	err = nv.PrettyPrint(&o, out, " ")
	if err != nil {
		return err
	}
	o.WriteTo(os.Stdout)

	return nil
}
