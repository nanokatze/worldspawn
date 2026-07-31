#!/usr/bin/env bash

# TODO: eventually we want an equivalent of a "Play" button in ue/godot/etc

set -e

game=deathmatch

# TODO: clean up repo_dir
repo_dir=$(dirname -- "$BASH_SOURCE")/..

# TODO: kill this once we can use system go again.
PATH="$HOME/code/go/bin:$PATH"

GOOS=$(go env GOOS)
GOARCH=$(go env GOARCH)

src_dir="$repo_dir"
out_dir="$repo_dir/$game/bin/$GOOS-$GOARCH"

go build -C $src_dir -o "$out_dir/client" "worldspawn/$game/client" &
go build -C $src_dir -o "$out_dir/server" "worldspawn/$game/server" &

wait
