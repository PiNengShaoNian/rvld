package linker

type MergedSection struct {
	Chunk
	Map map[string]*SectionFragment
}

func NewMergedSection(name string, flags uint64, typ uint32) *MergedSection {
	m := &MergedSection{
		Chunk: NewChunk(),
		Map:   make(map[string]*SectionFragment),
	}

	m.Name = name
	m.Shdr.Flags = flags
	m.Shdr.Type = typ
	return m
}
