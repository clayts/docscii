package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/clayts/docscii/asciiDoc"
	"github.com/clayts/docscii/docBook"
	"github.com/clayts/docscii/file"
	"github.com/clayts/docscii/xmlTree"

	"github.com/fatih/color"
)

func decorateIfNotBlank(s, pre, suf string) string {
	if s != "" {
		return pre + s + suf
	}
	return ""
}

func spaceTrimmings(input string) (string, string) {
	tlc := strings.TrimLeft(input, " ")
	var ls string
	for x := 0; x < len(input)-len(tlc); x++ {
		ls += " "
	}
	trc := strings.TrimRight(input, " ")
	var rs string
	for x := 0; x < len(input)-len(trc); x++ {
		rs += " "
	}
	return ls, rs
}

func quoteSafe(cs xmlTree.Chunks) bool {
	for _, c := range cs {
		if c.Children.Flatten().Contains("include") {
			return false
		}
		for _, b := range "*^#`_+" {
			if strings.Contains(c.Children.Flatten().Filter("TEXT").XML(), string(b)) {
				return false
			}
		}
	}
	return true
}

func bypassBrokenInclusions(cs xmlTree.Chunks) {
	for _, c := range cs {
		if c.IsKind("title", "indexterm") && c.Parent.IsKind("include") {
			var newSibs xmlTree.Chunks
			for _, sib := range c.Parent.Children {
				if sib != c {
					newSibs = append(newSibs, sib)
				}
			}
			c.Parent.Children = newSibs
			c.Parent.Parent.AddChild(c)
		}
		bypassBrokenInclusions(c.Children)
	}
}

