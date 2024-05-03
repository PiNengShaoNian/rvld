package linker

import (
	"debug/elf"
	"fmt"
	"rvld/pkg/utils"
)

type InputFile struct {
	File         *File
	ElfSections  []Shdr
	ElfSyms      []Sym
	FirstGlobal  int
	ShStrtab     []byte
	SymbolStrtab []byte
	IsAlive      bool
	Symbols      []*Symbol
	LocalSymbols []Symbol
}

func NewInputFile(file *File) InputFile {
	f := InputFile{
		File: file,
	}
	if len(file.Contents) < EhdrSize {
		utils.Fatal("file too small")
	}
	if !CheckMagic(f.File.Contents) {
		utils.Fatal("not an ELF file")
	}

	ehdr := utils.Read[Ehdr](file.Contents)
	contents := file.Contents[ehdr.ShOff:]
	shdr := utils.Read[Shdr](contents)

	numSections := uint64(ehdr.ShNum)
	if numSections == 0 {
		numSections = shdr.Size
	}

	f.ElfSections = []Shdr{shdr}

	for numSections > 1 {
		contents = contents[ShdrSize:]
		f.ElfSections = append(f.ElfSections, utils.Read[Shdr](contents))
		numSections--
	}

	shstrndx := int64(ehdr.ShStrndx)
	// 当ehdr.ShStrndx的值为UINT16_MAX的时候，
	// 实际的shstrndx存储在第一个SectionHeaderTable第一项中的Link中
	if ehdr.ShStrndx == uint16(elf.SHN_XINDEX) {
		shstrndx = int64(shdr.Link)
	}

	f.ShStrtab = f.GetBytesFromIdx(shstrndx)

	return f
}

func (f *InputFile) GetBytesFromShdr(s *Shdr) []byte {
	end := s.Offset + s.Size
	if uint64(len(f.File.Contents)) < end {
		utils.Fatal(fmt.Sprintf("section header is out of range: %d", s.Offset))
	}

	return f.File.Contents[s.Offset : s.Offset+s.Size]
}

func (f *InputFile) GetBytesFromIdx(idx int64) []byte {
	return f.GetBytesFromShdr(&f.ElfSections[idx])
}

func (f *InputFile) FillUpElfSyms(s *Shdr) {
	bytes := f.GetBytesFromShdr(s)
	f.ElfSyms = utils.ReadSlice[Sym](bytes, SymSize)
}

func (f *InputFile) FindSection(ty uint32) *Shdr {
	for i := 0; i < len(f.ElfSections); i++ {
		shdr := &f.ElfSections[i]
		if shdr.Type == ty {
			return shdr
		}
	}
	return nil
}

func (f *InputFile) GetEhdr() Ehdr {
	return utils.Read[Ehdr](f.File.Contents)
}
