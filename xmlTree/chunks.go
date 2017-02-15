package xmlTree

type Chunks []*Chunk

func (cs Chunks) XML() string {
	var output string
	for _, c := range cs {
		output += c.XML()
	}
	return output
}

func (cs Chunks) Filter(kinds ...string) Chunks {
	var output Chunks
	for _, c := range cs {
		if c.IsKind(kinds...) {
			output = append(output, c)
		}
	}
	return output
}

func (cs Chunks) Children() Chunks {
	var output Chunks
	for _, c := range cs {
		output = append(output, c.Children...)
	}
	return output
}

func (cs Chunks) Last(kinds ...string) *Chunk {
	if len(cs) == 0 {
		return nil
	}

	for x := len(cs) - 1; x > -1; x-- {
		for _, k := range kinds {
			if cs[x].Kind == k {
				return cs[x]
			}
		}
	}
	return nil
}

func (cs Chunks) Contains(kinds ...string) bool {
	for _, k := range kinds {
		for _, c := range cs {
			if c.Kind == k {
				return true
			}
		}
	}
	return false
}

func (cs Chunks) First(kinds ...string) *Chunk {
	for _, c := range cs {
		for _, k := range kinds {
			if c.Kind == k {
				return c
			}
		}
	}
	return nil
}

func (cs Chunks) FilterOut(kinds ...string) Chunks {
	var output Chunks
	for _, c := range cs {
		if !c.IsKind(kinds...) {
			output = append(output, c)
		}
	}
	return output
}

func (cs Chunks) Copy() Chunks {
	var output Chunks
	for _, c := range cs {
		n := newChunk(c.Kind)
		for k, v := range c.Attributes {
			n.Attributes[k] = v
		}
		n.Parent = c.Parent
		n.AddChildren(c.Children.Copy())

		output = append(output, n)
	}
	return output
}

func (cs Chunks) Flatten() Chunks {
	var output Chunks
	for _, c := range cs {
		output = append(output, c)
		output = append(output, c.Children.Flatten()...)
	}
	return output
}
