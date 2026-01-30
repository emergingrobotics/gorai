# Orange Pi Future Support

**Status:** Deferred
**Last Updated:** 2026-01-27

This document contains Orange Pi support that has been deferred to focus on Raspberry Pi 5 as the primary platform. The content here is preserved for future implementation when Orange Pi support is prioritized.

## Rationale for Deferral

Orange Pi support was deferred due to:

1. **Overlay complexity**: Device tree configuration on Ubuntu/Armbian proved unreliable
2. **Boot issues**: Adding overlays via extlinux caused boot failures
3. **Limited testing hardware**: Development focus shifted to well-tested RPi5 platform
4. **Ship date pressure**: Simplifying to one platform reduces testing and support burden

## Future Boards to Support

When Orange Pi support is revisited, the following boards should be considered:

| Board | Header | NPU | Priority |
|-------|--------|-----|----------|
| Orange Pi 5 Plus | 40-pin | 6 TOPS | High (RPi-compatible header) |
| Orange Pi 5 Pro | 40-pin | 6 TOPS | Medium |
| Orange Pi 5B | 26-pin | 6 TOPS | Low (non-standard header) |
| Orange Pi 5 | 26-pin | 6 TOPS | Low (non-standard header) |

**Recommendation**: Focus on Orange Pi 5 Plus first due to 40-pin header compatibility.

---

## Board Detection

Add to `driver/hal/detect.go`:

```go
const (
    // BoardOrangePi5Plus is Orange Pi 5 Plus (40-pin header).
    BoardOrangePi5Plus Board = "opi5plus"

    // BoardOrangePi5B is Orange Pi 5B (26-pin header).
    BoardOrangePi5B Board = "opi5b"

    // BoardOrangePi5 is Orange Pi 5 (26-pin header).
    BoardOrangePi5 Board = "opi5"
)

// In Detect() function:
case strings.Contains(modelStr, "orange pi 5 plus"):
    return BoardOrangePi5Plus
case strings.Contains(modelStr, "orange pi 5b"):
    return BoardOrangePi5B
case strings.Contains(modelStr, "orange pi 5"):
    return BoardOrangePi5
```

---

## RK3588 GPIO Calculation

The RK3588 SoC uses a bank/group/line GPIO numbering scheme:

```go
// GPIO number = bank*32 + group*8 + line
// Groups: A=0, B=1, C=2, D=3
// Example: GPIO4_B3 = 4*32 + 1*8 + 3 = 139
func rk3588Pin(bank, group, line int) int {
    return bank*32 + group*8 + line
}
```

---

## Orange Pi 5 Plus Pin Mapping (40-pin Header)

```go
var opi5PlusPhysicalPins = map[int]int{
    3:  139, // GPIO4_B3 / I2C2_SDA_M0
    5:  140, // GPIO4_B4 / I2C2_SCL_M0
    7:  36,  // GPIO1_A4
    8:  13,  // GPIO0_B5 / UART2_TX_M0
    10: 14,  // GPIO0_B6 / UART2_RX_M0
    11: 35,  // GPIO1_A3
    12: 42,  // GPIO1_B2 / PWM14_M0
    13: 150, // GPIO4_C6 / PWM13_M2
    15: 63,  // GPIO1_D7
    16: 62,  // GPIO1_D6
    18: 43,  // GPIO1_B3
    19: 33,  // GPIO1_A1 / SPI4_MOSI
    21: 32,  // GPIO1_A0 / SPI4_MISO
    22: 61,  // GPIO1_D5
    23: 34,  // GPIO1_A2 / SPI4_CLK
    24: 35,  // GPIO1_A3 / SPI4_CS0
    26: 36,  // GPIO1_A4 / SPI4_CS1
    27: 141, // GPIO4_B5 / I2C5_SDA
    28: 142, // GPIO4_B6 / I2C5_SCL
    29: 40,  // GPIO1_B0
    31: 39,  // GPIO1_A7
    32: 45,  // GPIO1_B5 / PWM1_M1
    33: 44,  // GPIO1_B4 / PWM0_M1
    35: 145, // GPIO4_C1
    36: 144, // GPIO4_C0
    37: 135, // GPIO4_A7
    38: 146, // GPIO4_C2
    40: 136, // GPIO4_B0
}
```

---

## Orange Pi 5/5B Pin Mapping (26-pin Header)

```go
var opi5PhysicalPins = map[int]int{
    3:  139, // GPIO4_B3 / I2C2_SDA_M0
    5:  140, // GPIO4_B4 / I2C2_SCL_M0
    7:  36,  // GPIO1_A4
    8:  13,  // GPIO0_B5 / UART2_TX_M0
    10: 14,  // GPIO0_B6 / UART2_RX_M0
    11: 35,  // GPIO1_A3
    12: 42,  // GPIO1_B2 / PWM14_M0
    13: 150, // GPIO4_C6
    15: 63,  // GPIO1_D7
    16: 138, // GPIO4_B2
    18: 43,  // GPIO1_B3
    19: 138, // GPIO4_B2 / SPI0_MOSI
    21: 41,  // GPIO1_B1 / SPI0_MISO
    22: 43,  // GPIO1_B3
    23: 44,  // GPIO1_B4 / SPI0_CLK
    24: 42,  // GPIO1_B2 / SPI0_CS0
    26: 149, // GPIO4_C5
}
```

---

## PWM Mapping

