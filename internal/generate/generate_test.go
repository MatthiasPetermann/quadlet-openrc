package generate

import (
	"github.com/petermann-digital/quadlet-openrc/internal/quadlet"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func parse(t *testing.T, name, body string) *quadlet.File {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	q, e := quadlet.Parse(p)
	if e != nil {
		t.Fatal(e)
	}
	return q
}
func TestImplicitDependenciesAndStableNaming(t *testing.T) {
	q := parse(t, "web.container", "[Unit]\nWants=metrics.container\nBefore=proxy.container\n[Container]\nImage=nginx\nNetwork=front.network\nVolume=data.volume:/data\n")
	r, e := New().Generate(q)
	if e != nil {
		t.Fatal(e)
	}
	for _, s := range []string{"need quadlet-network-front quadlet-volume-data", "want metrics", "after quadlet-network-front quadlet-volume-data", "before proxy", "systemd-front", "systemd-data:/data"} {
		if !strings.Contains(r.Content, s) {
			t.Fatalf("missing %q in:\n%s", s, r.Content)
		}
	}
}
func TestResourceNamesDoNotCollide(t *testing.T) {
	n := parse(t, "app.network", "[Network]\n")
	v := parse(t, "app.volume", "[Volume]\n")
	i := parse(t, "app.image", "[Image]\nImage=alpine\n")
	if ServiceName(n) == ServiceName(v) || ServiceName(v) == ServiceName(i) {
		t.Fatal("resource service names collide")
	}
}
func TestImageDependency(t *testing.T) {
	q := parse(t, "web.container", "[Container]\nImage=base.image\n")
	r, e := New().Generate(q)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(r.Content, "need quadlet-image-base") || !strings.Contains(r.Content, "localhost/systemd-base") {
		t.Fatal(r.Content)
	}
}
func TestSecurityLint(t *testing.T) {
	q := parse(t, "x.container", "[Container]\nImage=alpine\n")
	if len(SecurityWarnings(q)) < 5 {
		t.Fatal("expected security warnings")
	}
}
func TestMissingDependency(t *testing.T) {
	q := parse(t, "x.container", "[Unit]\nRequires=nope.network\n[Container]\nImage=alpine\n")
	if len(ValidateGraph([]*quadlet.File{q})) == 0 {
		t.Fatal("missing dependency not detected")
	}
}
func TestCycle(t *testing.T) {
	a := parse(t, "a.container", "[Unit]\nAfter=b.container\n[Container]\nImage=alpine\n")
	b := parse(t, "b.container", "[Unit]\nAfter=a.container\n[Container]\nImage=alpine\n")
	if len(ValidateGraph([]*quadlet.File{a, b})) == 0 {
		t.Fatal("cycle not detected")
	}
}
