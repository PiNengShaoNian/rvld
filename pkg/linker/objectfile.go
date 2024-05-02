package linker

import (
	"bytes"
	"debug/elf"
	"rvld/pkg/utils"
)

type ObjectFile struct {
	InputFile
	SymtabSec         *Shdr
	SymtabShndxSec    []uint32
	Sections          []*InputSection
	MergeableSections []*MergeableSection
}

func NewObjectFile(file *File, isAlive bool) *ObjectFile {
	o := &ObjectFile{InputFile: NewInputFile(file)}
	o.IsAlive = isAlive
	return o
}

func (o *ObjectFile) Parse(ctx *Context) {
	o.SymtabSec = o.FindSection(uint32(elf.SHT_SYMTAB))
	if o.SymtabSec != nil {
		o.FirstGlobal = int(o.SymtabSec.Info)
		o.FillUpElfSyms(o.SymtabSec)
		// SymbolTable中的Link字段存储的值是strtab的下标值
		o.SymbolStrtab = o.GetBytesFromIdx(int64(o.SymtabSec.Link))
	}

	o.InitializeSections()
	o.InitializeSymbols(ctx)
	o.InitializeMergeableSections(ctx)
}

func (o *ObjectFile) InitializeSections() {
	o.Sections = make([]*InputSection, len(o.ElfSections))
	for i := 0; i < len(o.ElfSections); i++ {
		shdr := &o.ElfSections[i]
		switch elf.SectionType(shdr.Type) {
		case elf.SHT_GROUP, elf.SHT_SYMTAB, elf.SHT_STRTAB, elf.SHT_REL, elf.SHT_RELA,
			elf.SHT_NULL:
		case elf.SHT_SYMTAB_SHNDX:
			o.FillUpSymtabShndxSec(shdr)
		default:
			o.Sections[i] = NewInputSection(o, uint32(i))
		}
	}
}

func (o *ObjectFile) FillUpSymtabShndxSec(s *Shdr) {
	bytes := o.GetBytesFromShdr(s)
	o.SymtabShndxSec = utils.ReadSlice[uint32](bytes, 4)
}

func (o *ObjectFile) InitializeSymbols(ctx *Context) {
	if o.SymtabSec == nil {
		return
	}

	o.LocalSymbols = make([]Symbol, o.FirstGlobal)
	for i := 0; i < len(o.LocalSymbols); i++ {
		o.LocalSymbols[i] = *NewSymbol("")
	}
	o.LocalSymbols[0].File = o

	for i := 1; i < len(o.LocalSymbols); i++ {
		elfSym := &o.ElfSyms[i]
		sym := &o.LocalSymbols[i]
		sym.Name = ElfGetName(o.SymbolStrtab, elfSym.Name)
		sym.File = o
		sym.Value = elfSym.Val
		sym.SymIdx = i
		if !elfSym.IsAbs() {
			sym.SetInputSection(o.Sections[o.GetShndx(elfSym, i)])
		}
	}

	o.Symbols = make([]*Symbol, len(o.ElfSyms))
	for i := 0; i < len(o.LocalSymbols); i++ {
		o.Symbols[i] = &o.LocalSymbols[i]
	}

	for i := len(o.LocalSymbols); i < len(o.ElfSyms); i++ {
		elfSym := &o.ElfSyms[i]
		name := ElfGetName(o.SymbolStrtab, elfSym.Name)
		o.Symbols[i] = GetSymbolByName(ctx, name)
	}
}

func (o *ObjectFile) GetShndx(elfSym *Sym, idx int) int64 {
	utils.Assert(idx >= 0 && idx < len(o.ElfSyms))
	if elfSym.Shndx == uint16(elf.SHN_XINDEX) {
		return int64(o.SymtabShndxSec[idx])
	}

	return int64(elfSym.Shndx)
}

func (o *ObjectFile) ResolveSymbols() {
	for i := o.FirstGlobal; i < len(o.ElfSyms); i++ {
		sym := o.Symbols[i]
		elfSym := &o.ElfSyms[i]

		if elfSym.IsUndef() {
			continue
		}

		var inputSection *InputSection
		if !elfSym.IsAbs() {
			inputSection = o.GetSection(elfSym, i)
			if inputSection == nil {
				continue
			}
		}

		if sym.File == nil {
			sym.File = o
			sym.SetInputSection(inputSection)
		}
	}
}

