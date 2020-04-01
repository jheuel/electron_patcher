package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jheuel/asar"
)

type node struct {
	Name     string
	IsDir    bool
	Flag     asar.Flag
	Parent   *node
	Children []*node
	Content  []byte
}

func decode(path string) (*node, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Could not open file: %v", err)
	}
	defer f.Close()

	archive, err := asar.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("Could not decode archive: %v", err)
	}

	return toMemory(archive), nil
}

func toMemory(e *asar.Entry) *node {
	n := &node{}
	n.Name = e.Name
	n.IsDir = e.Flags&asar.FlagDir != 0
	n.Content = e.Bytes()
	n.Flag = e.Flags
	for _, c := range e.Children {
		child := toMemory(c)
		child.Parent = n
		n.Children = append(n.Children, child)
	}
	return n
}

func populate(n *node, entries *asar.Builder) {
	for _, c := range n.Children {
		if c.IsDir {
			e := entries.AddDir(c.Name, asar.FlagDir)
			populate(c, e)
			entries.Parent()
		} else {
			entries.Add(c.Name, bytes.NewReader(c.Content), int64(len(c.Content)), c.Flag)
		}
	}
}

func encodeTo(archive *node, asarFileName string) error {
	asarArchive, err := os.Create(asarFileName)
	if err != nil {
		return fmt.Errorf("could not open file: %v", err)
	}
	defer asarArchive.Close()

	entries := asar.Builder{}

	populate(archive, &entries)
	if _, err := entries.Root().EncodeTo(asarArchive); err != nil {
		return fmt.Errorf("could not create: %s, the error was %v", asarFileName, err)
	}
	return nil
}

func modify(n *node) {
	replaceMap := map[string]string{
		"mainWindow.show();":  "1+1;",
		"mainWindow.focus();": "1+1;",
	}
	if strings.HasSuffix(n.Name, ".js") {
		for k, v := range replaceMap {
			n.Content = []byte(strings.ReplaceAll(string(n.Content), k, v))
		}
	}
	for _, c := range n.Children {
		modify(c)
	}
}

func main() {
	path := strings.ReplaceAll(os.Getenv("APPDATA"), "Roaming", "") + "Local\\Blitz\\current\\resources\\app.asar"
	log.Printf("Patch archive in %v", path)

	archive, err := decode(path)
	if err != nil {
		log.Fatalf("Archive could not be decoded: %v", err)
	}
	modify(archive)
	err = encodeTo(archive, path)
	if err != nil {
		log.Fatalf("Archive could not be encoded: %v", err)
	}
}
