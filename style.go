package main

import "strings"

func init() {
	DefaultStyle = NewStyle()
	DefaultStyle.AddFromString("admonitions", "note,warning,important")
	DefaultStyle.AddFromString("listitems", "listitem,step,biblioentry,member,contrib")
	DefaultStyle.AddFromString("paragraphs", "para,simpara,subtitle")
	DefaultStyle.AddFromString("literal", "screen,synopsis,programlisting,indexterm,mediaobject")

	DefaultStyle.AddFromString("custom", "package,application,citetitle,command,option")
	DefaultStyle.AddFromString("monospace", "literal,wordasword,filename,guilabel,systemitem,prompt,computeroutput,userinput,revnumber,parameter,guimenuitem,errortype,varname,function,methodname,classname,property,type,command,option,sgmltag,code,envar,guiicon")
	DefaultStyle.AddFromString("superscript", "superscript")
	DefaultStyle.AddFromString("italic", "firstterm,replaceable,citebiblioid,citetitle,citation,mathphrase,lineannotation")
	DefaultStyle.AddFromString("bold", "emphasis,orgname,trademark,acronym,abbrev,uri,refentrytitle,application,package,productname")
	DefaultStyle.AddFromString("highlight", "")
}

var DefaultStyle Style

type Style map[string][]string

func NewStyle() Style { return make(map[string][]string) }

func (s Style) Add(category string, kinds ...string) {
	s[category] = append(s[category], kinds...)
}

func (s Style) AddFromString(category string, kindList string) {
	s.Add(category, strings.Split(kindList, ",")...)
}

func (s Style) OverrideWith(s2 Style) {
	for k, v := range s2 {
		s[k] = v
	}
}

func (s Style) allQuotes() []string {
	var answer []string
	answer = append(answer, s["monospace"]...)
	answer = append(answer, s["superscript"]...)
	answer = append(answer, s["italics"]...)
	answer = append(answer, s["bold"]...)
	answer = append(answer, s["highlight"]...)
	return answer
}

func (s Style) unQuotedCustom() []string {
	var answer []string
	for _, cc := range s["custom"] {
		var alreadyQuoted bool
		for _, q := range s.allQuotes() {
			if cc == q {
				alreadyQuoted = true
				break
			}
		}
		if !alreadyQuoted {
			answer = append(answer, cc)
		}
	}
	return answer
}
