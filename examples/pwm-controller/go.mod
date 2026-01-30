module github.com/gorai/gorai/examples/pwm-controller

go 1.22

require (
	github.com/thefloweringash/gorai-gsp v0.0.0
	go.bug.st/serial v1.6.4
)

require (
	github.com/creack/goselect v0.1.2 // indirect
	golang.org/x/sys v0.19.0 // indirect
)

// Use local gorai-gsp module
replace github.com/thefloweringash/gorai-gsp => ../../../gorai-gsp
