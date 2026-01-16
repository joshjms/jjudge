package mq

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/pubsub"
	"github.com/jjudge-oj/apiserver/config"
	"google.golang.org/api/option"
)

// PubSubClient wraps the Google Cloud Pub/Sub SDK client.
type PubSubClient struct {
	client               *pubsub.Client
	subscriptionSuffix   string
}

// NewPubSubClient constructs a Pub/Sub client from config.
func NewPubSubClient(ctx context.Context, cfg config.PubSubConfig) (*PubSubClient, error) {
	if strings.TrimSpace(cfg.ProjectID) == "" {
		return nil, errors.New("pubsub project id is required")
	}

	var opts []option.ClientOption
	if strings.TrimSpace(cfg.CredentialsFile) != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.CredentialsFile))
	}

	client, err := pubsub.NewClient(ctx, cfg.ProjectID, opts...)
	if err != nil {
		return nil, err
	}

	suffix := cfg.SubscriptionSuffix
	if suffix == "" {
		suffix = "-sub"
	}

	return &PubSubClient{
		client:             client,
		subscriptionSuffix: suffix,
	}, nil
}

// Publish sends a message to the named topic.
func (p *PubSubClient) Publish(ctx context.Context, channel string, data []byte, attrs map[string]string) (string, error) {
	if strings.TrimSpace(channel) == "" {
		return "", errors.New("pubsub channel is required")
	}

	topic, err := p.ensureTopic(ctx, channel)
	if err != nil {
		return "", err
	}
	result := topic.Publish(ctx, &pubsub.Message{Data: data, Attributes: attrs})
	return result.Get(ctx)
}

// Subscribe consumes messages from the named channel.
func (p *PubSubClient) Subscribe(ctx context.Context, channel string, handler Handler) error {
	if strings.TrimSpace(channel) == "" {
		return errors.New("pubsub channel is required")
	}

	topic, err := p.ensureTopic(ctx, channel)
	if err != nil {
		return err
	}

	subscriptionName := p.subscriptionName(channel)
	sub, err := p.ensureSubscription(ctx, subscriptionName, topic)
	if err != nil {
		return err
	}

	return sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		message := Message{
			ID:         msg.ID,
			Data:       msg.Data,
			Attributes: msg.Attributes,
		}
		if err := handler(ctx, message); err != nil {
			msg.Nack()
			return
		}
		msg.Ack()
	})
}

// Close closes the underlying Pub/Sub client.
func (p *PubSubClient) Close() error {
	return p.client.Close()
}

func (p *PubSubClient) ensureTopic(ctx context.Context, name string) (*pubsub.Topic, error) {
	topic := p.client.Topic(name)
	exists, err := topic.Exists(ctx)
	if err != nil {
		return nil, err
	}
	if !exists {
		return p.client.CreateTopic(ctx, name)
	}
	return topic, nil
}

func (p *PubSubClient) ensureSubscription(ctx context.Context, name string, topic *pubsub.Topic) (*pubsub.Subscription, error) {
	sub := p.client.Subscription(name)
	exists, err := sub.Exists(ctx)
	if err != nil {
		return nil, err
	}
	if !exists {
		return p.client.CreateSubscription(ctx, name, pubsub.SubscriptionConfig{Topic: topic})
	}
	return sub, nil
}

func (p *PubSubClient) subscriptionName(channel string) string {
	if p.subscriptionSuffix == "" {
		return channel
	}
	return channel + p.subscriptionSuffix
}
