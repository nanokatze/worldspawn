#!/usr/bin/env bash

set -e

OUTDIR="$HOME/deck/home/deck/worldspawn/shaders/"

SLANGC=../slang/build/Release/bin/slangc
SLANGC_FLAGS="-target spirv -profile spirv_1_6 -fvk-use-entrypoint-name -force-glsl-scalar-layout -matrix-layout-row-major -capability vk_mem_model"

$SLANGC $SLANGC_FLAGS -o "$OUTDIR/mesh.spv" src/renderer/mesh.slang
$SLANGC $SLANGC_FLAGS -o "$OUTDIR/renderer.spv" src/renderer/renderer.slang
$SLANGC $SLANGC_FLAGS -o "$OUTDIR/scene.spv" src/renderer/scene.slang
$SLANGC $SLANGC_FLAGS -o "$OUTDIR/test.spv" src/renderer/test.slang
