package asciiDoc

import (
	"github.com/clayts/docscii/file"
	"github.com/clayts/docscii/xmlTree"
	"sort"
	"strconv"
	"strings"
)

type Doc struct {
	Keywords  map[string]struct{}
	Entities  map[string]string
	Data      map[string]string
	Resources map[string]string
	Metadata  xmlTree.Chunks
}

func (d *Doc) Create(title, data string) string {
	var count int
	name := func() string {
		if count == 0 {
			return title + ".adoc"
		}
		return title + "-" + strconv.Itoa(count) + ".adoc"
	}
	for {
		if d, ok := d.Data[name()]; ok {
			if d == data {
				return name()
			}
			count++
		} else {
			break
		}
	}
	d.Data[name()] = data
	return name()
}

func New() *Doc {
	d := &Doc{}
	d.Keywords = make(map[string]struct{})
	d.Entities = make(map[string]string)
	d.Entities["nbsp"] = ""
	d.Entities["blank"] = ""
	d.Data = make(map[string]string)
	d.Resources = make(map[string]string)
	return d
}

func (d Doc) Write(dir string) {
	if d.Data["master.adoc"] == "" {
		return
	}

	for dst, src := range d.Resources {
		file.Copy(src, dir+"/"+dst)
	}

	if len(d.Metadata) > 0 {
		var ms []string
		for _, m := range d.Metadata {
			ms = append(ms, m.XML())
		}
		sort.Strings(ms)
		file.Write(dir+"/master-docinfo.xml", strings.Join(ms, "\n"))
	}

	if len(d.Entities) > 0 {
		var es []string
		d.Data["master.adoc"] = "\n\ninclude::entities.adoc[]\n\n" + d.Data["master.adoc"]
		for k, v := range d.Entities {
			if k != "" && v != "" {
				es = append(es, "\n:"+k+": "+v)
			}
		}
		sort.Strings(es)
		file.Write(dir+"/entities.adoc", strings.Replace(strings.Join(es, "\n"), "&nbsp;", " ", -1))
	}

	if len(d.Keywords) > 0 {
		var ks []string
		for k := range d.Keywords {
			ks = append(ks, k)
		}
		sort.Strings(ks)
		d.Data["master.adoc"] = ":keywords: " + strings.Join(ks, ", ") + d.Data["master.adoc"]
	}

	d.Data["master.adoc"] = ":doctype: book\n" + d.Data["master.adoc"]
	d.Data["master.adoc"] = ":experimental:\n" + d.Data["master.adoc"]
	for f, d := range d.Data {
		file.Write(dir+"/"+f, d)
	}
}
