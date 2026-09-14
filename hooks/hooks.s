| Each .hook_XXXXXX section is placed at ROM offset XXXXXX by the HOOKS list in
| CMakeLists.txt; its .orig_ twin declares the clean-ROM bytes mdrom checks before writing.

| Startup hook at 0x300: the stack pointer is not valid yet, so jmp rather than jsr.
| poc_entry runs the lea the game executes at 0x306, calls C, then resumes at 0x306.
| No registers are saved: the code from 0x306 on reinitialises every register.
    .section .hook_000300,"ax"
    jmp     poc_entry
    .section .orig_000300,""
    .byte   0x4e,0x71,0x4e,0x71,0x4e,0x71

    .text
    .globl  poc_entry
poc_entry:
    lea     0xfffffff6,%sp
    jsr     poc_init
poc_ret:
    jmp     0x306

| Class list on the character-creation screen: FIGHTER becomes WARRIOR in both
| copies, the plain list at 0x22DA and the menu list with control codes at 0x230D.
    .section .hook_0022E1,"a"
    .ascii  "WARRIOR"
    .section .orig_0022E1,""
    .ascii  "FIGHTER"

    .section .hook_002316,"a"
    .ascii  "WARRIOR"
    .section .orig_002316,""
    .ascii  "FIGHTER"

    .section .note.GNU-stack,"",%progbits
