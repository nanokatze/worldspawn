custom_target(
  'client',
  env : go_env,
  depends : [physics],
  output : 'client',
  command : [go, '-C', go_src_dir, 'build', '-o', meson.global_build_root() / '@OUTPUT@', meson.current_source_dir()],
  install : true,
  install_dir : 'bin',
  install_tag : 'client',
  build_always_stale : true)
