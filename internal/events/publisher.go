package events

import "context"

type Publisher interface {
	Publish(ctx context.Context, eventType string, payload any) error
}
