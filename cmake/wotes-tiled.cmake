# Builds wotes-tiled into tools/wotes-tiled when Go is found; a wired map requires Go to configure.
# Go 1.26 or later is required; goenv or https://go.dev/dl/ provide it, apt's is too old.
find_program(WOTES_GO go)
set(WOTES_TILED "${CMAKE_SOURCE_DIR}/tools/wotes-tiled/wotes-tiled")
if(WOTES_GO)
  file(GLOB_RECURSE _wotes_tiled_sources CONFIGURE_DEPENDS
    "${CMAKE_SOURCE_DIR}/wotes-tiled/*.go" "${CMAKE_SOURCE_DIR}/wotes-tiled/go.mod")
  add_custom_command(
    OUTPUT "${WOTES_TILED}"
    COMMAND "${WOTES_GO}" build -o "${WOTES_TILED}" .
    WORKING_DIRECTORY "${CMAKE_SOURCE_DIR}/wotes-tiled"
    DEPENDS ${_wotes_tiled_sources}
    COMMENT "Building wotes-tiled"
    VERBATIM)
  add_custom_target(wotes-tiled DEPENDS "${WOTES_TILED}")
endif()

# wotes_tiled_map(NAME), called after add_executable(patch.elf ...), assembles maps/NAME.tmj into
# patch.elf; the depfile names every Tiled file and PNG the map pulls in.
function(wotes_tiled_map name)
  if(NOT WOTES_GO)
    message(FATAL_ERROR "go 1.26 or later is required to convert maps/${name}.tmj: install it with goenv or from https://go.dev/dl/")
  endif()
  set(_out "${CMAKE_BINARY_DIR}/gen/${name}-map.s")
  set(_dep "${CMAKE_BINARY_DIR}/gen/${name}-map.d")
  add_custom_command(
    OUTPUT "${_out}"
    COMMAND "${WOTES_TILED}" convert "maps/${name}.tmj"
            --rom "${CMAKE_BINARY_DIR}/clean.md" --out "${_out}" --depfile "${_dep}"
    WORKING_DIRECTORY "${CMAKE_SOURCE_DIR}"
    DEPENDS "${WOTES_TILED}" "${CMAKE_SOURCE_DIR}/maps/${name}.tmj" "${CMAKE_BINARY_DIR}/clean.md"
    DEPFILE "${_dep}"
    COMMENT "Converting maps/${name}.tmj"
    VERBATIM)
  target_sources(patch.elf PRIVATE "${_out}")
  set_source_files_properties("${_out}" PROPERTIES LANGUAGE ASM GENERATED TRUE)
endfunction()
