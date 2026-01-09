#!/usr/bin/env bash

meson install -C build_linux_amd64 --tags client --destdir=$HOME/deck
