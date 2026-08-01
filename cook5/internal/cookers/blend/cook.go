package blend

import (
	"io"
)

// Okay duh let's keep our python cooker I suppose.

// TODO: eventually we'll just generate the entire work.Action here
func Cook(outDir, guhDir, blend string) error {
	blender, err := openBlender()
	if err != nil {
		return err
	}

	// TODO: go:embed the python code and sneak it in by other means
	io.WriteString(blender.stdin, "import sys; sys.path.insert(0, 'cook5/internal/cookers/blend/python')\n")
	io.WriteString(blender.stdin, "import bpy\n")
	io.WriteString(blender.stdin, "import eugh\n")

	io.WriteString(blender.stdin, "eugh.cook(input(), input(), input())\n")
	io.WriteString(blender.stdin, outDir+"\n")
	io.WriteString(blender.stdin, guhDir+"\n")
	io.WriteString(blender.stdin, blend+"\n")

	io.WriteString(blender.stdin, "exit(0)\n")

	defer blender.cmd.Wait()

	return nil
}
