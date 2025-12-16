#!/usr/bin/env bash

set -e

OUTDIR=shaders

SLANG=../slang/build/Release/bin/slangc
SLANG_FLAGS="-target spirv -profile spirv_1_6 -fvk-use-entrypoint-name -fvk-use-c-layout -matrix-layout-row-major -capability vk_mem_model"

$SLANG $SLANG_FLAGS -o "$OUTDIR/mesh.spv" src/internal/pathtracer/mesh.slang
$SLANG $SLANG_FLAGS -o "$OUTDIR/scene_render.spv" src/internal/pathtracer/scene_render.slang
$SLANG $SLANG_FLAGS -o "$OUTDIR/scene_update.spv" src/internal/pathtracer/scene_update.slang
