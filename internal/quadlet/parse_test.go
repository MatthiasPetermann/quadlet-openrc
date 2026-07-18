package quadlet

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseRepeated(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "x.container")
	os.WriteFile(p, []byte("[Container]\nImage=alpine\nEnvironment=A=1\nEnvironment=B=2\n"), 0644)
	q, e := Parse(p)
	if e != nil {
		t.Fatal(e)
	}
	if len(q.Values("Container", "Environment")) != 2 {
		t.Fatal("repeated values lost")
	}
}
func TestDropInOverrideAndReset(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "x.container")
	os.WriteFile(p, []byte("[Container]\nImage=alpine:old\nEnvironment=A=1\n"), 0644)
	os.Mkdir(p+".d", 0755)
	os.WriteFile(filepath.Join(p+".d", "10-override.conf"), []byte("[Container]\nImage=alpine:new\nEnvironment=\nEnvironment=B=2\n"), 0644)
	q, e := Parse(p)
	if e != nil {
		t.Fatal(e)
	}
	if q.Value("Container", "Image") != "alpine:new" {
		t.Fatal(q.Value("Container", "Image"))
	}
	vs := q.Values("Container", "Environment")
	if len(vs) != 1 || vs[0] != "B=2" {
		t.Fatalf("%v", vs)
	}
}
