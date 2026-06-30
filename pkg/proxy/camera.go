package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/emergingrobotics/gorai/pkg/mesh"
)

// RemoteCamera is a proxy that implements the Camera interface via NATS.
type RemoteCamera struct {
	meshClient *mesh.Client
	natsConn   *nats.Conn
	descriptor mesh.ServiceDescriptor
	logger     *slog.Logger

	imageSubject string
	cmdSubject   string

	mu         sync.RWMutex
	lastFrame  []byte
	lastMeta   *FrameMetadata
	lastFrameAt time.Time

	imageSub *nats.Subscription
	cancel   context.CancelFunc
	done     chan struct{}
}

// FrameMetadata holds camera frame metadata.
type FrameMetadata struct {
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Format   string `json:"format"`
	Encoding string `json:"encoding"`
	Sequence int64  `json:"sequence"`
}

// RemoteCameraOption configures a RemoteCamera.
type RemoteCameraOption func(*RemoteCamera)

// WithRemoteCameraLogger sets the logger.
func WithRemoteCameraLogger(logger *slog.Logger) RemoteCameraOption {
	return func(c *RemoteCamera) {
		c.logger = logger
	}
}

// NewRemoteCamera creates a new remote camera proxy.
func NewRemoteCamera(mc *mesh.Client, nc *nats.Conn, desc mesh.ServiceDescriptor, opts ...RemoteCameraOption) (*RemoteCamera, error) {
	c := &RemoteCamera{
		meshClient: mc,
		natsConn:   nc,
		descriptor: desc,
		logger:     slog.Default(),
		done:       make(chan struct{}),
	}

	for _, opt := range opts {
		opt(c)
	}

	// Find image subject
	c.imageSubject = findImageSubject(desc)

	// Find command subject
	var err error
	c.cmdSubject, err = GetCommandSubject(desc)
	if err != nil {
		c.cmdSubject = "" // May not have commands
	}

	// Subscribe to image data if available
	if c.imageSubject != "" {
		c.imageSub, err = nc.Subscribe(c.imageSubject, c.handleImage)
		if err != nil {
			c.logger.Warn("failed to subscribe to images", "subject", c.imageSubject, "error", err)
		}
	}

	c.logger.Debug("created remote camera",
		"name", desc.Name,
		"image_subject", c.imageSubject,
	)

	return c, nil
}

// findImageSubject finds the image subject from the descriptor.
func findImageSubject(desc mesh.ServiceDescriptor) string {
	for _, pub := range desc.Publishes {
		if containsAny(pub, "image", "frame", "video", "camera") {
			return pub
		}
	}
	// Construct default
	return fmt.Sprintf("gorai.%s.%s.data", desc.RobotID, desc.Name)
}

// handleImage processes incoming image data.
func (c *RemoteCamera) handleImage(msg *nats.Msg) {
	// Check if this is metadata or raw image
	if len(msg.Data) > 0 && msg.Data[0] == '{' {
		// JSON metadata with image reference
		var meta struct {
			Width       int    `json:"width"`
			Height      int    `json:"height"`
			Format      string `json:"format"`
			Encoding    string `json:"encoding"`
			Sequence    int64  `json:"sequence"`
			DataSubject string `json:"data_subject"`
		}
		if err := json.Unmarshal(msg.Data, &meta); err == nil {
			c.mu.Lock()
			c.lastMeta = &FrameMetadata{
				Width:    meta.Width,
				Height:   meta.Height,
				Format:   meta.Format,
				Encoding: meta.Encoding,
				Sequence: meta.Sequence,
			}
			c.lastFrameAt = time.Now()
			c.mu.Unlock()
			return
		}
	}

	// Raw image data
	c.mu.Lock()
	c.lastFrame = msg.Data
	c.lastFrameAt = time.Now()
	c.mu.Unlock()
}

// GetImage returns the latest image frame.
func (c *RemoteCamera) GetImage(ctx context.Context) ([]byte, *FrameMetadata, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.lastFrame == nil {
		return nil, nil, fmt.Errorf("no image available")
	}

	// Return a copy
	frame := make([]byte, len(c.lastFrame))
	copy(frame, c.lastFrame)

	return frame, c.lastMeta, nil
}

// Stream returns a channel that receives frames.
func (c *RemoteCamera) Stream(ctx context.Context) (<-chan []byte, error) {
	ch := make(chan []byte, 10)

	sub, err := c.natsConn.Subscribe(c.imageSubject, func(msg *nats.Msg) {
		select {
		case ch <- msg.Data:
		default:
			// Channel full, drop frame
		}
	})
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		sub.Unsubscribe()
		close(ch)
	}()

	return ch, nil
}

// Properties returns camera properties.
func (c *RemoteCamera) Properties(ctx context.Context) (map[string]any, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	props := map[string]any{
		"name": c.descriptor.Name,
	}

	if c.lastMeta != nil {
		props["width"] = c.lastMeta.Width
		props["height"] = c.lastMeta.Height
		props["format"] = c.lastMeta.Format
		props["encoding"] = c.lastMeta.Encoding
	}

	return props, nil
}

// Close releases resources.
func (c *RemoteCamera) Close(ctx context.Context) error {
	if c.imageSub != nil {
		c.imageSub.Unsubscribe()
	}
	if c.cancel != nil {
		c.cancel()
	}
	close(c.done)
	return nil
}

// Name returns the camera name.
func (c *RemoteCamera) Name() string {
	return c.descriptor.Name
}

// Descriptor returns the service descriptor.
func (c *RemoteCamera) Descriptor() mesh.ServiceDescriptor {
	return c.descriptor
}

// DoCommand sends a command to the camera.
func (c *RemoteCamera) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if c.cmdSubject == "" {
		return nil, fmt.Errorf("camera does not support commands")
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}

	msg, err := c.natsConn.RequestWithContext(ctx, c.cmdSubject, data)
	if err != nil {
		return nil, err
	}

	var resp map[string]any
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}
