package xmlTree

type Chunk struct {
	Kind       string
	Attributes map[string]string
	Parent     *Chunk
	Children   Chunks
}

func newChunk(kind string) *Chunk {
	c := &Chunk{}
	c.Attributes = make(map[string]string)
	c.Kind = kind
	return c
}

func newTextChunk(text string) *Chunk {
	c := newChunk("TEXT")
	c.Attributes["TEXT"] = text
	return c
}

func newDirectiveChunk(text string) *Chunk {
	c := newChunk("DIRECTIVE")
	c.Attributes["DIRECTIVE"] = text
	return c
}

func newProcessingInstructionChunk(target, instruction string) *Chunk {
	c := newChunk("PROCINST")
	c.Attributes["TARGET"] = target
	c.Attributes["INSTRUCTION"] = instruction
	return c
}

func (c Chunk) XML() string {
	if c.IsKind("TEXT") {
		return c.Attributes["TEXT"]
	}
	var kvs string
	for k, v := range c.Attributes {
		kvs += " " + k + "=\"" + v + "\""
	}
	output := "<" + c.Kind + kvs + ">"

	for _, child := range c.Children {
		output += child.XML()
	}
	output += "</" + c.Kind + ">"
	return output
}

func (c *Chunk) AddChildren(children Chunks) {
	for _, ch := range children {
		c.AddChild(ch)
	}
}

func (c *Chunk) AddChild(child *Chunk) {
	c.Children = append(c.Children, child)
	child.Parent = c
}

func (c Chunk) Ancestors() Chunks {
	var output Chunks
	ancestor := c.Parent
	for ancestor != nil {
		output = append(output, ancestor)
		ancestor = ancestor.Parent
	}
	return output
}

func (c Chunk) IsWithin(kinds ...string) bool {
	ancestor := c.Parent
	for ancestor != nil {
		if ancestor.IsKind(kinds...) {
			return true
		}
		ancestor = ancestor.Parent
	}
	return false
}

func (c Chunk) IsKind(kinds ...string) bool {
	for _, k := range kinds {
		if c.Kind == k {
			return true
		}
	}
	return false
}
