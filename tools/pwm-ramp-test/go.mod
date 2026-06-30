module github.com/emergingrobotics/gorai/tools/pwm-ramp-test

go 1.22

require (
	github.com/emergingrobotics/rp2040-pwm v0.0.0
	go.bug.st/serial v1.6.4
)

require (
	github.com/creack/goselect v0.1.2 // indirect
	golang.org/x/sys v0.19.0 // indirect
)

replace github.com/emergingrobotics/rp2040-pwm => ../../../rp2040-pwm
