cmake_minimum_required(VERSION 3.28)
project(wotes_oltec ASM)

if(NOT CMAKE_SYSTEM_PROCESSOR STREQUAL "m68k")
  message(FATAL_ERROR "Configure with 'cmake --preset m68k' or 'task build' (see README.md).")
endif()

set(CLEAN_ROM "${CMAKE_SOURCE_DIR}/Dungeons & Dragons - Warriors of the Eternal Sun (USA, Europe).md" CACHE FILEPATH "Clean ROM image")
if(NOT EXISTS "${CLEAN_ROM}")
  message(FATAL_ERROR "Clean ROM not found at ${CLEAN_ROM}; pass -DCLEAN_ROM=<path> (see README.md).")
endif()
# A clean ROM that changes triggers a reconfigure, so the base is made again from it.
set_property(DIRECTORY APPEND PROPERTY CMAKE_CONFIGURE_DEPENDS "${CLEAN_ROM}")
set(CLEAN_ROM_SHA1 9135f7fda03ef7da92dfade9c0df75808214f693)

# The ROM the patch is built on: the clean ROM, or the clean ROM with planetchili's V1.0 or V2.0
# patch, which bootstrap fetches into tools/chili. Each preset builds one base in its own tree.
set(BASE clean CACHE STRING "Base ROM: clean, chili-v1 or chili-v2")
set_property(CACHE BASE PROPERTY STRINGS clean chili-v1 chili-v2)
if(BASE STREQUAL "clean")
  set(BASE_SHA1 ${CLEAN_ROM_SHA1})
  set(BPS_NAME oltec)
  set(IMAGE_NAME patched)
elseif(BASE STREQUAL "chili-v1")
  set(BASE_PATCH "${CMAKE_SOURCE_DIR}/tools/chili/V1.0.bps")
  set(BASE_SHA1 73205529c6d28bed15120eaae9f77707418b6dc3)
  set(BPS_NAME oltec-chili-v1)
  set(IMAGE_NAME patched-chili-v1)
elseif(BASE STREQUAL "chili-v2")
  set(BASE_PATCH "${CMAKE_SOURCE_DIR}/tools/chili/V2.0.bps")
  set(BASE_SHA1 2380b201b872f7bd2c66f2a4d82df86709332d55)
  set(BPS_NAME oltec-chili-v2)
  set(IMAGE_NAME patched-chili-v2)
else()
  message(FATAL_ERROR "BASE is '${BASE}'; it must be clean, chili-v1 or chili-v2.")
endif()
set(ROM_SIZE 2097152) # 2MB: half the cart limit, same as Chili's patch
# ROM offsets of every .hook_ section in hooks/hooks.s; mdrom fails the inject if one drifts.
set(HOOKS 0008D2 00BB90 0097C6 009D6E 0103F0 010482 010CEE 0113AC 010EEE 0D70A8 012F38 012D36 012CC8 012D00 013132 004F8D 00509B 0050DD 012CC5 00D30C 00E282 00B3A4 018452 00C504 00CEC2 01044A 009862)

include(cmake/mdrom.cmake)
include(cmake/wotes-tiled.cmake)

# The sources and the converted maps read the game's tables out of the base ROM with .incbin,
# through this fixed name so that no source file carries the ROM's file name or path.
if(BASE STREQUAL "clean")
  file(COPY_FILE "${CLEAN_ROM}" "${CMAKE_BINARY_DIR}/base.md" ONLY_IF_DIFFERENT)
else()
  if(NOT EXISTS "${BASE_PATCH}")
    message(FATAL_ERROR "${BASE_PATCH} not found; run 'task bootstrap' to fetch planetchili's patches.")
  endif()
  set_property(DIRECTORY APPEND PROPERTY CMAKE_CONFIGURE_DEPENDS "${BASE_PATCH}")
  execute_process(
    COMMAND ${MDROM} bps apply --base "${CLEAN_ROM}" --patch "${BASE_PATCH}" --out "${CMAKE_BINARY_DIR}/base.md.part"
    RESULT_VARIABLE _apply OUTPUT_QUIET)
  if(NOT _apply EQUAL 0)
    message(FATAL_ERROR "mdrom could not apply ${BASE_PATCH} to ${CLEAN_ROM}.")
  endif()
  # Copied only when the bytes differ, so a reconfigure does not rebuild what reads the base.
  file(COPY_FILE "${CMAKE_BINARY_DIR}/base.md.part" "${CMAKE_BINARY_DIR}/base.md" ONLY_IF_DIFFERENT)
  file(REMOVE "${CMAKE_BINARY_DIR}/base.md.part")
