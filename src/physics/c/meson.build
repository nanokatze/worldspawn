physics_src = files(
  'physics.cpp',
  'shape.cpp',
)

physics = static_library(
  'physics',
  physics_src,
  dependencies : [math_dep, jolt_dep])

go_env.append('CGO_LDFLAGS', '"-L@0@"'.format(meson.current_build_dir()), separator : ' ')
