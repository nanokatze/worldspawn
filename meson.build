project('worldspawn', ['c', 'cpp'],
  default_options : ['c_std=gnu2x', 'cpp_std=c++20'])

cmake = import('cmake')
fs = import('fs')

python = import('python').find_installation('python3')

cc = meson.get_compiler('c')
cxx = meson.get_compiler('c')

# TODO: ensure that cc and cxx are clang

cpp = find_program('cpp')

# https://github.com/mesonbuild/meson/issues/123
go = find_program('go')
go_env = environment()
go_env.append('CC', cc.cmd_array()[0])

# Always use SDL from the environment

math_dep = cc.find_library('m', required : false)

go_src_dir = meson.current_source_dir() # TODO: rename

subdir('deathmatch/client')
subdir('deathmatch/server')