endif()

# mdrom measures the ROM and writes the ROM_FREE memory region the linker script includes.
add_custom_command(
  OUTPUT ${CMAKE_BINARY_DIR}/free.ld
  COMMAND ${MDROM} free "${CMAKE_BINARY_DIR}/base.md" --size ${ROM_SIZE} --ld ${CMAKE_BINARY_DIR}/free.ld
  DEPENDS "${CMAKE_BINARY_DIR}/base.md"
  COMMENT "Measuring free ROM space"
  VERBATIM)
add_custom_target(free_ld DEPENDS ${CMAKE_BINARY_DIR}/free.ld)

add_executable(patch.elf
  src/oltec.s
  hooks/hooks.s)
add_dependencies(patch.elf free_ld)
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
set_source_files_properties(src/oltec.s hooks/hooks.s PROPERTIES OBJECT_DEPENDS "${CMAKE_BINARY_DIR}/base.md")
add_custom_command(TARGET patch.elf POST_BUILD
  COMMAND ${CMAKE_SOURCE_DIR}/scripts/check-68000.sh $<TARGET_FILE:patch.elf>
  COMMAND ${CMAKE_SOURCE_DIR}/scripts/check-hooks.sh $<TARGET_FILE:patch.elf> ${HOOKS}
  VERBATIM)

# The land's tiles stay below 760 (0x2F8): the walking monsters' frames load from tile 0x319, and
# the valley's lava check at 0x6822 burns the party on tiles 0x2F8 to 0x2FC wherever it stands.
wotes_tiled_map(oltec 760)

# Inject the ELF into the base ROM, then write the BPS patch that turns the base into the result.
# Each base's image has a name of its own: BlastEm keeps saves by it, and V2.0's save RAM has another layout.
add_custom_command(
  OUTPUT ${CMAKE_BINARY_DIR}/${IMAGE_NAME}.md
  COMMAND ${MDROM} inject --base "${CMAKE_BINARY_DIR}/base.md" --elf $<TARGET_FILE:patch.elf>
          --out ${CMAKE_BINARY_DIR}/${IMAGE_NAME}.md --size ${ROM_SIZE} --expect-sha1 ${BASE_SHA1}
  DEPENDS patch.elf "${CMAKE_BINARY_DIR}/base.md"
  COMMENT "Injecting patch.elf into the ROM"
  VERBATIM)
add_custom_command(
  OUTPUT ${CMAKE_BINARY_DIR}/${BPS_NAME}.bps
  COMMAND ${MDROM} bps create --base "${CMAKE_BINARY_DIR}/base.md" --target ${CMAKE_BINARY_DIR}/${IMAGE_NAME}.md
          --out ${CMAKE_BINARY_DIR}/${BPS_NAME}.bps
  DEPENDS ${CMAKE_BINARY_DIR}/${IMAGE_NAME}.md
  COMMENT "Writing ${BPS_NAME}.bps"
  VERBATIM)
add_custom_target(rom ALL DEPENDS ${CMAKE_BINARY_DIR}/${BPS_NAME}.bps)

# Emulator targets go through scripts/blastem.sh, which runs the bootstrapped nightly.
add_custom_target(run
  COMMAND ${CMAKE_SOURCE_DIR}/scripts/blastem.sh ${CMAKE_BINARY_DIR}/${IMAGE_NAME}.md
  DEPENDS ${CMAKE_BINARY_DIR}/${IMAGE_NAME}.md
  COMMENT "Running the patched ROM in BlastEm"
  USES_TERMINAL VERBATIM)

add_custom_target(check-target
  COMMAND ${CMAKE_SOURCE_DIR}/scripts/check-target.sh $<TARGET_FILE:patch.elf> ${CMAKE_BINARY_DIR}/${IMAGE_NAME}.md 0x64dba
  DEPENDS ${CMAKE_BINARY_DIR}/${IMAGE_NAME}.md
  COMMENT "Checking fade_in_hook on the target under GDB"
  USES_TERMINAL VERBATIM)