func AsciiDocFromDocBook(db *docBook.Doc, Styles ...Style) *asciiDoc.Doc {
	cfg := NewStyle()
	cfg.OverrideWith(DefaultStyle)
	for _, s := range Styles {
		cfg.OverrideWith(s)
	}
	ad := asciiDoc.New()
	data := db.Data.Copy()
	register := make(map[*xmlTree.Chunk]struct{})
	for _, t := range data.Flatten().Filter("TEXT") {
		register[t] = struct{}{}
	}
	var translate func(xmlTree.Chunks) string
	decorateTitle := func(c *xmlTree.Chunk, prefix string) string {
		output := "\n\n"
		if id, ok := c.Attributes["id"]; ok {
			output += "[[" + id + "]]\n"
		}
		title := c.Children.Filter("title")
		output += decorateIfNotBlank(translate(title), prefix, "")
		return output
	}
	quote := func(c *xmlTree.Chunk, quoter string) string {
		var output string
		contents := translate(c.Children)
		ls, rs := spaceTrimmings(contents)
		con := strings.TrimSpace(contents)
		if con != "" {
			literal := c.IsWithin(cfg["literal"]...)
			block := c.IsWithin("screen", "synopsis", "programlisting")
			safe := block && quoteSafe(c.Ancestors().Filter("screen", "synopsis", "programlisting"))
			esc := block && (!c.IsKind(cfg["custom"]...) && !c.IsWithin(cfg.allQuotes()...) && !c.IsWithin(cfg.unQuotedCustom()...) && !strings.Contains(con, "]"))
			if !literal || safe {
				var tag string
				if c.IsKind(cfg["custom"]...) {
					tag = "[" + c.Kind + "]"
				}
				output += ls + "pass:attributes[{blank}]" + tag + quoter + con + quoter + "pass:attributes[{blank}]" + rs
			} else if esc {
				pass := "quotes"
				for e := range ad.Entities {
					if strings.Contains(con, e) {
						pass += ",attributes"
						break
					}
				}
				output += ls + "pass:" + pass + "[" + quoter + con + quoter + "]" + rs
			} else {
				output += ls + con + rs
			}
		}
		return output
	}
	bypassBrokenInclusions(data)
	translate = func(cs xmlTree.Chunks) string {
		var output string
		for _, c := range cs {
			if c == nil {
				continue
			}
			if docBook.ConditionsMatch(db.PublicanCfg["condition"], c.Attributes["condition"]) {
				switch {
				case c.IsKind("ENTITY"):
					contents := translate(c.Children)
					for k, v := range ad.Entities {
						contents = strings.Replace(contents, "&"+k+";", v, -1)
					}
					for k, v := range ad.Entities {
						ad.Entities[k] = strings.Replace(v, "&"+c.Attributes["KEY"]+";", contents, -1)
					}
					ad.Entities[c.Attributes["KEY"]] = contents
					//output += translate(c.Children)
				case c.IsKind("TEXT"):
					delete(register, c)
					output += c.Attributes["TEXT"]
				case c.IsKind("variablelist", "itemizedlist", "bibliolist", "figure", "table"):
					output += decorateIfNotBlank(decorateTitle(c, "."), "", "\n")
					output += translate(c.Children.FilterOut("TEXT", "title"))
				case c.IsKind("ulink"):
					url := c.Attributes["url"]
					if cc := c.Children.First(cfg["custom"]...); cc != nil {
						text := translate(c.Children.Flatten().Filter("TEXT"))
						fake := &xmlTree.Chunk{}
						fake.Kind = cc.Kind
						ftext := &xmlTree.Chunk{}
						ftext.Kind = "TEXT"
						ftext.Attributes = make(map[string]string)
						ftext.Attributes["TEXT"] = "link:++" + url + "++[" + text + "]"
						fake.Children = append(fake.Children, ftext)
						output += translate(xmlTree.Chunks{fake})
					} else {
						output += "link:++" + url + "++[" + translate(c.Children) + "]"
					}
				case c.IsKind("xref", "link"):
					link := c.Attributes["linkend"]
					output += "<<" + link + decorateIfNotBlank(translate(c.Children), ",", "") + ">>"
				case c.IsKind("screen", "synopsis", "programlisting"):
					if c.Parent.IsKind(cfg["paragraphs"]...) {
						output += "\n"
					}
					var subs []string

					var escapeLtGt bool
					if quoteSafe(xmlTree.Chunks{c}) {
						subs = append(subs, "quotes")
						escapeLtGt = true
					}
					children := translate(c.Children)
					if strings.Contains(children, "pass:") || c.Children.Flatten().Contains("ulink") {
						subs = append(subs, "macros")
						escapeLtGt = true
					}
					if escapeLtGt {
						children = strings.Replace(children, "<", "&lt;", -1)
						children = strings.Replace(children, ">", "&gt;", -1)
					}
					for e := range ad.Entities {
						if strings.Contains(children, "&"+e+";") {
							subs = append(subs, "attributes")
							break
						}
					}
					if len(subs) > 0 {
						output += "\n[subs=\"" + strings.Join(subs, ", ") + "\"]"
					}

					output += "\n----\n" + children + "\n----\n"
				case c.IsKind(cfg["paragraphs"]...):
					var children string
					for _, child := range c.Children {
						if !child.IsKind(cfg["literal"]...) {
							text := translate(xmlTree.Chunks{child})
							for strings.Contains(text, "  ") {
								text = strings.Replace(text, "  ", " ", -1)
							}
							text = strings.Replace(text, "\t", "", -1)
							text = strings.Replace(text, "\n ", "\n", -1)
							children += text
						} else {
							children += translate(xmlTree.Chunks{child})
						}
					}
					output += "\n" + strings.TrimSpace(children) + "\n"
				case c.IsKind("abstract"):
					output += decorateTitle(c, ".")
					output += "\n[abstract]\n--\n" + translate(c.Children.FilterOut("TEXT", "title")) + "\n--\n"
				case c.IsKind("imagedata"):
					if href, ok := c.Attributes["fileref"]; ok {
						ad.Resources[filepath.Clean(c.Attributes["DIR"]+"/"+href)] = db.Resources[filepath.Clean(c.Attributes["DIR"]+"/"+href)]
						output += href
					}
				case c.IsKind("mediaobject"):
					output += "\nimage::" + translate(c.Children.Filter("imageobject")) + "[" + translate(c.Children.Filter("textobject")) + "]\n"
				case c.IsKind("inlinemediaobject"):
					output += "\nimage:" + translate(c.Children.Filter("imageobject")) + "[" + translate(c.Children.Filter("textobject")) + "]"
				case c.IsKind("tgroup"):
					head := translate(c.Children.Filter("thead"))
					var options []string
					if head != "" {
						options = append(options, "header")
					}
					if len(options) > 0 {
						output += "\n[options=\"" + strings.Join(options, ",") + "\"]"
					}
					foot := translate(c.Children.Filter("tfoot"))

					output += "\n|===" + head + translate(c.Children.FilterOut("TEXT", "thead", "tfoot")) + foot + "\n|===\n"
				case c.IsKind("row"):
					output += "\n" + translate(c.Children.FilterOut("TEXT"))
				case c.IsKind("entry"):
					var maxLen int
					if len(c.Parent.Children.Filter("entry")) == 1 {
						tgroup := c.Ancestors().Last("tgroup")
						if tgroup == nil {
							panic("entry outside tgroup")
						}

						for _, child := range tgroup.Children.Flatten().Filter("row") {
							length := len(child.Children.Filter("entry"))
							if length > maxLen {
								maxLen = length
							}
						}
						output += strconv.Itoa(maxLen) + "+"
					}

					output += "|" + strings.TrimSpace(translate(c.Children))
				case c.IsKind("footnote"):
					output += "footnote:[" + strings.TrimSpace(translate(c.Children)) + "]"
				case c.IsKind("bookinfo", "articleinfo"):
					output += decorateIfNotBlank(strings.TrimSpace(translate(c.Children.Filter("title"))), "= ", "")

					meta := c.Children.Filter("productname", "productnumber", "subtitle", "abstract", "edition", "pubsnumber")
					ad.Metadata = append(ad.Metadata, meta...)
					for _, ch := range meta.Flatten() {
						delete(register, ch)
					}

					output += "\n" + translate(c.Children.FilterOut("title", "TEXT", "productname", "productnumber", "edition", "pubsnumber"))
				case c.IsKind("bridgehead"):
					output += "\n." + strings.TrimSpace(translate(c.Children))
				case c.IsKind("chapter", "section", "part", "appendix", "preface"):
					titleDecor := "=="
					for _, ancestor := range c.Ancestors() {
						if ancestor.Children.Contains("title") {
							titleDecor += "="
						}
					}
					if len(titleDecor) > 6 {
						titleDecor = "."
					} else {
						titleDecor += " "
					}
					output += decorateTitle(c, titleDecor)
					output += "\n" + translate(c.Children.FilterOut("title", "TEXT"))
				case c.IsKind("include"):
					if href, ok := c.Attributes["href"]; ok {
						decor := "\n"
						if c.IsWithin(cfg["listitems"]...) && !c.IsWithin(cfg["literal"]...) {
							decor = "\n--\n"
						}
						if c.IsWithin("mediaobject", "inlinemediaobject") {
							output += translate(c.Children)
						} else {
							if c.Attributes["parse"] == "text" {
								if d, ok := db.Resources[filepath.Clean(c.Attributes["DIR"]+"/"+href)]; ok {
									ad.Resources[filepath.Clean(c.Attributes["DIR"]+"/"+href)] = d
									output += decor + "include::" + href + "[]" + decor
								} else {
									output += translate(c.Children.Filter("fallback"))
								}
							} else {
								newData := translate(c.Children.FilterOut("fallback", "TEXT"))
								if newData != "" {
									output += decor + "include::" + ad.Create(file.StripExt(href), newData) + "[]" + decor
								} else {
									output += translate(c.Children.Filter("fallback"))
								}
							}
						}
					}
				case c.IsKind("procedure", "formalpara"):
					output += decorateTitle(c, ".")
					output += translate(c.Children.FilterOut("TEXT", "title"))
				case c.IsKind("varlistentry"):
					output += translate(c.Children.FilterOut("TEXT", "term"))
				case c.IsKind(cfg["admonitions"]...) || c.IsKind("example"):
					decor := "\n===="
					for range c.Ancestors().Filter("example") {
						decor += "="
					}
					decor += "\n"
					output += decorateTitle(c, ".")
					if c.IsKind(cfg["admonitions"]...) {
						output += "\n[" + strings.ToUpper(c.Kind) + "]"
					}
					output += decor + translate(c.Children.FilterOut("TEXT").FilterOut("title")) + decor
				case c.IsKind("corpauthor", "pubdate", "biblioid"):
					var pre, suf string
					if c.IsWithin("biblioentry") {
						pre = ", "
					} else if c.IsWithin("authorgroup") {
						pre = "\n."
						suf = "\n&blank;\n\n"
					}
					output += decorateIfNotBlank(strings.TrimSpace(translate(c.Children)), pre, suf)
				case c.IsKind(cfg["listitems"]...):
					var bullet string
					switch c.Parent.Kind {
					case "itemizedlist", "varlistentry", "bibliolist", "simplelist", "author":
						bullet = "*"
					default:
						bullet = "."
					}
					var children string
					if c.IsKind("member", "contrib") {
						children = strings.TrimSpace(translate(c.Children))
					} else {
						children = strings.TrimSpace(translate(c.Children.FilterOut("TEXT")))
					}
					lines := strings.Split(children, "\n")
					var p string

					var blockDelimLength int
					var blockDelimChar byte

				lineLoop:
					for _, l := range lines {
						if l == "" {
							if blockDelimLength == 0 {
								p += "+"
							}
							p += "\n"
							continue
						}
						p += l + "\n"
						if blockDelimLength == 0 {
							if len(l) >= 4 &&
								(l[0] == '-' ||
									l[0] == '=' ||
									l[0] == '/' ||
									l[0] == '.' ||
									l[0] == '+' ||
									l[0] == '_' ||
									l[0] == '*' ||
									l[0] == '|') {
								var delim byte
								if l[0] == '|' {
									delim = '='
								} else {
									delim = l[0]
								}
								for i := range l {
									if delim != l[i] {
										//not a delim
										continue lineLoop
									}
								}
								blockDelimLength = len(l)
								blockDelimChar = l[0]
							}
						} else {
							if len(l) == blockDelimLength && blockDelimChar == l[0] {
								blockDelimLength = 0
							}
						}
					}
					for strings.Contains(p, "\n+\n+\n") {
						p = strings.Replace(p, "\n+\n+\n", "\n+\n", -1)
					}
					var itemDecor string

					term := translate(c.Parent.Children.Filter("term"))
					if term != "" {
						itemDecor = "\n" + term + ":"
						for range c.Ancestors().Filter("varlistentry") {
							itemDecor += ":"
						}
						itemDecor += " "
					} else {
						itemDecor = "\n" + bullet
						for range c.Ancestors().Filter(cfg["listitems"]...) {
							itemDecor += bullet
						}
					}
					if len(p) > 2 {

						if p[len(p)-2:] == "+\n" {
							p = p[:len(p)-2]
						}

						var bypassBrokenInclusionsed bool
						for _, start := range "1234567890qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM" {
							if string(p[0]) == string(start) || string(p[1]) == string(start) || string(p[2]) == string(start) {
								output += itemDecor + " " + p + "\n"
								bypassBrokenInclusionsed = true
								break
							}
						}
						if !bypassBrokenInclusionsed {
							output += itemDecor + " &blank;\n+\n" + p + "\n"
						}
					}
				case c.IsKind("revision"):
					number := translate(c.Children.Flatten().Filter("revnumber"))
					date := translate(c.Children.Flatten().Filter("date"))
					author := translate(c.Children.Filter("author"))

					output += "\n" + number + ":: " + date + ", " + author + "\n" + translate(c.Children.Filter("revdescription"))
				case c.IsKind("affiliation"):
					orgname := translate(c.Children.Filter("orgname"))
					output += decorateIfNotBlank(orgname, "\n", "\n")
					orgdiv := translate(c.Children.Filter("orgdiv"))
					if orgdiv != "" {
						if orgname == "" {
							output += "\n"
						}
						output += orgdiv + "\n"
					}
				case c.IsKind("author", "editor"):
					if c.IsWithin("authorgroup") {
						fn := decorateIfNotBlank(translate(c.Children.Filter("firstname")), "", " ")
						output += "\n." + fn + translate(c.Children.Filter("surname")) + "\n"
						var content []string
						affiliation := translate(c.Children.Filter("affiliation"))
						if affiliation != "" {
							content = append(content, affiliation)
						}

						email := translate(c.Children.Filter("email"))
						if email != "" {
							content = append(content, email)
						}

						contrib := translate(c.Children.Filter("contrib"))
						if contrib != "" {
							content = append(content, contrib)
						}

						if len(content) == 0 {
							output += "\n&blank;"
						}
						output += strings.Join(content, "\n") + "\n"
					} else {
						output += translate(c.Children.Filter("firstname")) + " " + translate(c.Children.Filter("surname")) + " (" + translate(c.Children.Filter("email")) + ")"
					}
				case c.IsKind("term"):
					term := strings.TrimSpace(translate(c.Children))
					plain := strings.TrimSpace(translate(c.Children.FilterOut("indexterm").Flatten().Filter("TEXT")))
					if id, ok := c.Parent.Attributes["id"]; ok {
						output += "[[" + id + "," + plain + "]]\n"
					}
					output += strings.Replace(term, "\n", "", -1)
				case c.IsKind("title", "phrase", "date", "firstname", "surname", "orgdiv", "email", "textobject", "primary", "secondary", "tertiary", "seealso", "see"):
					output += strings.TrimSpace(translate(c.Children))
				case c.IsKind(cfg["monospace"]...):
					if c.IsWithin(cfg["monospace"]...) {
						output += translate(c.Children)
					} else {
						output += quote(c, "`")
					}
				case c.IsKind(cfg["superscript"]...):
					if c.IsWithin(cfg["superscript"]...) {
						output += translate(c.Children)
					} else {
						output += quote(c, "^")
					}
				case c.IsKind(cfg["italics"]...):
					if c.IsWithin(cfg["italics"]...) {
						output += translate(c.Children)
					} else {
						output += quote(c, "_")
					}
				case c.IsKind(cfg["bold"]...):
					if c.IsWithin(cfg["bold"]...) {
						output += translate(c.Children)
					} else {
						output += quote(c, "*")
					}
				case c.IsKind(cfg["highlight"]...) || c.IsKind(cfg.unQuotedCustom()...):
					if c.IsWithin(cfg["highlight"]...) {
						output += translate(c.Children)
					} else {
						output += quote(c, "#")
					}
				case c.IsKind("indexterm"):
					var terms []string
					s := strings.TrimSpace(translate(c.Children.Filter("primary").Flatten().Filter("TEXT")))
					if s != "" {
						terms = append(terms, s)
					}

					s = strings.TrimSpace(translate(c.Children.Filter("secondary").Flatten().Filter("TEXT")))
					if s != "" {
						terms = append(terms, s)
					}

					s = strings.TrimSpace(translate(c.Children.Filter("tertiary").Flatten().Filter("TEXT")))
					if s != "" {
						terms = append(terms, s)
					}

					s = strings.TrimSpace(translate(c.Children.Filter("see").Flatten().Filter("TEXT")))
					if s != "" {
						terms = append(terms, s)
					}

					s = strings.TrimSpace(translate(c.Children.Filter("seealso").Flatten().Filter("TEXT")))
					if s != "" {
						terms = append(terms, s)
					}

					if len(terms) > 0 {
						output += "indexterm:[" + strings.Join(terms, ",") + "]"
					}
				case c.IsKind("quote"):
					output += "\"" + translate(c.Children) + "\""
				case c.IsKind("manvolnum"):
					output += "(" + translate(c.Children) + ")"
				case c.IsKind("guibutton"):
					if c.IsWithin(cfg["literal"]...) {
						output += translate(c.Children)
					} else {
						output += "btn:[" + translate(c.Children) + "]"
					}
				case c.IsKind("menuchoice"):
					var chs []string
					for _, ch := range c.Children.FilterOut("guimenu") {
						chs = append(chs, translate(xmlTree.Chunks{ch}))
					}
					output += "menu:" + strings.TrimSpace(translate(c.Children.Filter("guimenu"))) + "[" + strings.Join(chs, " > ") + "]"
				case c.IsKind("keycap"):
					if c.IsWithin("keycombo") {
						output += translate(c.Children)
					} else {
						output += "kbd:[" + translate(c.Children) + "]"
					}
				case c.IsKind("keycombo"):
					var chs []string
					for _, ch := range c.Children {
						chs = append(chs, translate(xmlTree.Chunks{ch}))
					}
					output += "kbd:[" + strings.Join(chs, " + ") + "]"
				case c.IsKind("guimenu", "guisubmenu", "optional", "productnumber", "edition", "pubsnumber"):
					output += translate(c.Children)
				case c.IsKind("remark"):
					output += "\n//" + translate(c.Children) + "\n"
				case c.IsKind("keyword", "subjectterm"):
					if ad.Keywords == nil {
						ad.Keywords = make(map[string]struct{})
					}
					ad.Keywords[strings.TrimSpace(translate(c.Children))] = struct{}{}
				default:
					for _, ch := range c.Children {
						s := translate(xmlTree.Chunks{ch})
						if ch.IsKind("TEXT") {
							s = strings.TrimSpace(s)
							if s != "" {
								fmt.Println(color.YellowString("Unknown:"), ch.XML())
							}
						}
						output += s
					}
				}
			} else {
				delete(register, c)
				for _, ch := range c.Children.Flatten() {
					delete(register, ch)
				}
			}
		}
		return output
	}

	ad.Data["master.adoc"] = translate(data)

	for f, d := range ad.Data {
		d = strings.Replace(d, "``", "` `", -1)
		for _, delim := range " ,.!?-\n()|" {
			d = strings.Replace(d, "pass:attributes[{blank}]"+string(delim), string(delim), -1)
			d = strings.Replace(d, string(delim)+"pass:attributes[{blank}]", string(delim), -1)
		}
		d = strings.Replace(d, "pass:attributes[{blank}]:", ":", -1)
		for e := range ad.Entities {
			d = strings.Replace(d, "&"+e+";", "{"+e+"}", -1)
		}
		for strings.Contains(d, "\n\n\n") {
			d = strings.Replace(d, "\n\n\n", "\n\n", -1)
		}
		d = strings.TrimSpace(d)
		ad.Data[f] = d
	}
	for c := range register {
		if s := strings.TrimSpace(c.XML()); s != "" {
			fmt.Println(color.RedString("\nUnprocessed:"), s)
			fmt.Println(c.Parent.XML())
		}
	}
	return ad

}
