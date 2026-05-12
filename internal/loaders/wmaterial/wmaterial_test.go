package wmaterial

import (
	"testing"

	"worldspawn/internal/compiler"
	"worldspawn/internal/compiler/core"
	"worldspawn/internal/renderer/matc"
)

func TestXxx(t *testing.T) {
	src := []byte(`[
{"Bind": "ten",
 "Op": "IConst",
 "Type": "Int[32]",
 "Imm": "1092616192"},
{"Bind": "emissionSpectrum",
 "Op": "MakeArray",
 "Type": "Array[3, Int[32]]",
 "Args": ["ten", "ten", "ten"]},
{"Bind": "bsdf",
 "Op": "DFComposition",
 "Type": "BSDF"},
{"Bind": "uniformEDF",
 "Op": "UniformEDF",
 "Type": "EDF"},
{"Bind": "edf",
 "Op": "DFComposition",
 "Type": "EDF",
 "Args": ["emissionSpectrum", "uniformEDF"]},
{"Bind": "program",
 "Op": "MakeSurface",
 "Type": "Empty",
 "Args": ["bsdf", "edf"]}
]`)
	sea := compiler.NewSea()
	b := &compiler.Builder{Sea: sea, Rules: append(append([]compiler.RewriteRule(nil), core.Rules...), matc.LowerToInterpreter...)}
	ir, err := Parse(b, src)
	if err != nil {
		t.Fatal(err)
	}
	compiler.Dump(sea, ir, nil)
	_ = b
}
