package main

import (
	"os"
	"rvld/pkg/linker"
	"rvld/pkg/utils"
)

func main() {
	if len(os.Args) < 2 {
		utils.Fatal("wrong args")
	}

	file := linker.MustNewFile(os.Args[1])

	if !linker.CheckMagic(file.Contents) {
		utils.Fatal("not an ELF file")
	}

	objFile := linker.NewObjectFile(file)
	utils.Assert(len(objFile.ElfSections) == 11)

	objFile.Parse()
	utils.Assert(objFile.FirstGlobal == 10)
	utils.Assert(len(objFile.ElfSyms) == 12)

	for _, sym := range objFile.ElfSyms {
		println(linker.ElfGetName(objFile.SymbolStrtab, sym.Name))
	}
}
