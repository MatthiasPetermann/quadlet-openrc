package generate

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/petermann-digital/quadlet-openrc/internal/quadlet"
)

const Version = "0.3.0"

//go:embed templates/container.tmpl
var containerTemplateText string

//go:embed templates/network.tmpl
var networkTemplateText string

//go:embed templates/volume.tmpl
var volumeTemplateText string

//go:embed templates/image.tmpl
var imageTemplateText string

var (
	containerTemplate = template.Must(template.New("container").Parse(containerTemplateText))
	networkTemplate   = template.Must(template.New("network").Parse(networkTemplateText))
	volumeTemplate    = template.Must(template.New("volume").Parse(volumeTemplateText))
	imageTemplate     = template.Must(template.New("image").Parse(imageTemplateText))
)

type Result struct {
	Service, Content string
	Warnings         []string
}
type Generator struct {
	Podman, SourceDir, ServicePrefix string
	ResourceNames                    map[string]string
}

type containerTemplateData struct {
	Header        string
	Description   string
	Name          string
	Command       string
	CommandArgs   string
	ServiceName   string
	RespawnDelay  string
	RespawnMax    string
	Depend        string
	Podman        string
	Timeout       string
	ContainerName string
}

type resourceTemplateData struct {
	Header      string
	Description string
	Depend      string
	Podman      string
	Name        string
	CreateArgs  string
	PullArgs    string
	Image       string
}

func New() *Generator {
	return &Generator{Podman: "/usr/bin/podman", ServicePrefix: "quadlet-", ResourceNames: map[string]string{}}
}

func ServiceName(q *quadlet.File) string {
	return RefServiceNameWithPrefix(filepath.Base(q.Path), "quadlet-")
}
func RefServiceName(v string) string { return RefServiceNameWithPrefix(v, "quadlet-") }
func RefServiceNameWithPrefix(v, prefix string) string {
	v = filepath.Base(v)
	switch {
	case strings.HasSuffix(v, ".container"):
		return strings.TrimSuffix(v, ".container")
	case strings.HasSuffix(v, ".network"):
		return prefix + "network-" + strings.TrimSuffix(v, ".network")
	case strings.HasSuffix(v, ".volume"):
		return prefix + "volume-" + strings.TrimSuffix(v, ".volume")
	case strings.HasSuffix(v, ".image"):
		return prefix + "image-" + strings.TrimSuffix(v, ".image")
	case strings.HasSuffix(v, ".service"):
		return strings.TrimSuffix(v, ".service")
	default:
		return v
	}
}
func (g *Generator) serviceName(q *quadlet.File) string {
	return RefServiceNameWithPrefix(filepath.Base(q.Path), g.ServicePrefix)
}
func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'" }
func boolValue(v string, d bool) bool {
	if v == "" {
		return d
	}
	b, e := strconv.ParseBool(strings.ToLower(v))
	if e != nil {
		return d
	}
	return b
}
func words(vs []string) []string {
	var o []string
	for _, v := range vs {
		o = append(o, strings.Fields(v)...)
	}
	return o
}
func uniq(xs []string) []string {
	m := map[string]bool{}
	var o []string
	for _, x := range xs {
		if x != "" && !m[x] {
			m[x] = true
			o = append(o, x)
		}
	}
	sort.Strings(o)
	return o
}
func joinQuoted(xs []string) string {
	var o []string
	for _, x := range xs {
		o = append(o, shellQuote(x))
	}
	return strings.Join(o, " ")
}
func hashSources(q *quadlet.File) string {
	h := sha256.New()
	for _, p := range q.Sources {
		d, _ := os.ReadFile(p)
		h.Write(d)
	}
	return hex.EncodeToString(h.Sum(nil)[:8])
}
func description(q *quadlet.File) string {
	if d := q.Value("Unit", "Description"); d != "" {
		return d
	}
	return "Generated from " + filepath.Base(q.Path)
}

