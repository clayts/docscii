package docBook

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/clayts/docscii/file"
	"github.com/clayts/docscii/xmlTree"
)

func findBetween(s, a, b string) string {
	aSplit := strings.SplitN(s, a, 2)
	if len(aSplit) == 2 {
		bSplit := strings.SplitN(aSplit[1], b, 2)
		if len(bSplit) == 2 {
			return bSplit[0]
		}
	}
	return ""
}

func FindDocRoot(dir string) string {
	var filename string
	fs, err := ioutil.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, f := range fs {
		if filepath.Ext(f.Name()) == ".xml" {
			b, err2 := ioutil.ReadFile(dir + "/" + f.Name())
			if err2 != nil {
				continue
			}
			s := string(b)
			for _, m := range []string{"</book>", "</article>"} {
				if strings.Contains(s, m) {
					filename = dir + "/" + f.Name()
					break
				}
			}
			if filename != "" {
				break
			}
		}
	}
	return filepath.Clean(filename)
}

func ConditionsMatch(docConditions, chunkCondition string) bool {
	if chunkCondition == "" {
		return true
	}
	for _, d := range strings.Split(docConditions, ";") {
		d = strings.TrimSpace(d)
		for _, c := range strings.Split(chunkCondition, ";") {
			c = strings.TrimSpace(c)
			if c == d {
				return true
			}
		}
	}
	return false
}

type Doc struct {
	PublicanCfg map[string]string
	Resources   map[string]string
	Data        xmlTree.Chunks
}

func New() *Doc {
	d := &Doc{}
	d.Resources = make(map[string]string)
	return d
}

func NewFromDir(dir string) *Doc {
	return NewFromFile(FindDocRoot(dir))
}

func NewFromFile(filename string) *Doc {
	d := New()
	d.loadData(filename)
	if len(d.Data) == 0 {
		return nil
	}
	return d
}

func (d *Doc) loadData(filename string) {
	directory := filepath.Dir(filename)
	d.Data = xmlTree.New(file.Read(filename))

	entityFiles := make(map[string]struct{})
	var process func(dir string, cs xmlTree.Chunks)
	process = func(dir string, cs xmlTree.Chunks) {
		for _, c := range cs {
			if d.PublicanCfg == nil || ConditionsMatch(d.PublicanCfg["condition"], c.Attributes["condition"]) {
				switch {
				case c.IsKind("DIRECTIVE"):
					if f := findBetween(c.Attributes["DIRECTIVE"], "<!ENTITY % BOOK_ENTITIES SYSTEM \"", "\">"); f != "" {
						f = filepath.Clean(dir + "/" + f)
						chs := c.Children
						if _, ok := entityFiles[f]; !ok {
							entityFiles[f] = struct{}{}
							n := xmlTree.New(file.Read(f))
							c.AddChildren(n)
							process(filepath.Dir(f), n)
						}
						process(dir, chs)
					} else if k := findBetween(c.Attributes["DIRECTIVE"], "ENTITY ", " \""); strings.HasPrefix(c.Attributes["DIRECTIVE"], "ENTITY ") && k != "" {
						if v := findBetween(c.Attributes["DIRECTIVE"], " \"", "\""); v != "" {
							c.Kind = "ENTITY"
							c.Attributes["KEY"] = k
							c.AddChildren(xmlTree.New(v))
						}
						process(dir, c.Children)
					} else {
						process(dir, c.Children)
					}
				case c.IsKind("imagedata"):
					if href, ok := c.Attributes["fileref"]; ok {
						var src string
						if d.PublicanBrandDir() != "" && strings.HasPrefix(href, "Common_Content/") {
							src = d.PublicanBrandDir() + strings.Replace(href, "Common_Content/", "", -1)
						} else {
							src = dir + "/" + href
						}
						c.Attributes["DIR"], _ = filepath.Rel(directory, dir)
						d.Resources[filepath.Clean(c.Attributes["DIR"]+"/"+href)] = src
					}
				case c.IsKind("include"):
					if href, ok := c.Attributes["href"]; ok {
						var fname string
						if d.PublicanBrandDir() != "" && strings.HasPrefix(href, "Common_Content/") {
							fname = d.PublicanBrandDir() + strings.Replace(href, "Common_Content/", "", -1)
						} else {
							fname = filepath.Clean(dir + "/" + href)
						}

						if file.Exists(fname) {
							chs := c.Children
							if c.Attributes["parse"] == "text" {
								c.Attributes["DIR"], _ = filepath.Rel(directory, dir)
								d.Resources[filepath.Clean(c.Attributes["DIR"]+"/"+href)] = fname
							} else {
								newData := xmlTree.New(file.Read(fname))
								c.AddChildren(newData)
								process(filepath.Dir(fname), newData)
							}
							process(dir, chs.FilterOut("fallback"))
						} else {
							process(dir, c.Children)
						}
					}
				default:
					process(dir, c.Children)
				}
			}
		}
	}
	process(directory, d.Data)
}

func NewFromPublicanCfg(filename string) *Doc {
	publicanCfg := make(map[string]string)
	s := file.Read(filename)
	for _, l := range strings.Split(s, "\n") {
		l = strings.Replace(l, "\"", "", -1)
		lSplit := strings.SplitN(l, ":", 2)
		if len(lSplit) == 2 {
			publicanCfg[strings.TrimSpace(lSplit[0])] = strings.TrimSpace(lSplit[1])
		}
	}

	if _, ok := publicanCfg["xml_lang"]; !ok {
		return nil
	}
	d := New()
	d.PublicanCfg = publicanCfg
	d.loadData(FindDocRoot(filepath.Dir(filename) + "/" + publicanCfg["xml_lang"]))
	return d
}

func (d Doc) PublicanBrandDir() string {
	if d.PublicanCfg == nil {
		return ""
	}
	return "/usr/share/publican/Common_Content/" + d.PublicanCfg["brand"] + "/" + d.PublicanCfg["xml_lang"] + "/"
}
