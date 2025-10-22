package renderer

import (
	"log"
	"math"
	"sync"
	"unsafe"

	"worldspawn/gpu"
	"worldspawn/internal/renderer/internal/compiler"
	"worldspawn/internal/renderer/internal/compiler/core"
	"worldspawn/internal/renderer/internal/mc"
)

type MaterialSet struct {
	pipeline *gpu.RayTracingPipeline
	sbt      gpu.ShaderBindingTable
}

type Material struct {
	// TODO: rename to something like interpreterProgram
	code    gpu.Slice[uint32]
	outputs uint32 // start of the register array containing the outputs

	emissive gpu.Slice[uint32]
}

func buildArith1(b *compiler.Rewriter, op compiler.Op, x *compiler.Class) *compiler.Class {
	return b.Value2(op, x.Type(), nil, x)
}

func buildArith2(b *compiler.Rewriter, op compiler.Op, x, y *compiler.Class) *compiler.Class {
	return b.Value2(op, x.Type(), nil, x, y)
}

func buildFract(b *compiler.Rewriter, v *compiler.Class) *compiler.Class {
	return buildArith2(b, mc.OpFSub, v, buildArith1(b, mc.OpFFloor, v))
}

var TestMaterial = sync.OnceValue(func() *Material {
	sea := compiler.NewSea()
	b := compiler.Rewriter{
		Sea:   sea,
		Rules: append(append([]compiler.RewriteRule(nil), mc.CommonRules...), mc.LowerToMVM...),
	}

	normal := b.Value2(
		mc.OpMVMLoadNormal,
		core.MakeArrayType(3, core.Int32),
		nil)
	normal_x := core.ArrayExtract(&b, normal, 0)
	normal_y := core.ArrayExtract(&b, normal, 1)
	normal_z := core.ArrayExtract(&b, normal, 2)

	uv := b.Value2(
		mc.OpMVMLoadAttr,
		core.MakeArrayType(2, core.Int32),
		uint32(unsafe.Offsetof(materialParams{}.UVs)))
	u := core.ArrayExtract(&b, uv, 0)
	v := core.ArrayExtract(&b, uv, 1)

	one := core.Const(&b, core.Int32, int64(math.Float32bits(1)))

	fractU := buildFract(&b, u)
	fractV := buildFract(&b, v)

	oneMinusFractU := buildArith2(&b, mc.OpFSub, one, fractU)
	oneMinusFractV := buildArith2(&b, mc.OpFSub, one, fractV)

	idk1 := buildArith2(&b, mc.OpFMin, fractU, fractV)
	idk2 := buildArith2(&b, mc.OpFMin, oneMinusFractU, oneMinusFractV)
	idk3 := buildArith2(&b, mc.OpFMin, idk1, idk2)

	selector := b.Value2(
		mc.OpMVMFLessOrEqualE8M23,
		core.Int32,
		nil,
		idk3,
		core.Const(&b, core.Int32, int64(math.Float32bits(0.025))))

	// TODO: introduce a rule for scalarizing CondSelect. Note that right now we
	// fall over at extraction time, so...
	color_r := core.CondSelect(
		&b,
		one,
		b.Value2(
			mc.OpMVMLoadParam,
			core.Int32,
			uint32(unsafe.Offsetof(materialParams{}.BaseColor))+0),
		selector)
	color_g := core.CondSelect(
		&b,
		one,
		b.Value2(
			mc.OpMVMLoadParam,
			core.Int32,
			uint32(unsafe.Offsetof(materialParams{}.BaseColor))+4),
		selector)
	color_b := core.CondSelect(
		&b,
		one,
		b.Value2(
			mc.OpMVMLoadParam,
			core.Int32,
			uint32(unsafe.Offsetof(materialParams{}.BaseColor))+8),
		selector)

	program := core.MakeArray(
		&b,
		core.Int32,
		normal_x, normal_y, normal_z,
		color_r, color_g, color_b)

	return NewMaterial(sea, program)
})

var TestMaterial2 = sync.OnceValue(func() *Material {
	sea := compiler.NewSea()
	b := &compiler.Rewriter{
		Sea:   sea,
		Rules: append(append([]compiler.RewriteRule(nil), mc.CommonRules...), mc.LowerToMVM...),
	}

	normal := b.Value2(
		mc.OpMVMLoadNormal,
		core.MakeArrayType(3, core.Int32),
		nil)
	normal_x := core.ArrayExtract(b, normal, 0)
	normal_y := core.ArrayExtract(b, normal, 1)
	normal_z := core.ArrayExtract(b, normal, 2)
	_diffuse_r := core.Const(b, core.Int32, int64(math.Float32bits(0.0)))
	_diffuse_g := core.Const(b, core.Int32, int64(math.Float32bits(0.0)))
	_diffuse_b := core.Const(b, core.Int32, int64(math.Float32bits(0.0)))
	_emission_r := core.Const(b, core.Int32, int64(math.Float32bits(10.0)))
	_emission_g := core.Const(b, core.Int32, int64(math.Float32bits(3.33)))
	_emission_b := core.Const(b, core.Int32, int64(math.Float32bits(10.0)))
	_color := core.MakeArray(
		b,
		core.Int32,
		normal_x, normal_y, normal_z,
		_diffuse_r, _diffuse_g, _diffuse_b,
		_emission_r, _emission_g, _emission_b)

	tmp := NewMaterial(sea, _color)
	tmp.emissive = tmp.code
	return tmp
})

func NewMaterial(sea *compiler.Sea, v *compiler.Class) *Material {
	host, outputs := mc.CompileMVMProgram(sea, v)

	device := gpu.MakeSliceUncached[uint32](len(host))
	copy(device.Value(), host)

	log.Println("outputs register", uint32(outputs))

	return &Material{
		code:    device,
		outputs: uint32(outputs),
	}
}
