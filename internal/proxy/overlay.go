package proxy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// OverlayNotifier sends events to the agent overlay server.
type OverlayNotifier struct {
	endpoint string
	client   *http.Client
	enabled  bool
	mu       sync.RWMutex
}

// OverlayEvent represents an event to send to the overlay.
type OverlayEvent struct {
	Type      string      `json:"type"`
	ProxyID   string      `json:"proxy_id"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// NewOverlayNotifier creates a new overlay notifier.
func NewOverlayNotifier() *OverlayNotifier {
	return &OverlayNotifier{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		enabled: false,
	}
}

// SetEndpoint sets the overlay endpoint URL.
// Example: "http://127.0.0.1:19191"
func (n *OverlayNotifier) SetEndpoint(endpoint string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.endpoint = endpoint
	n.enabled = endpoint != ""
}

// GetEndpoint returns the current endpoint.
func (n *OverlayNotifier) GetEndpoint() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.endpoint
}

// IsEnabled returns whether the notifier is enabled.
func (n *OverlayNotifier) IsEnabled() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.enabled
}

// NotifyPanelMessage sends a panel message to the overlay.
func (n *OverlayNotifier) NotifyPanelMessage(proxyID string, msg *PanelMessage) error {
	return n.send(OverlayEvent{
		Type:      "panel_message",
		ProxyID:   proxyID,
		Timestamp: time.Now(),
		Data:      msg,
	})
}

// NotifySketch sends a sketch to the overlay.
func (n *OverlayNotifier) NotifySketch(proxyID string, sketch *SketchEntry) error {
	return n.send(OverlayEvent{
		Type:      "sketch",
		ProxyID:   proxyID,
		Timestamp: time.Now(),
		Data:      sketch,
	})
}

// NotifyInteraction sends an interaction event to the overlay.
func (n *OverlayNotifier) NotifyInteraction(proxyID string, interaction *InteractionEvent) error {
	return n.send(OverlayEvent{
		Type:      "interaction",
		ProxyID:   proxyID,
		Timestamp: time.Now(),
		Data:      interaction,
	})
}

// NotifyCustom sends a custom event to the overlay.
func (n *OverlayNotifier) NotifyCustom(proxyID, eventType string, data interface{}) error {
	return n.send(OverlayEvent{
		Type:      eventType,
		ProxyID:   proxyID,
		Timestamp: time.Now(),
		Data:      data,
	})
}

func (n *OverlayNotifier) send(event OverlayEvent) error {
	n.mu.RLock()
	if !n.enabled {
		n.mu.RUnlock()
		return nil
	}
	endpoint := n.endpoint
	n.mu.RUnlock()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	resp, err := n.client.Post(endpoint+"/event", "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// TypeToOverlay sends a type command to the overlay.
// The overlay will inject this text into the PTY.
func (n *OverlayNotifier) TypeToOverlay(text string, enter bool) error {
	n.mu.RLock()
	if !n.enabled {
		n.mu.RUnlock()
		return nil
	}
	endpoint := n.endpoint
	n.mu.RUnlock()

	data, err := json.Marshal(map[string]interface{}{
		"text":    text,
		"enter":   enter,
		"instant": true,
	})
	if err != nil {
		return err
	}

	resp, err := n.client.Post(endpoint+"/type", "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// SendKeyToOverlay sends a key to the overlay.
func (n *OverlayNotifier) SendKeyToOverlay(key string, ctrl, alt, shift bool) error {
	n.mu.RLock()
	if !n.enabled {
		n.mu.RUnlock()
		return nil
	}
	endpoint := n.endpoint
	n.mu.RUnlock()

	data, err := json.Marshal(map[string]interface{}{
		"key":   key,
		"ctrl":  ctrl,
		"alt":   alt,
		"shift": shift,
	})
	if err != nil {
		return err
	}

	resp, err := n.client.Post(endpoint+"/key", "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
