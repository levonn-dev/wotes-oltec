# Locates mdrom: the copy `task bootstrap` puts in tools/mdrom, else a Go install.
# Override with -DMDROM=/path/to/mdrom.
execute_process(COMMAND go env GOPATH OUTPUT_VARIABLE _gopath OUTPUT_STRIP_TRAILING_WHITESPACE ERROR_QUIET)
set(_mdrom_hints "${CMAKE_SOURCE_DIR}/tools/mdrom")
if(_gopath)
  list(APPEND _mdrom_hints "${_gopath}/bin")
endif()
list(APPEND _mdrom_hints "$ENV{HOME}/go/bin")
find_program(MDROM mdrom HINTS ${_mdrom_hints} DOC "mdrom ROM patch tool")
if(NOT MDROM)
  message(FATAL_ERROR "mdrom not found. Run 'task bootstrap', or pass -DMDROM=/path/to/mdrom.")
endif()
message(STATUS "mdrom: ${MDROM}")