func (g *Generator) Register(files []*quadlet.File) {
	if g.ResourceNames == nil {
		g.ResourceNames = map[string]string{}
	}
	for _, q := range files {
		switch q.Kind {
		case "network":
			g.ResourceNames[filepath.Base(q.Path)] = resourceName(q, "Network", "NetworkName", "systemd-")
		case "volume":
			g.ResourceNames[filepath.Base(q.Path)] = resourceName(q, "Volume", "VolumeName", "systemd-")
		case "image":
			name := q.Value("Image", "ImageTag")
			if name == "" {
				name = "localhost/systemd-" + q.Name
			}
			g.ResourceNames[filepath.Base(q.Path)] = name
		}
	}
}
func (g *Generator) translateRef(v, kind string) string {
	suffix := "." + kind
	if kind == "volume" {
		p := strings.SplitN(v, ":", 2)
		if strings.HasSuffix(p[0], suffix) {
			if n := g.ResourceNames[filepath.Base(p[0])]; n != "" {
				p[0] = n
			} else {
				p[0] = "systemd-" + strings.TrimSuffix(filepath.Base(p[0]), suffix)
			}
		}
		return strings.Join(p, ":")
	}
	if strings.HasSuffix(v, suffix) {
		if n := g.ResourceNames[filepath.Base(v)]; n != "" {
			return n
		}
		return "systemd-" + strings.TrimSuffix(filepath.Base(v), suffix)
	}
	return v
}
func (g *Generator) imageValue(v string) string {
	if strings.HasSuffix(v, ".image") {
		if n := g.ResourceNames[filepath.Base(v)]; n != "" {
			return n
		}
		return "localhost/systemd-" + strings.TrimSuffix(filepath.Base(v), ".image")
	}
	return v
}
func (g *Generator) Generate(q *quadlet.File) (Result, error) {
	switch q.Kind {
	case "container":
		return g.container(q)
	case "network":
		return g.network(q)
	case "volume":
		return g.volume(q)
	case "image":
		return g.image(q)
	default:
		return Result{}, fmt.Errorf("unsupported quadlet type .%s", q.Kind)
	}
}

func (g *Generator) dependencies(q *quadlet.File) (need, want, after, before []string, w []string) {
	add := func(dst *[]string, vals []string) {
		for _, v := range words(vals) {
			if strings.HasSuffix(v, ".target") || strings.HasSuffix(v, ".mount") || strings.HasSuffix(v, ".socket") {
				w = append(w, "ignored unsupported systemd dependency: "+v)
				continue
			}
			*dst = append(*dst, RefServiceNameWithPrefix(v, g.ServicePrefix))
		}
	}
	add(&need, q.Values("Unit", "Requires"))
	add(&want, q.Values("Unit", "Wants"))
	add(&after, q.Values("Unit", "After"))
	add(&before, q.Values("Unit", "Before"))
	if q.Kind == "container" {
		for _, v := range q.Values("Container", "Network") {
			if strings.HasSuffix(v, ".network") {
				s := RefServiceNameWithPrefix(v, g.ServicePrefix)
				need = append(need, s)
				after = append(after, s)
			}
		}
		for _, v := range q.Values("Container", "Volume") {
			src := strings.SplitN(v, ":", 2)[0]
			if strings.HasSuffix(src, ".volume") {
				s := RefServiceNameWithPrefix(src, g.ServicePrefix)
				need = append(need, s)
				after = append(after, s)
			}
		}
		if v := q.Value("Container", "Image"); strings.HasSuffix(v, ".image") {
			s := RefServiceNameWithPrefix(v, g.ServicePrefix)
			need = append(need, s)
			after = append(after, s)
		}
	}
	return uniq(need), uniq(want), uniq(after), uniq(before), w
}
func (g *Generator) renderDepend(q *quadlet.File) (string, []string) {
	need, want, after, before, w := g.dependencies(q)
	var b strings.Builder
	b.WriteString("depend() {\n    need podman\n")
	if len(need) > 0 {
		b.WriteString("    need " + strings.Join(need, " ") + "\n")
	}
	if len(want) > 0 {
		b.WriteString("    want " + strings.Join(want, " ") + "\n")
	}
	if len(after) > 0 {
		b.WriteString("    after " + strings.Join(after, " ") + "\n")
	}
	if len(before) > 0 {
		b.WriteString("    before " + strings.Join(before, " ") + "\n")
	}
	b.WriteString("}\n")
	return b.String(), w
}

