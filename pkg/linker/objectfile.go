package linker

import "debug/elf"

type ObjectFile struct {
	InputFile
	SymtabSec *Shdr
}

func NewObjectFile(file *File) *ObjectFile {
	o := &ObjectFile{InputFile: NewInputFile(file)}
	return o
}

func (o *ObjectFile) Parse() {
	o.SymtabSec = o.FindSection(uint32(elf.SHT_SYMTAB))
	if o.SymtabSec != nil {
		o.FirstGlobal = int64(o.SymtabSec.Info)
		o.FillUpElfSyms(o.SymtabSec)
		// SymbolTable中的Link字段存储的值是strtab的下标值
		o.SymbolStrtab = o.GetBytesFromIdx(int64(o.SymtabSec.Link))
	}
}
