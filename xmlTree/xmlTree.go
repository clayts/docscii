package xmlTree

import (
	"encoding/xml"
	"io"
	"strings"
)

func tokens(data string) []xml.Token {
	var output []xml.Token
	decoder := xml.NewDecoder(strings.NewReader(data))
	decoder.Strict = false
	for {
		token, err := decoder.Token()
		if err != nil && err != io.EOF {
			break
		}
		if token == nil {
			break
		}

		output = append(output, xml.CopyToken(token))
	}
	return output
}

func New(data string) Chunks {
	var output Chunks
	var current *Chunk
	stack := func(c *Chunk) {
		if current == nil {
			output = append(output, c)
		} else {
			current.AddChildren(Chunks{c})
		}
	}
	for _, token := range tokens(data) {
		switch element := token.(type) {
		case xml.EndElement:
			current = current.Parent
		case xml.CharData:
			stack(newTextChunk(string(element)))
		case xml.StartElement:
			c := newChunk(element.Name.Local)
			for _, a := range element.Attr {
				c.Attributes[a.Name.Local] = a.Value
			}
			stack(c)
			current = c
		case xml.Directive:
			stack(newDirectiveChunk(string(element)))
		case xml.ProcInst:
			stack(newProcessingInstructionChunk(element.Target, string(element.Inst)))
		}
	}
	return output
}
