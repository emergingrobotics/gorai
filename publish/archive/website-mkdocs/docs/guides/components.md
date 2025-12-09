# Components Guide

Components are hardware abstractions in GoRAI.

## Component Types

- **Sensors**: Read data from the physical world
- **Actuators**: Affect the physical world (motors, servos)
- **Vision**: Cameras and image processing

## The Resource Interface

All components implement the Resource interface:

```go
type Resource interface {
    Name() string
    Reconfigure(ctx context.Context, config Config) error
    Close(ctx context.Context) error
}
```

*More content coming soon*
