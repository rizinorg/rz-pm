project('simple_project', 'c')

rz_core_dep = dependency('rz_core')
plugins_dir = get_option('rizin_plugdir')
message('Plugins install directory: ' + plugins_dir)

library(
  'plugin',
  ['plugin.c'],
  dependencies: [rz_core_dep],
  install: true,
  install_dir: plugins_dir,
)
