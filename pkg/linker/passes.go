package linker

import "rvld/pkg/utils"

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

		file.MarkLiveObjects(ctx, func(of *ObjectFile) {
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
