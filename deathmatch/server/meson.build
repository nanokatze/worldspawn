custom_target(
  'server',
  env : go_env,
  depends : [physics],
  output : 'deathmatch-server',
  command : [go, '-C', go_src_dir, 'build', '-o', meson.global_build_root() / '@OUTPUT@', meson.current_source_dir()],
  install : true,
  install_dir : 'bin',
  build_always_stale : true)
