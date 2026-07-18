package quadlet

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type File struct {
	Path     string
	Name     string
	Kind     string
	Sections map[string]map[string][]string
	Sources  []string
}

func Parse(path string) (*File, error) {
	q := &File{Path: path, Name: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), Kind: strings.TrimPrefix(filepath.Ext(path), "."), Sections: map[string]map[string][]string{}}
	if err := q.merge(path); err != nil {
		return nil, err
	}
	// Stable drop-in API: foo.type.d/*.conf, lexical order, later values append;
	// scalar consumers use the last value, repeated consumers retain all values.
	dropin := path + ".d"
	entries, err := os.ReadDir(dropin)
	if err == nil {
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".conf") {
				continue
			}
			if err := q.merge(filepath.Join(dropin, e.Name())); err != nil {
				return nil, err
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return q, nil
}

func (q *File) merge(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	q.Sources = append(q.Sources, path)
	section := ""
	s := bufio.NewScanner(f)
	pending := ""
	for lineNo := 1; s.Scan(); lineNo++ {
		line := strings.TrimSpace(s.Text())
		if pending != "" {
			line = pending + line
			pending = ""
		}
		if strings.HasSuffix(line, "\\") {
			pending = strings.TrimSuffix(line, "\\")
			continue
		}
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			if q.Sections[section] == nil {
				q.Sections[section] = map[string][]string{}
			}
			continue
		}
		if section == "" {
			return fmt.Errorf("%s:%d: key outside section", path, lineNo)
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			return fmt.Errorf("%s:%d: expected key=value", path, lineNo)
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		// Empty assignment resets a repeated option, matching systemd-style drop-ins.
		if v == "" {
			q.Sections[section][k] = nil
		} else {
			q.Sections[section][k] = append(q.Sections[section][k], v)
		}
	}
	return s.Err()
}
func (f *File) Values(section, key string) []string {
	return append([]string(nil), f.Sections[section][key]...)
}
func (f *File) Value(section, key string) string {
	vs := f.Values(section, key)
	if len(vs) == 0 {
		return ""
	}
	return vs[len(vs)-1]
}