func resourceName(q *quadlet.File, section, key, defaultPrefix string) string {
	if v := q.Value(section, key); v != "" {
		return v
	}
	return defaultPrefix + q.Name
}
func SecurityWarnings(q *quadlet.File) []string {
	if q.Kind != "container" {
		return nil
	}
	var w []string
	if q.Value("Container", "UserNS") == "" {
		w = append(w, "security: UserNS is not set (consider auto)")
	}
	drops := strings.ToUpper(strings.Join(q.Values("Container", "DropCapability"), ","))
	if !strings.Contains(drops, "ALL") {
		w = append(w, "security: capabilities are not dropped with DropCapability=ALL")
	}
	if !boolValue(q.Value("Container", "NoNewPrivileges"), false) {
		w = append(w, "security: NoNewPrivileges=true is not set")
	}
	if !boolValue(q.Value("Container", "ReadOnly"), false) {
		w = append(w, "security: ReadOnly=true is not set")
	}
	if len(q.Values("Container", "Tmpfs")) == 0 {
		w = append(w, "security: no Tmpfs= configured for writable runtime paths")
	}
	if q.Value("Container", "PidsLimit") == "" {
		w = append(w, "security: PidsLimit is not set")
	}
	if q.Value("Container", "Memory") == "" {
		w = append(w, "security: Memory limit is not set")
	}
	if q.Value("Container", "CPUs") == "" {
		w = append(w, "security: CPUs limit is not set")
	}
	if strings.EqualFold(q.Value("Container", "Privileged"), "true") {
		w = append(w, "security: Privileged=true disables major isolation boundaries")
	}
	return w
}

func (g *Generator) header(q *quadlet.File) string {
	return fmt.Sprintf("#!/sbin/openrc-run\n# Generated by quadlet-openrc %s from %s\n# source-hash: %s\n# DO NOT EDIT: change the Quadlet source and regenerate.\n\n", Version, filepath.Base(q.Path), hashSources(q))
}

func renderTemplate(t *template.Template, data any) (string, error) {
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return "", err
	}
	return b.String(), nil
}

