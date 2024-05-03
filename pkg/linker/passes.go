package linker

import (
	"debug/elf"
	"math"
	"rvld/pkg/utils"
	"sort"
)

func CreateInternalFile(ctx *Context) {
	obj := &ObjectFile{}
	ctx.InternalObj = obj
	ctx.Objs = append(ctx.Objs, obj)

	ctx.InternalEsyms = make([]Sym, 1)
	obj.Symbols = append(obj.Symbols, NewSymbol(""))
	obj.FirstGlobal = 1
	obj.IsAlive = true

	obj.ElfSyms = ctx.InternalEsyms
}

func ResolveSymbols(ctx *Context) {
	for _, file := range ctx.Objs {
		file.ResolveSymbols()
	}

	MarkLiveObjects(ctx)
}

func MarkLiveObjects(ctx *Context) {
	roots := make([]*ObjectFile, 0)
	for _, file := range ctx.Objs {
		if file.IsAlive {
			roots = append(roots, file)
		}
	}

	utils.Assert(len(roots) > 0)

	for len(roots) > 0 {
		file := roots[0]
		if !file.IsAlive {
			continue
		}

		file.MarkLiveObjects(func(of *ObjectFile) {
			roots = append(roots, file)
		})
		roots = roots[1:]
	}

	for _, file := range ctx.Objs {
		if !file.IsAlive {
			file.ClearSymbols()
		}
	}

	ctx.Objs = utils.RemoveIf[*ObjectFile](ctx.Objs, func(file *ObjectFile) bool {
		return !file.IsAlive
	})
}

func RegisterSectionPieces(ctx *Context) {
	for _, file := range ctx.Objs {
		file.RegisterSectionPieces()
	}
}

func CreateSyntheticSections(ctx *Context) {
	push := func(chunk Chunker) Chunker {
		ctx.Chunks = append(ctx.Chunks, chunk)
		return chunk
	}

	ctx.Ehdr = push(NewOutputEhdr()).(*OutputEhdr)
	ctx.Shdr = push(NewOutputShdr()).(*OutputShdr)
}

func SetOutputSectionOffsets(ctx *Context) uint64 {
	addr := IMAGE_BASE

	for _, chunk := range ctx.Chunks {
		if chunk.GetShdr().Flags&uint64(elf.SHF_ALLOC) == 0 {
			continue
		}

		addr = utils.AlignTo(addr, chunk.GetShdr().AddrAlign)
		chunk.GetShdr().Addr = addr

		// 程序初始化时tbss并不需要分配内存，而是在每个线程第一次访问TLS变量时动态分配并初始化。
		if !isTbss(chunk) {
			addr += chunk.GetShdr().Size
		}
	}

	i := 0
	first := ctx.Chunks[0]
	for {
		shdr := ctx.Chunks[i].GetShdr()
		shdr.Offset = shdr.Addr - first.GetShdr().Addr
		i++

		if i >= len(ctx.Chunks) || ctx.Chunks[i].GetShdr().Flags&uint64(elf.SHF_ALLOC) == 0 {
			break
		}
	}

	lastShdr := ctx.Chunks[i-1].GetShdr()
	fileOff := lastShdr.Offset + lastShdr.Size

	for ; i < len(ctx.Chunks); i++ {
		shdr := ctx.Chunks[i].GetShdr()
		fileOff = utils.AlignTo(fileOff, shdr.AddrAlign)
		shdr.Offset = fileOff
		fileOff += shdr.Size
	}

	return fileOff
}

func BinSections(ctx *Context) {
	group := make([][]*InputSection, len(ctx.OutputSections))
	for _, file := range ctx.Objs {
		for _, section := range file.Sections {
			if section == nil || !section.IsAlive {
				continue
			}

			idx := section.OutputSection.Idx
			group[idx] = append(group[idx], section)
		}
	}

	for idx, section := range ctx.OutputSections {
		section.Members = group[idx]
	}
}

func CollectOutputSections(ctx *Context) []Chunker {
	chunks := make([]Chunker, 0)
	for _, section := range ctx.OutputSections {
		if len(section.Members) > 0 {
			chunks = append(chunks, section)
		}
	}

	return chunks
}

func ComputeSectionSizes(ctx *Context) {
	for _, outputSection := range ctx.OutputSections {
		offset := uint64(0)
		p2align := uint64(0)

		for _, inputSection := range outputSection.Members {
			offset = utils.AlignTo(offset, 1<<inputSection.P2Align)
			inputSection.Offset = uint32(offset)
			offset += uint64(inputSection.ShSize)
			p2align = uint64(math.Max(float64(p2align), float64(inputSection.P2Align)))
		}

		outputSection.Shdr.Size = offset
		outputSection.Shdr.AddrAlign = 1 << p2align
	}
}

func SortOutputSections(ctx *Context) {
	rank := func(chunk Chunker) int32 {
		typ := chunk.GetShdr().Type
		flags := chunk.GetShdr().Flags

		if flags&uint64(elf.SHF_ALLOC) == 0 {
			return math.MaxInt32 - 1
		}
		if chunk == ctx.Shdr {
			return math.MaxInt32
		}
		if chunk == ctx.Ehdr {
			return 0
		}
		if typ == uint32(elf.SHT_NOTE) {
			return 2
		}

		b2i := func(b bool) int {
			if b {
				return 1
			}
			return 0
		}

		writeable := b2i(flags&uint64(elf.SHF_WRITE) != 0)
		notExec := b2i(flags&uint64(elf.SHF_EXECINSTR) == 0)
		notTls := b2i(flags&uint64(elf.SHF_TLS) == 0)
		isBss := b2i(typ == uint32(elf.SHT_NOBITS))

		return int32(writeable<<7 | notExec<<6 | notTls<<5 | isBss<<4)
	}

	sort.SliceStable(ctx.Chunks, func(i, j int) bool {
		return rank(ctx.Chunks[i]) < rank(ctx.Chunks[j])
	})
}

// tls段中也分data和bss段
func isTbss(chunk Chunker) bool {
	shdr := chunk.GetShdr()
	return shdr.Type == uint32(elf.SHT_NOBITS) &&
		shdr.Flags&uint64(elf.SHF_TLS) != 0
}
