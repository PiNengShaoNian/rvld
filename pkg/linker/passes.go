package linker

import "rvld/pkg/utils"

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
	ctx.Ehdr = NewOutputEhdr()
	ctx.Chunks = append(ctx.Chunks, ctx.Ehdr)
}

func GetFileSize(ctx *Context) uint64 {
	fileOff := uint64(0)

	for _, c := range ctx.Chunks {
		fileOff += utils.AlignTo(fileOff, c.GetShdr().AddrAlign)
		fileOff += c.GetShdr().Size
	}

	return fileOff
}
