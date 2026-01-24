module github.com/gorai/gorai

go 1.22

require (
	github.com/blackjack/webcam v0.6.1
	github.com/coder/websocket v1.8.12
	github.com/go-chi/chi/v5 v5.1.0
	github.com/gorai/gorai-gps v0.1.0
	github.com/nats-io/nats.go v1.37.0
	github.com/spf13/cobra v1.10.2
	github.com/stretchr/testify v1.11.1
	github.com/warthog618/go-gpiocdev v0.9.1
	golang.org/x/sys v0.27.0
	google.golang.org/protobuf v1.35.2
	gopkg.in/yaml.v3 v3.0.1
)

// For local development - use the local gorai-gps module
replace github.com/gorai/gorai-gps => ../gorai-gps

require (
	github.com/creack/goselect v0.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/nats-io/nkeys v0.4.7 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	go.bug.st/serial v1.6.2 // indirect
	golang.org/x/crypto v0.29.0 // indirect
)
