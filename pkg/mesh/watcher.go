package mesh

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Watcher watches for mesh changes (services joining/leaving, channels added/removed).
type Watcher struct {
	client     *Client
	query      Query
	events     chan Event
	cancel     context.CancelFunc
	done       chan struct{}
	mu         sync.RWMutex
	announceSub *nats.Subscription
}

// WatcherOption configures a watcher.
type WatcherOption func(*watcherConfig)

type watcherConfig struct {
	bufferSize    int
	watchKV       bool
	watchAnnounce bool
}

// WithBufferSize sets the event channel buffer size.
func WithBufferSize(size int) WatcherOption {
	return func(c *watcherConfig) {
		c.bufferSize = size
	}
}

// WithKVWatch enables watching KV changes (more comprehensive but more overhead).
func WithKVWatch(enabled bool) WatcherOption {
	return func(c *watcherConfig) {
		c.watchKV = enabled
	}
}

// WithAnnounceWatch enables watching announcements (lighter weight).
func WithAnnounceWatch(enabled bool) WatcherOption {
	return func(c *watcherConfig) {
		c.watchAnnounce = enabled
	}
}

// WatchServices starts watching for service changes.
func (c *Client) WatchServices(ctx context.Context, q Query, opts ...WatcherOption) (*Watcher, error) {
	cfg := &watcherConfig{
		bufferSize:    100,
		watchKV:       true,
		watchAnnounce: true,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	watchCtx, cancel := context.WithCancel(ctx)
	w := &Watcher{
		client: c,
		query:  q,
		events: make(chan Event, cfg.bufferSize),
		cancel: cancel,
		done:   make(chan struct{}),
	}

	// Subscribe to announcements
	if cfg.watchAnnounce {
		sub, err := c.nc.Subscribe(SubjectAnnounce, func(msg *nats.Msg) {
			var ann Announcement
			if err := json.Unmarshal(msg.Data, &ann); err != nil {
				return
			}

			if !q.Matches(ann.Service) {
				return
			}

			var eventType EventType
			switch ann.Type {
			case "join":
				eventType = EventServiceJoined
			case "leave":
				eventType = EventServiceLeft
			default:
				return
			}

			select {
			case w.events <- Event{
				Type:      eventType,
				Service:   &ann.Service,
				Timestamp: ann.Timestamp,
			}:
			default:
				// Buffer full, drop event
			}
		})
		if err != nil {
			cancel()
			return nil, err
		}
		w.announceSub = sub
	}

	// Watch KV for changes
	if cfg.watchKV {
		go w.watchKV(watchCtx, q)
	}

	// Cleanup goroutine
	go func() {
		<-watchCtx.Done()
		if w.announceSub != nil {
			w.announceSub.Unsubscribe()
		}
		close(w.events)
		close(w.done)
	}()

	return w, nil
}

// watchKV watches the services KV bucket for changes.
func (w *Watcher) watchKV(ctx context.Context, q Query) {
	watcher, err := w.client.kv.Services().WatchAll(ctx)
	if err != nil {
		w.client.logger.Warn("failed to start KV watcher", "error", err)
		return
	}
	defer watcher.Stop()

	// Track known services to detect deletions
	known := make(map[string]ServiceDescriptor)

	for {
		select {
		case <-ctx.Done():
			return
		case entry := <-watcher.Updates():
			if entry == nil {
				continue
			}

			key := entry.Key()

			// Handle deletion
			if entry.Operation() == jetstream.KeyValueDelete {
				if svc, ok := known[key]; ok {
					delete(known, key)
					if q.Matches(svc) {
						select {
						case w.events <- Event{
							Type:      EventServiceLeft,
							Service:   &svc,
							Timestamp: time.Now(),
						}:
						default:
						}
					}
				}
				continue
			}

			// Handle put/update
			var svc ServiceDescriptor
			if err := json.Unmarshal(entry.Value(), &svc); err != nil {
				continue
			}

			if !q.Matches(svc) {
				continue
			}

			_, existed := known[key]
			known[key] = svc

			var eventType EventType
			if existed {
				eventType = EventServiceUpdated
			} else {
				eventType = EventServiceJoined
			}

			select {
			case w.events <- Event{
				Type:      eventType,
				Service:   &svc,
				Timestamp: time.Now(),
			}:
			default:
			}
		}
	}
}

// Events returns the event channel.
func (w *Watcher) Events() <-chan Event {
	return w.events
}

// Stop stops the watcher.
func (w *Watcher) Stop() {
	w.cancel()
	<-w.done
}

// ChannelWatcher watches for channel changes.
type ChannelWatcher struct {
	client *Client
	query  ChannelQuery
	events chan Event
	cancel context.CancelFunc
	done   chan struct{}
}

// WatchChannels starts watching for channel changes.
func (c *Client) WatchChannels(ctx context.Context, q ChannelQuery, opts ...WatcherOption) (*ChannelWatcher, error) {
	cfg := &watcherConfig{
		bufferSize: 100,
		watchKV:    true,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	watchCtx, cancel := context.WithCancel(ctx)
	w := &ChannelWatcher{
		client: c,
		query:  q,
		events: make(chan Event, cfg.bufferSize),
		cancel: cancel,
		done:   make(chan struct{}),
	}

	if cfg.watchKV {
		go w.watchKV(watchCtx, q)
	}

	go func() {
		<-watchCtx.Done()
		close(w.events)
		close(w.done)
	}()

	return w, nil
}

// watchKV watches the channels KV bucket for changes.
func (w *ChannelWatcher) watchKV(ctx context.Context, q ChannelQuery) {
	watcher, err := w.client.kv.Channels().WatchAll(ctx)
	if err != nil {
		w.client.logger.Warn("failed to start channel KV watcher", "error", err)
		return
	}
	defer watcher.Stop()

	known := make(map[string]ChannelDescriptor)

	for {
		select {
		case <-ctx.Done():
			return
		case entry := <-watcher.Updates():
			if entry == nil {
				continue
			}

			key := entry.Key()

			if entry.Operation() == jetstream.KeyValueDelete {
				if ch, ok := known[key]; ok {
					delete(known, key)
					if w.client.matchesChannelQuery(ch, q) {
						select {
						case w.events <- Event{
							Type:      EventChannelRemoved,
							Channel:   &ch,
							Timestamp: time.Now(),
						}:
						default:
						}
					}
				}
				continue
			}

			var ch ChannelDescriptor
			if err := json.Unmarshal(entry.Value(), &ch); err != nil {
				continue
			}

			if !w.client.matchesChannelQuery(ch, q) {
				continue
			}

			_, existed := known[key]
			known[key] = ch

			var eventType EventType
			if existed {
				eventType = EventChannelUpdated
			} else {
				eventType = EventChannelAdded
			}

			select {
			case w.events <- Event{
				Type:      eventType,
				Channel:   &ch,
				Timestamp: time.Now(),
			}:
			default:
			}
		}
	}
}

// Events returns the event channel.
func (w *ChannelWatcher) Events() <-chan Event {
	return w.events
}

// Stop stops the watcher.
func (w *ChannelWatcher) Stop() {
	w.cancel()
	<-w.done
}

// WatchAnnouncements subscribes to all mesh announcements.
func (c *Client) WatchAnnouncements(ctx context.Context) (<-chan Announcement, error) {
	ch := make(chan Announcement, 100)

	sub, err := c.nc.Subscribe(SubjectAnnounce, func(msg *nats.Msg) {
		var ann Announcement
		if err := json.Unmarshal(msg.Data, &ann); err != nil {
			return
		}

		select {
		case ch <- ann:
		default:
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

// WatchHeartbeats subscribes to heartbeats for a specific service.
func (c *Client) WatchHeartbeats(ctx context.Context, serviceID string) (<-chan HeartbeatMessage, error) {
	ch := make(chan HeartbeatMessage, 10)
	subject := HeartbeatSubject(serviceID)

	sub, err := c.nc.Subscribe(subject, func(msg *nats.Msg) {
		var hb HeartbeatMessage
		if err := json.Unmarshal(msg.Data, &hb); err != nil {
			return
		}

		select {
		case ch <- hb:
		default:
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

// WatchAllHeartbeats subscribes to all service heartbeats.
func (c *Client) WatchAllHeartbeats(ctx context.Context) (<-chan HeartbeatMessage, error) {
	ch := make(chan HeartbeatMessage, 100)
	subject := SubjectHeartbeatPrefix + ".>"

	sub, err := c.nc.Subscribe(subject, func(msg *nats.Msg) {
		var hb HeartbeatMessage
		if err := json.Unmarshal(msg.Data, &hb); err != nil {
			return
		}

		select {
		case ch <- hb:
		default:
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
