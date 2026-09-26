/* Places the patch in the free region mdrom measured. free.ld (generated into the
   build directory) and wotes.ld (game/) are found through the -L paths CMakeLists.txt
   passes before -T. Hook sections are placed by --section-start flags, not here. */
INCLUDE free.ld
INCLUDE wotes.ld

/* Work RAM the game clears at boot and never touches afterwards. */
MEMORY
{
  RAM_FREE (rw) : ORIGIN = 0xFFAC00, LENGTH = 0x1800
}

SECTIONS
{
  .text : { *(.text*) *(.rodata*) } > ROM_FREE

  .data : { *(.data*) } > ROM_FREE
  .bss (NOLOAD) : { *(.bss*) *(COMMON) } > RAM_FREE
  ASSERT(SIZEOF(.data) == 0, "no initialised globals: nothing copies .data to RAM")
}
