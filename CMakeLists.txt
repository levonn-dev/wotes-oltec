cmake_minimum_required(VERSION 3.28)
project(wotes_oltec C ASM)

if(NOT CMAKE_SYSTEM_PROCESSOR STREQUAL "m68k")
  message(FATAL_ERROR "Configure with 'cmake --preset m68k' or 'task build' (see README.md).")
endif()

set(CLEAN_ROM "${CMAKE_SOURCE_DIR}/Dungeons & Dragons - Warriors of the Eternal Sun (USA, Europe).md" CACHE FILEPATH "Clean ROM image")
if(NOT EXISTS "${CLEAN_ROM}")
  message(FATAL_ERROR "Clean ROM not found at ${CLEAN_ROM}; pass -DCLEAN_ROM=<path> (see README.md).")
endif()
# src/oltec.s reads original tables and cell arrays out of the clean ROM with .incbin,
# through this fixed name so that no source file carries the ROM's file name or path.
file(COPY_FILE "${CLEAN_ROM}" "${CMAKE_BINARY_DIR}/clean.md" ONLY_IF_DIFFERENT)
# A clean ROM that changes triggers a reconfigure, so the copy is refreshed with it.
set_property(DIRECTORY APPEND PROPERTY CMAKE_CONFIGURE_DEPENDS "${CLEAN_ROM}")
set(CLEAN_ROM_SHA1 9135f7fda03ef7da92dfade9c0df75808214f693)
set(ROM_SIZE 2097152) # 2MB: half the cart limit, same as Chili's patch
# ROM offsets of every .hook_ section in hooks/hooks.s; mdrom fails the inject if one drifts.
set(HOOKS 000300 0022E1 002316 0008D2 00BB90 0097C6 009D6E 010424 01055C 0D70A8 012F38 012D36 012CC8 012D00 013132 004F8D 00509B 0050DD 012CC5 00D30C 00E282 00B3A4)

include(cmake/mdrom.cmake)
include(cmake/wotes-tiled.cmake)

# mdrom measures the ROM and writes the ROM_FREE memory region the linker script includes.
add_custom_command(
  OUTPUT ${CMAKE_BINARY_DIR}/free.ld
  COMMAND ${MDROM} free "${CLEAN_ROM}" --size ${ROM_SIZE} --ld ${CMAKE_BINARY_DIR}/free.ld
  DEPENDS "${CLEAN_ROM}"
  COMMENT "Measuring free ROM space"
  VERBATIM)
add_custom_target(free_ld DEPENDS ${CMAKE_BINARY_DIR}/free.ld)

add_executable(patch.elf
  src/poc.c
  src/rt/div.c
  src/rt/mul.s
  src/oltec.s
  hooks/hooks.s)
add_dependencies(patch.elf free_ld)
target_include_directories(patch.elf PRIVATE game)
set_target_properties(patch.elf PROPERTIES
  LINK_DEPENDS "${CMAKE_SOURCE_DIR}/link/rom.ld;${CMAKE_SOURCE_DIR}/game/wotes.ld;${CMAKE_BINARY_DIR}/free.ld")
# -L before -T: the linker resolves INCLUDE against -L paths seen before the script.
# -Xlinker keeps each path one argument, so a checkout under a path with commas or spaces still links.
# One SHELL: entry per pair stops CMake's option de-duplication from folding the -Xlinker tokens.
target_link_options(patch.elf PRIVATE
  "SHELL:-Xlinker -L -Xlinker \"${CMAKE_BINARY_DIR}\""
  "SHELL:-Xlinker -L -Xlinker \"${CMAKE_SOURCE_DIR}/game\""
  "SHELL:-Xlinker -T -Xlinker \"${CMAKE_SOURCE_DIR}/link/rom.ld\""
  "SHELL:-Xlinker -Map -Xlinker \"${CMAKE_BINARY_DIR}/patch.map\"")
foreach(hook IN LISTS HOOKS)
  target_link_options(patch.elf PRIVATE -Wl,--section-start=.hook_${hook}=0x${hook})
endforeach()
# .incbin resolves its file against the assembler's include path; -Xassembler keeps the
# path one argument, as -Xlinker does above.
target_compile_options(patch.elf PRIVATE "SHELL:-Xassembler -I -Xassembler \"${CMAKE_BINARY_DIR}\"")
set_source_files_properties(src/oltec.s PROPERTIES OBJECT_DEPENDS "${CMAKE_BINARY_DIR}/clean.md")
add_custom_command(TARGET patch.elf POST_BUILD
  COMMAND ${CMAKE_SOURCE_DIR}/scripts/check-68000.sh $<TARGET_FILE:patch.elf>
  COMMAND ${CMAKE_SOURCE_DIR}/scripts/check-hooks.sh $<TARGET_FILE:patch.elf> ${HOOKS}
  VERBATIM)

# The land's tiles stay below 760 (0x2F8): the walking monsters' frames load from tile 0x319, and
# the valley's lava check at 0x6822 burns the party on tiles 0x2F8 to 0x2FC wherever it stands.
wotes_tiled_map(oltec 760)

# Host-side check of the division helpers; scripts/check-rt.sh holds the compile line
# so `task check:host` can run the same test without CMake.
add_custom_target(check-rt ALL
  COMMAND ${CMAKE_SOURCE_DIR}/scripts/check-rt.sh ${CMAKE_BINARY_DIR}/host
  COMMENT "Checking division helpers against native arithmetic"
  VERBATIM)

# Inject the ELF into the clean ROM, then write the BPS patch.
add_custom_command(
  OUTPUT ${CMAKE_BINARY_DIR}/patched.md
  COMMAND ${MDROM} inject --base "${CLEAN_ROM}" --elf $<TARGET_FILE:patch.elf>
          --out ${CMAKE_BINARY_DIR}/patched.md --size ${ROM_SIZE} --expect-sha1 ${CLEAN_ROM_SHA1}
  DEPENDS patch.elf "${CLEAN_ROM}"
  COMMENT "Injecting patch.elf into the ROM"
  VERBATIM)
add_custom_command(
  OUTPUT ${CMAKE_BINARY_DIR}/patch.bps
  COMMAND ${MDROM} bps create --base "${CLEAN_ROM}" --target ${CMAKE_BINARY_DIR}/patched.md
          --out ${CMAKE_BINARY_DIR}/patch.bps
  DEPENDS ${CMAKE_BINARY_DIR}/patched.md
  COMMENT "Writing patch.bps"
  VERBATIM)
add_custom_target(rom ALL DEPENDS ${CMAKE_BINARY_DIR}/patch.bps)

# Emulator targets go through scripts/blastem.sh, which runs the bootstrapped nightly.
add_custom_target(run
  COMMAND ${CMAKE_SOURCE_DIR}/scripts/blastem.sh ${CMAKE_BINARY_DIR}/patched.md
  DEPENDS ${CMAKE_BINARY_DIR}/patched.md
  COMMENT "Running the patched ROM in BlastEm"
  USES_TERMINAL VERBATIM)

add_custom_target(check-target
  COMMAND ${CMAKE_SOURCE_DIR}/scripts/check-target.sh $<TARGET_FILE:patch.elf> ${CMAKE_BINARY_DIR}/patched.md 356872
  DEPENDS ${CMAKE_BINARY_DIR}/patched.md
  COMMENT "Checking poc_init on the target under GDB"
  USES_TERMINAL VERBATIM)
