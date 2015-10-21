package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"gozfs/nv"
)

func printList(indent string, m map[string]interface{}) {
	for name, v := range m {
		value, ok := v.(map[string]interface{})
		if !ok {
			fmt.Printf("%sName: %s, Value:%+v\n", indent, name, v)
			continue
		}

		if value["type"] == "NVLIST" {
			fmt.Printf("%sName: %s, Type: %s\n", indent, name, value["type"])
			printList(strings.Repeat(" ", len(indent)+2), value["value"].(map[string]interface{}))
		} else {
			fmt.Printf("%sName: %s, Type: %s, Value:%+v\n",
				indent, name, value["type"], value["value"])
		}
	}
}

func main() {
	skip := flag.Int("skip", 0, "number of leading bytes to skip")
	flag.Parse()

	if *skip > 0 {
		buf := make([]byte, *skip)
		i, err := io.ReadFull(os.Stdin, buf)
		if i != *skip {
			fmt.Println("failed to skip leading bytes")
			return
		}
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	var r io.Reader
	files := flag.Args()
	if len(files) == 0 {
		r = os.Stdin
	} else {
		readers := make([]io.Reader, len(files))
		for i := range files {
			f, err := os.Open(files[i])
			if err != nil {
				panic(err)
			}
			readers[i] = f
		}
		r = io.MultiReader(readers...)
	}

	buf, err := ioutil.ReadAll(r)
	if err != nil {
		fmt.Println(err)
		return
	}

	l, err := nv.Pretty(buf)
	if err != nil {
		fmt.Println(err)
		return
	}

	printList("", l)
}
