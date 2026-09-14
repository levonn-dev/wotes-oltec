# Debugging the patch

## Prerequisites

`task bootstrap` installs BlastEm nightly into `tools/blastem/` and sets `gl off` in its `default.cfg`, which BlastEm reads when you have no `~/.config/blastem/blastem.cfg` of your own; if you do, it must say `gl off` too. Under WSLg the OpenGL renderer cannot set the swap interval, and its warning lands on the GDB pipe and breaks the connection. The Ubuntu package (0.6.3.4) does not work either: its GDB stub answers GDB's `vMustReplyEmpty` probe incorrectly and exits, so GDB 15 cannot use it. Every launch goes through `scripts/blastem.sh`; a BlastEm built elsewhere is used by setting `BLASTEM=/path/to/blastem`.

`gdb-multiarch`, from the apt line in the README, knows the m68k target.

## Command line

`task debug` builds, then opens this session:

```
gdb-multiarch build/m68k/patch.elf -ex 'set architecture m68k' \
    -ex 'target remote | scripts/blastem.sh build/m68k/patched.md -D'
```

The CPU starts paused at the reset vector (`0x200`), so breakpoints set now catch startup code:

```
(gdb) break poc_init
(gdb) continue
(gdb) next
(gdb) finish        # or: break poc_ret, continue, p/d $d0
```

`build/m68k/patch.elf` carries DWARF for `src/` and `hooks/`, so `list`, `step`, and `info locals` work in C. Game code outside the patch shows as raw addresses; `x/10i $pc` disassembles it.

## Scripted check

`task check:target` runs `scripts/check-target.sh`, which breaks on `poc_init` and on `poc_ret` and asserts `d0` is `356872`. `task check` builds first.

## IDEA Ultimate

Open the folder natively (File, Open, the `\\wsl.localhost\<distro>\...` path, with the advanced setting "Open WSL projects natively as local projects" on) and build and debug through a WSL toolchain. CMake, the compiler, GDB, and BlastEm all run inside the distro; the debugger UI runs in the local IDE, so the Memory View and Disassembly panels work.

IDEA's Native Build Tools plugin ships the WSL toolchain code but hides the kind in the toolchain dialog. [idea-wsl-toolchains](https://github.com/levonn-dev/idea-wsl-toolchains) unhides it: build it as its README says, install the plugin folder into IDEA's user plugins directory, and restart. Rebuild it when IDEA or the Native Build Tools plugin changes major version.

One-time setup, verified with IDEA 2026.2.2:

1. Settings, Build Execution Deployment, Toolchains: add a WSL toolchain for the distro and move it to the top so it is the default. It detects `cmake`, `ninja`, and `m68k-linux-gnu-gcc` on its own.
2. Settings, Build Execution Deployment, CMake: enable the `m68k` profile IDEA created from `CMakePresets.json` and disable any other. It builds into `build/m68k/`, the same directory as the command line. After the first configure every reload warns that `CMAKE_TOOLCHAIN_FILE` was not used; CMake reads it once and caches it, so ignore the warning.
3. Settings, Build Execution Deployment, Debugger, Debug Profiles: the committed `gdb-multiarch` profile in `.idea/debug-profiles.xml` runs `scripts/gdb.sh`, a wrapper for `gdb-multiarch`; select it in the main toolbar's debug profile selector. The toolchain's own Debugger field is deprecated and ignored by Remote Debug.
4. Build, Build Project. This is the `all` target. IDEA also generates one CMake Application configuration per target, and the ones for `run`, `rom`, and `check-target` are committed; Run is meaningless for them, but Build on `run` or `check-target` executes those targets.
5. Set a breakpoint in `src/poc.c` and debug the committed `BlastEm` configuration (`.idea/runConfigurations/BlastEm.xml`, a C/C++ Remote Debug with `'target remote' args` `| scripts/blastem.sh build/m68k/patched.md -D` and symbol file `$PROJECT_DIR$/build/m68k/patch.elf`). BlastEm opens through WSLg and the session starts paused at the reset vector; press Resume to reach the breakpoint. The Memory View is empty until you enter an address in its Go to address field (`0xFF0000` for work RAM, `$sp`, `&name`); Ctrl+Enter on a variable jumps there. If the session fails to start because GDB cannot run `scripts/blastem.sh`, GDB did not launch in the project directory; change the `'target remote' args` to `| $PROJECT_DIR$/scripts/blastem.sh $PROJECT_DIR$/build/m68k/patched.md -D`.

## Emulator only

`task run` launches BlastEm on the patched ROM without a debugger. Start a new game: the class list on the first character screen reads `WARRIOR` where the original reads `FIGHTER`.
