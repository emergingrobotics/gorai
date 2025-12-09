## 1.5 Prerequisites

GoRAI is designed to be approachable, but some background knowledge will help you get the most from this book.

### Required: Basic Go Knowledge

You should be comfortable with Go fundamentals:

- **Variables and types**: `var`, `:=`, basic types (`int`, `string`, `float64`)
- **Functions**: Declaration, multiple return values, error handling
- **Structs**: Field definition, methods with receivers
- **Interfaces**: How they work, implicit satisfaction
- **Slices and maps**: Creation, access, iteration
- **Goroutines and channels**: Basic concurrent patterns
- **Packages and imports**: Go module structure

If you're new to Go, spend a few hours with the [Go Tour](https://go.dev/tour/) before diving in. The concepts translate quickly, especially if you know Python, JavaScript, or C.

### Required: Command-Line Familiarity

You'll spend time in the terminal:

- Navigating directories (`cd`, `ls`, `pwd`)
- Running commands with flags
- Understanding stdout, stderr, and exit codes
- Basic environment variables

Nothing exotic—if you've used a Unix-like terminal, you're prepared.

### Helpful: Networking Basics

Understanding helps but isn't required:

- IP addresses and ports
- TCP vs UDP (NATS uses TCP)
- Client-server vs peer-to-peer models
- What "localhost" means

The book explains what you need when you need it.

### Helpful: Basic Electronics/Hardware

If you want to connect real hardware:

- What GPIO, I2C, SPI, and UART mean
- How to read a pinout diagram
- Basic electrical safety (don't short 5V to ground)

For the first several chapters, you'll work with simulated and fake components. Hardware comes later, and we'll explain what you need.

### Optional: Prior Robotics Experience

Experience with ROS, ROS 2, or other robotics frameworks helps you appreciate GoRAI's design choices. But it's not required—we explain concepts from first principles.

If you're coming from ROS, you'll recognize familiar patterns: nodes, topics, publishers, subscribers, services. GoRAI's versions are simpler but serve the same purposes.

### Development Environment

You'll need:

- A computer running Linux, macOS, or Windows (with WSL2 for Linux compatibility)
- Go 1.21 or later installed
- A text editor or IDE (VS Code with Go extension recommended)
- Git for cloning repositories
- Network access for downloading dependencies

Chapter 8 covers setup in detail. For now, confirm you can run `go version` and see output like `go version go1.22.0 linux/amd64`.

---

With these foundations in place, you're ready to understand how GoRAI thinks about robotics. Chapter 2 introduces the mental model and architecture that makes everything else make sense.
