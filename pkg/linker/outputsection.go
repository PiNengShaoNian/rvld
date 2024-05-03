package linker

import "debug/elf"

type OutputSection struct {
	Chunk
	Members []*InputSection
	Idx     uint32
}

func NewOutputSection(name string, typ uint32, flags uint64, idx uint32) *OutputSection {
	o := &OutputSection{
		Chunk: NewChunk(),
	}
	o.Name = name
	o.Shdr.Type = typ
	o.Shdr.Flags = flags
	o.Idx = idx
	return o
}

func (o *OutputSection) CopyBuf(ctx *Context) {
	if o.Shdr.Type == uint32(elf.SHT_NOBITS) {
		return
	}

	base := ctx.Buf[o.Shdr.Offset:]
	for _, section := range o.Members {
		section.WriteTo(base[section.Offset:])
	}
}

func GetOutputSection(ctx *Context, name string, typ uint64, flags uint64) *OutputSection {
	name = GetOutputName(name, flags)
	flags = flags & ^uint64(elf.SHF_GROUP) & ^uint64(elf.SHF_MERGE) &
		^uint64(elf.SHF_STRINGS) & ^uint64(elf.SHF_COMPRESSED)
	find := func() *OutputSection {
		for _, outputSection := range ctx.OutputSections {
			if name == outputSection.Name && typ == uint64(outputSection.Shdr.Type) &&
				flags == outputSection.Shdr.Flags {
				return outputSection
			}
		}
		return nil
	}

	if outputSection := find(); outputSection != nil {
		return outputSection
	}

	outputSection := NewOutputSection(name, uint32(typ), flags, uint32(len(ctx.OutputSections)))
	ctx.OutputSections = append(ctx.OutputSections, outputSection)
	return outputSection
}