func (g *Generator) container(q *quadlet.File) (Result, error) {
	image := q.Value("Container", "Image")
	if image == "" {
		return Result{}, fmt.Errorf("%s: [Container] Image= is required", q.Path)
	}
	image = g.imageValue(image)
	name := q.Value("Container", "ContainerName")
	if name == "" {
		name = q.Name
	}
	args := []string{"run", "--rm", "--replace", "--name", name}
	scalar := func(k, o string) {
		if v := q.Value("Container", k); v != "" {
			args = append(args, o, v)
		}
	}
	repeated := func(k, o string) {
		for _, v := range q.Values("Container", k) {
			args = append(args, o, v)
		}
	}
	scalar("Pull", "--pull")
	if boolValue(q.Value("Container", "ReadOnly"), false) {
		args = append(args, "--read-only")
	}
	if boolValue(q.Value("Container", "RunInit"), false) {
		args = append(args, "--init")
	}
	if boolValue(q.Value("Container", "Privileged"), false) {
		args = append(args, "--privileged")
	}
	repeated("Environment", "--env")
	repeated("EnvironmentFile", "--env-file")
	for _, v := range q.Values("Container", "Volume") {
		args = append(args, "--volume", g.translateRef(v, "volume"))
	}
	repeated("PublishPort", "--publish")
	repeated("ExposeHostPort", "--expose")
	repeated("Label", "--label")
	repeated("AddCapability", "--cap-add")
	repeated("DropCapability", "--cap-drop")
	for _, v := range q.Values("Container", "Network") {
		args = append(args, "--network", g.translateRef(v, "network"))
	}
	repeated("Tmpfs", "--tmpfs")
	repeated("Secret", "--secret")
	repeated("Device", "--device")
	scalar("User", "--user")
	scalar("UserNS", "--userns")
	scalar("WorkingDir", "--workdir")
	scalar("HostName", "--hostname")
	scalar("PidsLimit", "--pids-limit")
	scalar("Memory", "--memory")
	scalar("CPUs", "--cpus")
	scalar("HealthCmd", "--health-cmd")
	scalar("HealthInterval", "--health-interval")
	scalar("HealthTimeout", "--health-timeout")
	scalar("HealthRetries", "--health-retries")
	if boolValue(q.Value("Container", "NoNewPrivileges"), false) {
		args = append(args, "--security-opt", "no-new-privileges")
	}
	for _, v := range q.Values("Container", "PodmanArgs") {
		args = append(args, strings.Fields(v)...)
	}
	args = append(args, image)
	if ex := q.Value("Container", "Exec"); ex != "" {
		args = append(args, strings.Fields(ex)...)
	}
	dep, w := g.renderDepend(q)
	for k := range q.Sections["Service"] {
		switch k {
		case "Restart", "RestartSec", "TimeoutStopSec":
		default:
			w = append(w, "[Service] "+k+" is systemd-specific and ignored")
		}
	}
	delay := q.Value("Service", "RestartSec")
	if delay == "" {
		delay = "2"
	}
	timeout := q.Value("Service", "TimeoutStopSec")
	if timeout == "" {
		timeout = "10"
	}
	respawn := "0"
	if strings.EqualFold(q.Value("Service", "Restart"), "no") {
		respawn = "1"
	}
	content, err := renderTemplate(containerTemplate, containerTemplateData{
		Header:        g.header(q),
		Description:   shellQuote(description(q)),
		Name:          shellQuote(name),
		Command:       shellQuote(g.Podman),
		CommandArgs:   shellQuote(strings.Join(args, " ")),
		ServiceName:   name,
		RespawnDelay:  delay,
		RespawnMax:    respawn,
		Depend:        dep,
		Podman:        g.Podman,
		Timeout:       shellQuote(timeout),
		ContainerName: shellQuote(name),
	})
	if err != nil {
		return Result{}, err
	}
	return Result{g.serviceName(q), content, uniq(w)}, nil
}
func (g *Generator) network(q *quadlet.File) (Result, error) {
	name := resourceName(q, "Network", "NetworkName", "systemd-")
	create := []string{"network", "create"}
	if v := q.Value("Network", "Driver"); v != "" {
		create = append(create, "--driver", v)
	}
	for _, k := range []struct{ key, opt string }{{"Subnet", "--subnet"}, {"Gateway", "--gateway"}, {"Label", "--label"}, {"Options", "--opt"}} {
		for _, v := range q.Values("Network", k.key) {
			create = append(create, k.opt, v)
		}
	}
	create = append(create, name)
	dep, w := g.renderDepend(q)
	content, err := renderTemplate(networkTemplate, resourceTemplateData{
		Header:      g.header(q),
		Description: shellQuote(description(q)),
		Depend:      dep,
		Podman:      g.Podman,
		Name:        shellQuote(name),
		CreateArgs:  joinQuoted(create),
	})
	if err != nil {
		return Result{}, err
	}
	return Result{g.serviceName(q), content, w}, nil
}
func (g *Generator) volume(q *quadlet.File) (Result, error) {
	name := resourceName(q, "Volume", "VolumeName", "systemd-")
	create := []string{"volume", "create"}
	if v := q.Value("Volume", "Driver"); v != "" {
		create = append(create, "--driver", v)
	}
	for _, k := range []struct{ key, opt string }{{"Label", "--label"}, {"Options", "--opt"}} {
		for _, v := range q.Values("Volume", k.key) {
			create = append(create, k.opt, v)
		}
	}
	create = append(create, name)
	dep, w := g.renderDepend(q)
	content, err := renderTemplate(volumeTemplate, resourceTemplateData{
		Header:      g.header(q),
		Description: shellQuote(description(q)),
		Depend:      dep,
		Podman:      g.Podman,
		Name:        shellQuote(name),
		CreateArgs:  joinQuoted(create),
	})
	if err != nil {
		return Result{}, err
	}
	return Result{g.serviceName(q), content, w}, nil
}
func (g *Generator) image(q *quadlet.File) (Result, error) {
	image := q.Value("Image", "Image")
	if image == "" {
		return Result{}, fmt.Errorf("%s: [Image] Image= is required", q.Path)
	}
	name := q.Value("Image", "ImageTag")
	if name == "" {
		name = "localhost/systemd-" + q.Name
	}
	dep, w := g.renderDepend(q)
	pull := []string{"pull", image}
	content, err := renderTemplate(imageTemplate, resourceTemplateData{
		Header:      g.header(q),
		Description: shellQuote(description(q)),
		Depend:      dep,
		Podman:      g.Podman,
		Name:        shellQuote(name),
		PullArgs:    joinQuoted(pull),
		Image:       shellQuote(image),
	})
	if err != nil {
		return Result{}, err
	}
	return Result{g.serviceName(q), content, w}, nil
}
