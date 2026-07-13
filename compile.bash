#!/usr/bin/env bash

# TODO: eventually we want an equivalent of a "Play" button in ue/godot/etc

set -e

PATH="$HOME/code/go/bin:$PATH"

src_dir=. # TODO: should be where this script is probs
out_dir=. # idk

go build -C $src_dir -o $out_dir/bin/client worldspawn/deathmatch/client &
go build -C $src_dir -o $out_dir/bin/server worldspawn/deathmatch/server &
wait
