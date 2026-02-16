module github.com/gorai/gorai

go 1.24.0

require (
	github.com/blackjack/webcam v0.6.1
	github.com/coder/websocket v1.8.12
	github.com/go-chi/chi/v5 v5.1.0
	github.com/google/uuid v1.6.0
	github.com/nats-io/nats.go v1.37.0
	github.com/spf13/cobra v1.10.2
	github.com/stretchr/testify v1.11.1
	github.com/warthog618/go-gpiocdev v0.9.1
	go.bug.st/serial v1.6.4
	golang.org/x/sys v0.38.0
	google.golang.org/protobuf v1.35.2
	gopkg.in/yaml.v3 v3.0.1
)

// gorai-gps is required only when building with the gorai_gps build tag.
// For local development: go build -tags gorai_gps
// and add to go.mod: require github.com/gorai/gorai-gps v0.1.0
// replace github.com/gorai/gorai-gps => ../gorai-gps

require (
	github.com/creack/goselect v0.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/nats-io/nkeys v0.4.7 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	golang.org/x/crypto v0.45.0 // indirect
	golang.org/x/text v0.31.0 // indirect
)
