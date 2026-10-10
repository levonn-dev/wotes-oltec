# Bare-metal 68000 toolchain: the Ubuntu m68k-linux-gnu GCC drives the assembler and the linker.
set(CMAKE_SYSTEM_NAME Generic)
set(CMAKE_SYSTEM_PROCESSOR m68k)
set(CMAKE_ASM_COMPILER m68k-linux-gnu-gcc)
set(CMAKE_TRY_COMPILE_TARGET_TYPE STATIC_LIBRARY)

# The assembler defaults to the 68020; -m68000 is mandatory. -g never changes ROM bytes.
set(CMAKE_ASM_FLAGS_INIT "-m68000 -g")
# A project with only ASM enabled never seeds the linker flags from an _INIT value, so the cache entry is set here.
set(CMAKE_EXE_LINKER_FLAGS "-m68000 -nostdlib -static -Wl,--build-id=none -Wl,-z,noexecstack -Wl,--entry=0"
  CACHE STRING "Flags used by the linker during all build types.")

# Build types must not alter code generation; the flags above are the whole story.
foreach(cfg DEBUG RELEASE RELWITHDEBINFO MINSIZEREL)
  set(CMAKE_ASM_FLAGS_${cfg}_INIT "")
endforeach()
