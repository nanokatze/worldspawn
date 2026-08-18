#!/usr/bin/env bash

# TODO: go:embed the shaders

set -e

OUTDIR=shaders

SLANG=../slang/build/Release/bin/slangc
SLANG_FLAGS="-target spirv -profile spirv_1_6 -fvk-use-entrypoint-name -fvk-use-c-layout -matrix-layout-row-major -capability vk_mem_model -capability spvDescriptorHeapEXT"

$SLANG $SLANG_FLAGS -o "$OUTDIR/geometry_skinning.spv" internal/geometry/skinning.slang &
$SLANG $SLANG_FLAGS -o "$OUTDIR/postprocess_bloom.spv" internal/postprocess/bloom.slang &
$SLANG $SLANG_FLAGS -o "$OUTDIR/renderer_scene_render.spv" internal/renderer/scene_render.slang &

wait
