# Oltec land patch for Warriors of the Eternal Sun

C and 68000 assembly compiled with the m68k-linux-gnu GCC, injected into the clean ROM with [mdrom](https://github.com/levonn-dev/mdrom), and shipped as a BPS patch.

## Prerequisites

- Ubuntu 24.04, natively or under WSL2 (the emulator window needs WSLg or an X display).
- [Task](https://taskfile.dev/installation/) 3.
- The cross compiler, debugger, and helpers:

  ```
  sudo apt install cmake ninja-build gcc gcc-m68k-linux-gnu gdb-multiarch binutils shellcheck curl
  ```

- The clean ROM, `Dungeons & Dragons - Warriors of the Eternal Sun (USA, Europe).md`, SHA-1 `9135f7fda03ef7da92dfade9c0df75808214f693`. It is not part of the repository.

## Quick start

```
git clone https://github.com/levonn-dev/wotes-oltec.git
cd wotes-oltec
task bootstrap
cp /path/to/'Dungeons & Dragons - Warriors of the Eternal Sun (USA, Europe).md' .
task build
```

`task bootstrap` downloads the BlastEm nightly and the mdrom release into `tools/`, checks the apt tools, and prints the apt line above if one is missing. It writes nothing outside the repository and never uses sudo. A ROM kept elsewhere is named with `task configure -- -DCLEAN_ROM=<path>`; options after `--` reach CMake even when the tree is already configured.

## Tasks

`task` with no arguments lists them.

| Task | Does |
|---|---|
| `bootstrap` | BlastEm and mdrom into `tools/`; checks the apt tools and the ROM |
| `build` | Configures `build/m68k/` with the `m68k` preset if needed, then links, runs the 68000 and hook checks, injects, writes `patch.bps`, and runs the host helper test |
| `run` | The patched ROM in BlastEm |
| `check` | `build`, then the scripted GDB check of `poc_init` on the target |
| `check:host` | Helper test plus shellcheck; no cross compiler or ROM needed (what CI runs) |
| `check:target` | The on-target GDB check alone |
| `debug` | Interactive `gdb-multiarch` against BlastEm, paused at the reset vector |
| `configure` | The CMake configure alone; extra options after `--` |
| `clean` | Removes `build/` and the Task cache (keeps `tools/`) |

## Outputs

`build/m68k/patch.elf` (the linked patch with DWARF), `build/m68k/patched.md` (the 2MB patched image), `build/m68k/patch.bps` (the patch to distribute).

## Layout

- `src/`, `hooks/`, `game/`, `link/`: patch code, hook sections, game symbols, linker script
- `cmake/`: toolchain file and the mdrom lookup
- `scripts/`: bootstrap, the BlastEm launcher, and the build-time checks
- `tests/`: host-side test of the runtime helpers
- `docs/`: `debugging.md` for GDB and IDEA
- `tools/`: created by bootstrap, not committed
- `.github/`: the CI workflow, which runs `task check:host`, and `CODEOWNERS`

## License

MIT, see `LICENSE`. The game and its ROM belong to their rights holders; this repository contains only the patch source.
