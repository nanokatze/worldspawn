#!/usr/bin/env bash

set -e

OUTDIR="$HOME/deck/home/deck/worldspawn/shaders/"

SLANG=../slang/build/Release/bin/slangc
SLANG_FLAGS="-target spirv -profile spirv_1_6 -fvk-use-entrypoint-name -force-glsl-scalar-layout -matrix-layout-row-major -capability vk_mem_model"

$SLANG $SLANG_FLAGS -o "$OUTDIR/mesh.spv" src/internal/renderer/mesh.slang
$SLANG $SLANG_FLAGS -o "$OUTDIR/renderer.spv" src/internal/renderer/renderer.slang
$SLANG $SLANG_FLAGS -o "$OUTDIR/scene.spv" src/internal/renderer/scene.slang
$SLANG $SLANG_FLAGS -o "$OUTDIR/test.spv" src/internal/renderer/test.slang
