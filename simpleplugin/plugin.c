// SPDX-FileCopyrightText: 2018-2023 Giovanni Dante Grazioli <deroad@libero.it>
// SPDX-FileCopyrightText: 2025 Eyad Issa <eyadlorenzo@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

#include <rz_analysis.h>
#include <rz_cmd.h>
#include <rz_cons.h>
#include <rz_core.h>
#include <rz_lib.h>
#include <rz_types.h>

static bool rz_cmd_init(RzCore *core) {
  // no-op plugin, does nothing
  RZ_LOG_INFO("Initializing simple plugin\n");
  return true;
}

static bool rz_cmd_fini(RzCore *core) {
  // no-op plugin, does nothing
  RZ_LOG_INFO("Finalizing simple plugin\n");
  return true;
}

RzCorePlugin rz_core_plugin_example = {
    .name = "simple-plugin",
    .desc = "A simple no-op plugin for Rizin",
    .license = "BSD-3-Clause",
    .init = rz_cmd_init,
    .fini = rz_cmd_fini,
};

#ifdef _MSC_VER
#define _RZ_API __declspec(dllexport)
#else
#define _RZ_API __attribute__((visibility("default")))
#endif

#ifndef CORELIB
_RZ_API RzLibStruct rizin_plugin = {
    .type = RZ_LIB_TYPE_CORE,
    .data = &rz_core_plugin_example,
    .version = RZ_VERSION,
};
#endif
