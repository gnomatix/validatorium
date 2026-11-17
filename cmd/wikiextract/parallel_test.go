package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func pageXML(id int, title, text string) string {
	return fmt.Sprintf("<page><title>%s</title><ns>0</ns><id>%d</id><revision><id>%d</id><text>%s</text></revision></page>\n", title, id, id*10, text)
}

// TestParallelMatchesSerial builds a three-stream multistream dump (header,
// a stream of pages, a final stream ending in </mediawiki>) with the bzip2
// command and checks that the parallel path writes exactly what the serial
// path writes.
func TestParallelMatchesSerial(t *testing.T) {
	bz, err := exec.LookPath("bzip2")
	if err != nil {
		t.Skip("bzip2 command not available")
	}
	long := strings.Repeat("Hydrogen is the lightest element. ", 10)
	parts := []string{
		"<mediawiki><siteinfo><sitename>Test</sitename></siteinfo>\n",
		pageXML(1, "Alpha", long) + pageXML(2, "Beta", long+"<ref>{{cite journal |doi=10.1/x |title=T}}</ref>"),
		pageXML(3, "Gamma", long) + "</mediawiki>\n",
	}
	dir := t.TempDir()
	var dump bytes.Buffer
	var index strings.Builder
	for i, p := range parts {
		cmd := exec.Command(bz, "-c")
		cmd.Stdin = strings.NewReader(p)
		b, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		if i > 0 {
			for _, title := range []string{"Alpha", "Beta", "Gamma"} {
				if strings.Contains(p, "<title>"+title+"<") {
					fmt.Fprintf(&index, "%d:%d:%s\n", dump.Len(), i, title)
				}
			}
		}
		dump.Write(b)
	}
	dumpPath := filepath.Join(dir, "d.xml.bz2")
	indexPath := filepath.Join(dir, "i.txt")
	if err := os.WriteFile(dumpPath, dump.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(indexPath, []byte(index.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	var serial, par bytes.Buffer
	ns, err := run(dumpPath, "testwiki", 0, 10, &serial)
	if err != nil {
		t.Fatal(err)
	}
	np, err := runParallel(dumpPath, indexPath, "testwiki", 0, 10, 3, &par)
	if err != nil {
		t.Fatal(err)
	}
	if ns != 3 || np != 3 {
		t.Fatalf("articles: serial %d, parallel %d, want 3", ns, np)
	}
	if serial.String() != par.String() {
		t.Fatalf("outputs differ:\nserial:   %s\nparallel: %s", serial.String(), par.String())
	}
	var one bytes.Buffer
	if n, err := runParallel(dumpPath, indexPath, "testwiki", 1, 10, 2, &one); err != nil || n != 1 || !strings.Contains(one.String(), `"Alpha"`) {
		t.Fatalf("max 1: n=%d err=%v out=%s", n, err, one.String())
	}
}
