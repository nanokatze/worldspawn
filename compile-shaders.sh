#!/usr/bin/env bash

set -e

OUTDIR="$HOME/deck/home/deck/worldspawn/shaders/"

SLANG=../slang/build/Release/bin/slangc
SLANG_FLAGS="-target spirv -profile spirv_1_6 -fvk-use-entrypoint-name -fvk-use-cpu-layout -matrix-layout-row-major -capability vk_mem_model"

$SLANG $SLANG_FLAGS -o "$OUTDIR/mesh.spv" src/internal/renderer/mesh.slang
$SLANG $SLANG_FLAGS -o "$OUTDIR/scene_render.spv" src/internal/renderer/scene_render.slang
$SLANG $SLANG_FLAGS -o "$OUTDIR/scene_update.spv" src/internal/renderer/scene_update.slang
$SLANG $SLANG_FLAGS -o "$OUTDIR/test.spv" src/internal/renderer/test.slang
