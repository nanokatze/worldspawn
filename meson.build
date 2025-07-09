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
go_env.append('GOEXPERIMENT', 'rangefunc')

# Always use SDL from the environment

math_dep = cc.find_library('m', required : false)

vulkan_headers_dep = dependency('Vulkan-Headers')

jolt_options = cmake.subproject_options()
jolt_options.add_cmake_defines({
  'CROSS_PLATFORM_DETERMINISTIC': true,
  'DOUBLE_PRECISION': true,
  'USE_ASSERTS': true,
})
jolt_dep = cmake.subproject('JoltPhysics', options : jolt_options).dependency('Jolt')
go_env.append('CGO_LDFLAGS', '"-L@0@"'.format(meson.current_build_dir() / 'subprojects/JoltPhysics'), separator : ' ')

subdir('src')
