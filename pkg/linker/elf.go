package linker

import (
	"bytes"
	"debug/elf"
	"rvld/pkg/utils"
	"strconv"
	"strings"
	"unsafe"
)

const IMAGE_BASE uint64 = 0x200000
const EF_RISCV_RVC uint32 = 1
const PageSize = 4096

const EhdrSize = int(unsafe.Sizeof(Ehdr{}))
const ShdrSize = int(unsafe.Sizeof(Shdr{}))
const PhdrSize = int(unsafe.Sizeof(Phdr{}))
const SymSize = int(unsafe.Sizeof(Sym{}))
const ArHdrSize = int(unsafe.Sizeof(ArHdr{}))
const RelaSize = int(unsafe.Sizeof(Rela{}))

type Ehdr struct {
	Ident     [16]uint8
	Type      uint16
	Machine   uint16
	Version   uint32
	Entry     uint64
	PhOff     uint64
	ShOff     uint64
	Flags     uint32
	EhSize    uint16
	PhEntSize uint16
	PhNum     uint16
	ShEntSize uint16
	ShNum     uint16
	ShStrndx  uint16
}

type Phdr struct {
	Type     uint32
	Flags    uint32
	Offset   uint64
	VAddr    uint64
	PAddr    uint64
	FileSize uint64
	MemSize  uint64
	Align    uint64
}

type Shdr struct {
	Name      uint32
	Type      uint32
	Flags     uint64
	Addr      uint64
	Offset    uint64
	Size      uint64
	Link      uint32
	Info      uint32
	AddrAlign uint64
	EntSize   uint64
}

type Sym struct {
	Name  uint32
	Info  uint8
	Other uint8
	Shndx uint16
	Val   uint64
	Size  uint64
}

type Rela struct {
	Offset uint64
	Type   uint32
	Sym    uint32
	Addend int64
}

type ArHdr struct {
	Name [16]byte
	Date [12]byte
	Uid  [6]byte
	Gid  [6]byte
	Mode [8]byte
	Size [10]byte
	Fmag [2]byte
}

func (a *ArHdr) HasPrefix(s string) bool {
	return strings.HasPrefix(string(a.Name[:]), s)
}

func (a *ArHdr) IsStrtab() bool {
	return a.HasPrefix("// ")
}

func (a *ArHdr) IsSymtab() bool {
	return a.HasPrefix("/ ") || a.HasPrefix("/Sym64/ ")
}

func (a *ArHdr) GetSize() int {
	size, err := strconv.Atoi(strings.TrimSpace(string(a.Size[:])))
	utils.MustNo(err)
	return size
}

func (a *ArHdr) ReadName(strTab []byte) string {
	// Long filename
	if a.HasPrefix("/") {
		start, err := strconv.Atoi(strings.TrimSpace(string(a.Name[1:])))
		utils.MustNo(err)
		end := start + bytes.Index(strTab[start:], []byte("/\n"))
		return string(strTab[start:end])
	}

	// Short filename
	end := bytes.Index(a.Name[:], []byte("/"))
	utils.Assert(end != -1)
	return string(a.Name[:end])
}

func ElfGetName(strTab []byte, offset uint32) string {
	length := uint32(bytes.Index(strTab[offset:], []byte{0}))
	return string(strTab[offset : offset+length])
}

func (s *Sym) IsAbs() bool {
	return s.Shndx == uint16(elf.SHN_ABS)
}

func (s *Sym) IsUndef() bool {
	return s.Shndx == uint16(elf.SHN_UNDEF)
}

func (s *Sym) IsCommon() bool {
	return s.Shndx == uint16(elf.SHN_COMMON)
}