### Orange Pi 5 Plus (4 PWM channels)

```go
var opi5PlusPWMPins = map[int]PWMPinMapping{
    44:  {Chip: 0, Channel: 0}, // PWM0 (GPIO1_B4), Physical Pin 33
    45:  {Chip: 0, Channel: 1}, // PWM1 (GPIO1_B5), Physical Pin 32
    42:  {Chip: 3, Channel: 2}, // PWM14 (GPIO1_B2), Physical Pin 12
    150: {Chip: 3, Channel: 1}, // PWM13 (GPIO4_C6), Physical Pin 13
}
```

### Orange Pi 5/5B (2 PWM channels)

```go
var opi5PWMPins = map[int]PWMPinMapping{
    42:  {Chip: 3, Channel: 2}, // PWM14 (GPIO1_B2), Physical Pin 12
    150: {Chip: 3, Channel: 1}, // PWM13 (GPIO4_C6), Physical Pin 13
}
```

---

## Device Tree Configuration

### Armbian/Orange Pi OS

Edit `/boot/armbianEnv.txt` or `/boot/orangepiEnv.txt`:

```ini
# Orange Pi 5 Plus
overlays=rk3588-pwm0-m1 rk3588-pwm1-m1 rk3588-i2c5-m0 rk3588-spi4-m0-cs0-spidev rk3588-uart2-m0

# Orange Pi 5/5B
overlays=rk3588-pwm13-m0 rk3588-pwm14-m0 rk3588-i2c5-m3 rk3588-spi0-m2-cs0-spidev rk3588-uart2-m0
```

### Ubuntu with extlinux

**WARNING**: This method proved unreliable and caused boot failures during testing.

Edit `/boot/extlinux/extlinux.conf` and add `fdtoverlays` line:

```
label l0
    ...
    fdtoverlays /lib/firmware/.../rockchip/overlay/rk3588-pwm0-m1.dtbo
    ...
```

---

## Recommended Pin Allocation for Robotics

### Orange Pi 5 Plus (40-pin Header)

| Function | Physical Pins | GPIO (RK3588) | Device |
|----------|---------------|---------------|--------|
| PWM0 | 33 | GPIO1_B4 (44) | pwmchip0 ch0 |
| PWM1 | 32 | GPIO1_B5 (45) | pwmchip0 ch1 |
| I2C2 | 3, 5 | GPIO4_B3 (139), GPIO4_B4 (140) | /dev/i2c-2 |
| I2C5 | 27, 28 | GPIO4_B5 (141), GPIO4_B6 (142) | /dev/i2c-5 |
| SPI4 | 19, 21, 23, 24, 26 | GPIO1_A1, A0, A2, A3, A4 | /dev/spidev4.0 |
| UART2 | 8, 10 | GPIO0_B5 (13), GPIO0_B6 (14) | /dev/ttyS2 |

**Power pins**: +5V (2, 4), +3.3V (1, 17)
**Ground pins**: 6, 9, 14, 20, 25, 30, 34, 39

### Orange Pi 5/5B (26-pin Header)

| Function | Physical Pins | GPIO (RK3588) | Device |
|----------|---------------|---------------|--------|
| PWM14 | 12 | GPIO1_B2 (42) | pwmchip3 ch2 |
| PWM13 | 13 | GPIO4_C6 (150) | pwmchip3 ch1 |
| I2C2 | 3, 5 | GPIO4_B3 (139), GPIO4_B4 (140) | /dev/i2c-2 |
| SPI0 | 19, 21, 23, 24 | Various | /dev/spidev0.0 |
| UART2 | 8, 10 | GPIO0_B5 (13), GPIO0_B6 (14) | /dev/ttyS2 |

**Power pins**: +5V (2, 4), +3.3V (1, 17)
**Ground pins**: 6, 9, 14, 20, 25

---

## Known Issues

1. **PWM channels claimed by system**: On Ubuntu, pwmchip0/1 are used by backlight and fan drivers
2. **Overlay boot failures**: Adding overlays via extlinux can prevent boot
3. **Pull-up/pull-down resistors**: Do not function correctly via libgpiod
4. **Pin muxing conflicts**: Many pins have multiple functions that conflict

---

## OS Recommendations

For future Orange Pi support, use:

1. **Official Orange Pi OS** - Best for NPU + GPIO together
2. **Armbian** - Good overlay support, active community
3. **Avoid stock Ubuntu** - Overlay configuration via extlinux is fragile

---

## Resources

- [Orange Pi 5 Plus Wiki](http://www.orangepi.org/orangepiwiki/index.php/Orange_Pi_5_Plus)
- [Armbian for Orange Pi 5 Plus](https://www.armbian.com/orangepi-5-plus/)
- [RK3588 Datasheet](https://opensource.rock-chips.com/wiki_RK3588)
- [RKNN-Toolkit2](https://github.com/rockchip-linux/rknn-toolkit2)

---

## Implementation Checklist

When implementing Orange Pi support:

- [ ] Add board constants to `detect.go`
- [ ] Create `pins_opi.go` with pin mappings
- [ ] Add PWM mappings to `pwm_mapping.go`
- [ ] Update `GetPinMapper()` in `pins.go`
- [ ] Add tests for all new boards
- [ ] Test on actual hardware with oscilloscope
- [ ] Document working OS/overlay combinations
- [ ] Update hardware-abstraction.md
