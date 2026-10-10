# Debugging the patch

GDB drives BlastEm's GDB stub over a pipe: `gdb-multiarch` loads `build/m68k/patch.elf` for symbols and speaks the
remote protocol to a BlastEm started with `-D`. The command line, the scripted check, and the IDE setup below are the
same connection with a different front end. Nothing here is WSL-only except where a section says so.

## Prerequisites

`gdb-multiarch`, from the apt line in the README, knows the m68k target.

`task bootstrap` installs the pinned BlastEm nightly into `tools/blastem/` and sets `gl off` in its `default.cfg`. Every
launch goes through `scripts/blastem.sh`; a BlastEm built elsewhere is used by setting `BLASTEM=/path/to/blastem`. The
Ubuntu package (0.6.3.4) does not work: its GDB stub answers GDB's `vMustReplyEmpty` probe incorrectly and exits, so GDB
15 cannot use it.

BlastEm reads `default.cfg` beside the binary only when you have no `~/.config/blastem/blastem.cfg` of your own.

### For WSL

Under WSLg the OpenGL renderer cannot set the swap interval, and its warning lands on the GDB pipe and breaks the
connection. `gl off` is required; bootstrap sets it in `default.cfg`, and a personal `blastem.cfg` must repeat it.

### For native Linux

`gl off` is not required, but bootstrap sets it and it does no harm. BlastEm opens on your X or Wayland display.

## Command line

`task debug` builds, then opens this session on the clean ROM's build; `task debug BASE=chili-v1` or `BASE=chili-v2`
opens it on that build, with that tree's `patch.elf` and its image, `build/m68k-chili-v1/patched-chili-v1.md` or
`build/m68k-chili-v2/patched-chili-v2.md`:

```
gdb-multiarch build/m68k/patch.elf -ex 'set architecture m68k' \
    -ex 'target remote | scripts/blastem.sh build/m68k/patched.md -D'
```

The CPU starts paused at the reset vector (`0x200`), so breakpoints set now catch the boot. The intro's first picture
fades in through the patch's `fade_in_hook` before any button is pressed:

```
(gdb) break fade_in_hook
(gdb) continue
(gdb) p/x $a0       # 0x64dba, the palette the intro fades to
(gdb) stepi
```

`build/m68k/patch.elf` carries DWARF for `src/` and `hooks/`, so `list`, `step` and `stepi` follow the patch's
assembly. Game code outside the patch shows as raw addresses; `x/10i $pc` disassembles it.

## Scripted check

`task check:target` runs `scripts/check-target.sh` on each of the three builds, which breaks on `fade_in_hook` and
asserts `a0` is `0x64dba`, the palette of the intro's first picture. `task check` builds first.

## IDEA Ultimate

Build and debug through the IDE: CMake, the compiler, GDB, and BlastEm run through a toolchain, while the debugger UI
runs in the IDE, so the Memory View and Disassembly panels work.

The one-time setup below is verified with IDEA 2026.2.2. Only step 1, the toolchain, differs between WSL and native
Linux; do the matching subsection first, then work through the common steps.

### For WSL

Open the folder natively (File, Open, the `\\wsl.localhost\<distro>\...` path, with the advanced setting "Open WSL
projects natively as local projects" on). CMake, the compiler, GDB, and BlastEm all run inside the distro.

Step 1 is: Settings, Build Execution Deployment, Toolchains: add a WSL toolchain for the distro and move it to the top
so it is the default. It detects `cmake`, `ninja`, and `m68k-linux-gnu-gcc` on its own.

IDEA's Native Build Tools plugin ships the WSL toolchain code but hides the kind in the toolchain
dialog. [idea-wsl-toolchains](https://github.com/levonn-dev/idea-wsl-toolchains) unhides it: build it as its README
says, install the plugin folder into IDEA's user plugins directory, and restart. Rebuild it when IDEA or the Native
Build Tools plugin changes major version.

### For native Linux

Open the folder directly.

Step 1 is: Settings, Build Execution Deployment, Toolchains: the default (System) toolchain detects `cmake`, `ninja`,
and `m68k-linux-gnu-gcc` on your PATH. No extra plugin is needed.

### Common steps

2. Settings, Build Execution Deployment, CMake: enable the `m68k` profile IDEA created from `CMakePresets.json` and
   disable any other. It builds into `build/m68k/`, the same directory as the command line. After the first configure
   every reload warns that `CMAKE_TOOLCHAIN_FILE` was not used; CMake reads it once and caches it, so ignore the
   warning. To debug a build for planetchili's patch, enable `m68k-chili-v1` or `m68k-chili-v2` beside `m68k`, copy the
   `BlastEm` configuration (Run, Edit Configurations, Copy Configuration, with Store as project file unchecked) and
   point the copy's two paths at that tree's `patch.elf` and its image, `patched-chili-v1.md` or `patched-chili-v2.md`.
3. Settings, Build Execution Deployment, Debugger, Debug Profiles: the committed `gdb-multiarch` profile in
   `.idea/debug-profiles.xml` runs `scripts/gdb.sh`, a wrapper for `gdb-multiarch`; select it in the main toolbar's
   debug profile selector. The toolchain's own Debugger field is deprecated and ignored by Remote Debug.
4. Build, Build Project. This is the `all` target. IDEA also generates one CMake Application configuration per target,
   and the ones for `run`, `rom`, and `check-target` are committed; Run is meaningless for them, but Build on `run` or
   `check-target` executes those targets.
5. Add a symbolic breakpoint on `fade_in_hook` (Run, View Breakpoints, the + button) and debug the committed
   `BlastEm` configuration (`.idea/runConfigurations/BlastEm.xml`, a C/C++ Remote Debug with `'target remote' args`
   `| scripts/blastem.sh build/m68k/patched.md -D` and symbol file `$PROJECT_DIR$/build/m68k/patch.elf`). BlastEm opens
   and the session starts paused at the reset vector; press Resume to reach the breakpoint. The Memory View is empty
   until you enter an address in its Go to address field (`0xFF0000` for work RAM, `$sp`, `&name`); Ctrl+Enter on a
   variable jumps there. If the session fails to start because GDB cannot run `scripts/blastem.sh`, GDB did not launch
   in the project directory; change the `'target remote' args` to
   `| $PROJECT_DIR$/scripts/blastem.sh $PROJECT_DIR$/build/m68k/patched.md -D`.

## Emulator only

`task run` launches BlastEm on the patched ROM without a debugger; `task run BASE=chili-v1` or `BASE=chili-v2` launches
that build, whose image has a name of its own, so BlastEm keeps its saves apart from the clean build's.
