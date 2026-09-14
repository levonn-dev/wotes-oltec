#include <stdint.h>
#include "wotes.h"

/* Exercises the 32-bit multiply, divide, and modulus helpers on the target; volatile
 * stops build-time folding. Expected return value: 356872 (0x57208), read from d0. */
uint32_t poc_init(void)
{
    volatile uint32_t a = 123456789u, b = 1000u, e = 3u;
    volatile int32_t c = -100000, d = 7;
    return (a / b) * e + (uint32_t)(c / d) + (a % b);
}