func (o *ObjectFile) GetSection(elfSym *Sym, idx int) *InputSection {
	return o.Sections[o.GetShndx(elfSym, idx)]
}

func (o *ObjectFile) MarkLiveObjects(feeder func(*ObjectFile)) {
	utils.Assert(o.IsAlive)

	for i := o.FirstGlobal; i < len(o.ElfSyms); i++ {
		sym := o.Symbols[i]
		elfSym := &o.ElfSyms[i]

		if sym.File == nil {
			continue
		}

		if elfSym.IsUndef() && !sym.File.IsAlive {
			sym.File.IsAlive = true
			feeder(sym.File)
		}
	}
}

func (o *ObjectFile) ClearSymbols() {
	for _, sym := range o.Symbols[o.FirstGlobal:] {
		if sym.File == o {
			sym.Clear()
		}
	}
}

func (o *ObjectFile) InitializeMergeableSections(ctx *Context) {
	o.MergeableSections = make([]*MergeableSection, len(o.Sections))
	for i := 0; i < len(o.Sections); i++ {
		section := o.Sections[i]
		if section != nil && section.IsAlive &&
			section.Shdr().Flags&uint64(elf.SHF_MERGE) != 0 {
			o.MergeableSections[i] = splitSection(ctx, section)
			section.IsAlive = false
		}
	}
}

func findNull(data []byte, entSize int) int {
	if entSize == 1 {
		return bytes.Index(data, []byte{0})
	}

	for i := 0; i <= len(data)-entSize; i += entSize {
		bytes := data[i : i+entSize]
		if utils.AllZeros(bytes) {
			return i
		}
	}

	return -1
}

func splitSection(ctx *Context, section *InputSection) *MergeableSection {
	m := &MergeableSection{}
	shdr := section.Shdr()

	m.Parent = GetMergedSectionInstance(ctx, section.Name(), shdr.Type, shdr.Flags)
	m.P2Align = section.P2Align
	data := section.Contents
	offset := uint64(0)
	if (shdr.Flags & uint64(elf.SHF_STRINGS)) != 0 {
		for len(data) > 0 {
			end := findNull(data, int(shdr.EntSize))
			if end == -1 {
				utils.Fatal("string is not null terminated")
			}

			size := uint64(end) + shdr.EntSize
			substr := data[:size]
			data = data[size:]
			m.Strs = append(m.Strs, string(substr))
			m.FragOffsets = append(m.FragOffsets, uint32(offset))
			offset += size
		}
	} else {
		if uint64(len(data))%shdr.EntSize != 0 {
			utils.Fatal("section size is not multiple of entsize")
		}

		for len(data) > 0 {
			substr := data[:shdr.EntSize]
			data = data[shdr.EntSize:]
			m.Strs = append(m.Strs, string(substr))
			m.FragOffsets = append(m.FragOffsets, uint32(offset))
			offset += shdr.EntSize
		}
	}

	return m
}

func (o *ObjectFile) RegisterSectionPieces() {
	for _, m := range o.MergeableSections {
		if m == nil {
			continue
		}

		m.Fragments = make([]*SectionFragment, 0, len(m.Strs))
		for i := 0; i < len(m.Strs); i++ {
			m.Fragments = append(m.Fragments,
				m.Parent.Insert(m.Strs[i], uint32(m.P2Align)))
		}
	}

	for i := 1; i < len(o.ElfSyms); i++ {
		sym := o.Symbols[i]
		elfSym := &o.ElfSyms[i]

		if elfSym.IsAbs() || elfSym.IsUndef() || elfSym.IsCommon() {
			continue
		}

		m := o.MergeableSections[o.GetShndx(elfSym, i)]
		if m == nil {
			continue
		}
		frag, fragOffset := m.GetFragment(uint32(elfSym.Val))
		if frag == nil {
			utils.Fatal("bad symbol value")
		}
		sym.SetSectionFragment(frag)
		sym.Value = uint64(fragOffset)
	}
}
