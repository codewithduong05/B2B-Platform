package messaging

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/atlas-platform/backend/internal/config"
	"github.com/rabbitmq/amqp091-go"
)

const (
	defaultReconnectDelay = 5 * time.Second
	defaultHeartbeat      = 10 * time.Second
	defaultLocale         = "en_US"
)

type RabbitMQ struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
	config  *config.RabbitMQConfig
	closed  chan struct{}
}

func New(ctx context.Context, cfg *config.RabbitMQConfig) (*RabbitMQ, error) {
	url := fmt.Sprintf(
		"amqp://%s:%s@%s:%d%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.VHost,
	)

	conn, err := amqp091.DialConfig(url, amqp091.Config{
		Heartbeat: defaultHeartbeat,
		Locale:    defaultLocale,
	})
	if err != nil {
		return nil, fmt.Errorf("dial rabbitmq: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open channel: %w", err)
	}

	rmq := &RabbitMQ{
		conn:    conn,
		channel: channel,
		config:  cfg,
		closed:  make(chan struct{}),
	}

	go rmq.handleConnectionClose(ctx)

	slog.InfoContext(ctx, "rabbitmq connection established",
		slog.String("host", cfg.Host),
		slog.Int("port", cfg.Port),
		slog.String("vhost", cfg.VHost),
	)

	return rmq, nil
}

func (r *RabbitMQ) handleConnectionClose(ctx context.Context) {
	notifyClose := r.conn.NotifyClose(make(chan *amqp091.Error, 1))
	select {
	case err := <-notifyClose:
		if err != nil {
			slog.ErrorContext(ctx, "rabbitmq connection closed", slog.String("error", err.Error()))
		} else {
			slog.InfoContext(ctx, "rabbitmq connection closed gracefully")
		}
	case <-r.closed:
		return
	}
}

func (r *RabbitMQ) Channel() *amqp091.Channel {
	return r.channel
}

func (r *RabbitMQ) Connection() *amqp091.Connection {
	return r.conn
}

func (r *RabbitMQ) DeclareExchange(name, kind string, durable, autoDelete bool) error {
	return r.channel.ExchangeDeclare(
		name,
		kind,
		durable,
		autoDelete,
		false,
		false,
		nil,
	)
}

func (r *RabbitMQ) DeclareQueue(name string, durable, autoDelete, exclusive bool) (amqp091.Queue, error) {
	return r.channel.QueueDeclare(
		name,
		durable,
		autoDelete,
		exclusive,
		false,
		nil,
	)
}

func (r *RabbitMQ) BindQueue(queue, routingKey, exchange string) error {
	return r.channel.QueueBind(
		queue,
		routingKey,
		exchange,
		false,
		nil,
	)
}

func (r *RabbitMQ) Publish(ctx context.Context, exchange, routingKey string, mandatory, immediate bool, msg amqp091.Publishing) error {
	return r.channel.PublishWithContext(ctx, exchange, routingKey, mandatory, immediate, msg)
}

func (r *RabbitMQ) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp091.Table) (<-chan amqp091.Delivery, error) {
	return r.channel.Consume(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
}

func (r *RabbitMQ) Close() error {
	close(r.closed)
	var errs []error

	if r.channel != nil {
		if err := r.channel.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close channel: %w", err))
		}
	}

	if r.conn != nil {
		if err := r.conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close connection: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing rabbitmq: %v", errs)
	}

	slog.InfoContext(context.Background(), "rabbitmq connection closed")
	return nil
}

func (r *RabbitMQ) IsConnected() bool {
	return r.conn != nil && !r.conn.IsClosed()
}
