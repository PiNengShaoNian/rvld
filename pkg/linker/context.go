package linker

import "debug/elf"

type ContextArgs struct {
	Output       string
	Emulation    MachineType
	LibraryPaths []string
}

type Context struct {
	Args           ContextArgs
	Objs           []*ObjectFile
	SymbolMap      map[string]*Symbol
	MergedSections []*MergedSection
}

func NewContext() *Context {
	return &Context{
		Args: ContextArgs{
			Output:    "a.out",
			Emulation: MachineTypeNone,
		},
		SymbolMap: make(map[string]*Symbol),
	}
}

func GetMergedSectionInstance(ctx *Context, name string, typ uint32, flags uint64) *MergedSection {
	name = GetOutputName(name, flags)
	flags = flags & ^uint64(elf.SHF_GROUP) & ^uint64(elf.SHF_MERGE) &
		^uint64(elf.SHF_STRINGS) & ^uint64(elf.SHF_COMPRESSED)

	find := func() *MergedSection {
		for _, section := range ctx.MergedSections {
			if name == section.Name && flags == section.Shdr.Flags && typ == section.Shdr.Type {
				return section
			}
		}
		return nil
	}

	if section := find(); section != nil {
		return section
	}

	section := NewMergedSection(name, flags, typ)
	ctx.MergedSections = append(ctx.MergedSections, section)
	return section
}

func (m *MergedSection) Insert(key string, p2align uint32) *SectionFragment {
	frag, ok := m.Map[key]
	if !ok {
		frag = NewSectionFragment(m)
		m.Map[key] = frag
	}

	if frag.P2Align < p2align {
		frag.P2Align = p2align
	}

	return frag
}
