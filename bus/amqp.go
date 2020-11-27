package bus

import (
	"context"
	"fmt"
	"time"

	"github.com/streadway/amqp"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/DoomSentinel/scheduler/config"
)

var Module = fx.Provide(
	NewAMQPMessageBus,
)

type (
	AMQP struct {
		writeSession *session
		readSession  *session

		log        *zap.Logger
		shutdowner fx.Shutdowner
	}
	session struct {
		*amqp.Connection
		*amqp.Channel

		closeChan chan *amqp.Error
	}
)

const (
	DelayedExchange     = "delayed"
	DelayedExchangeKind = "x-delayed-message"
	DelayedQueue        = "delayed-queue"

	NotificationsExchange = "notifications_fanout"
)

func NewAMQPMessageBus(lc fx.Lifecycle, shutdowner fx.Shutdowner, conf config.AMQPConfig, log *zap.Logger) (*AMQP, error) {
	dsn := fmt.Sprintf("amqp://%s:%s@%s:%d/", conf.User, conf.Password, conf.Host, conf.Port)
	writeSession, err := newSession(dsn)
	if err != nil {
		return nil, err
	}
	readSession, err := newSession(dsn)
	if err != nil {
		return nil, err
	}

	bus := &AMQP{
		readSession:  readSession,
		writeSession: writeSession,
		log:          log,
		shutdowner:   shutdowner,
	}

	err = bus.setup()
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go bus.handleErrors()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return bus.Close()
		},
	})

	return bus, nil
}

func (a *AMQP) PublishDelayed(body []byte, delay time.Duration, headers map[string]interface{}) error {
	if headers == nil {
		headers = make(map[string]interface{})
	}
	headers["x-delay"] = delay.Milliseconds()

	return a.writeSession.Publish(
		DelayedExchange, // exchange
		DelayedQueue,    // routing key
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Headers:      headers,
			Body:         body,
			Timestamp:    time.Now().UTC(),
			DeliveryMode: amqp.Persistent,
		})
}

func (a *AMQP) PublishNotification(body []byte) error {
	return a.writeSession.Publish(
		NotificationsExchange, // exchange
		"",                    // routing key
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
}

func (a *AMQP) ConsumeNotifications() (<-chan amqp.Delivery, error) {
	queue, err := a.readSession.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		return nil, err
	}
	err = a.readSession.QueueBind(queue.Name, "", NotificationsExchange, false, nil)
	if err != nil {
		return nil, err
	}

	messages, err := a.readSession.Consume(queue.Name, "", true, true, false, false, nil)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

func (a *AMQP) ConsumeTasks() (<-chan amqp.Delivery, error) {
	messages, err := a.readSession.Consume(DelayedQueue, "", false,
		false, false, false, nil)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

func (a *AMQP) setup() error {
	err := a.writeSession.ExchangeDeclare(
		DelayedExchange, DelayedExchangeKind, true,
		false, false, false, amqp.Table{
			"x-delayed-type": "direct",
		})
	if err != nil {
		return err
	}

	q, err := a.writeSession.QueueDeclare(
		DelayedQueue, // name
		true,         // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return err
	}
	err = a.writeSession.QueueBind(DelayedQueue, q.Name, DelayedExchange, false, nil)
	if err != nil {
		return err
	}

	return a.writeSession.ExchangeDeclare(
		NotificationsExchange, "fanout", true, false, false, false, nil,
	)
}

func (a *AMQP) handleErrors() {
	for {
		select {
		case err, ok := <-a.readSession.closeChan:
			if ok {
				a.log.Error("amqp read connection error: %v", zap.Error(err))
			}
			sError := a.shutdowner.Shutdown()
			if err != nil {
				a.log.Error("unable to shut down: %v", zap.Error(sError))
			}
			return
		case err, ok := <-a.writeSession.closeChan:
			if ok {
				a.log.Error("amqp write connection error: %v", zap.Error(err))
			}
			sError := a.shutdowner.Shutdown()
			if err != nil {
				a.log.Error("unable to shut down: %v", zap.Error(sError))
			}
			return
		}
	}
}

func (a *AMQP) Close() error {
	err := a.writeSession.Close()
	if err != nil {
		return err
	}
	err = a.readSession.Close()
	if err != nil {
		return err
	}

	return nil
}

func newSession(dsn string) (*session, error) {
	conn, channel, err := makeConnection(dsn)
	if err != nil {
		return nil, err
	}

	closeChan := make(chan *amqp.Error)
	conn.NotifyClose(closeChan)

	return &session{
		Connection: conn,
		Channel:    channel,
		closeChan:  closeChan,
	}, nil
}

func (s session) Close() error {
	if s.Connection == nil {
		return nil
	}
	return s.Connection.Close()
}

func makeConnection(dsn string) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("AMQP: unable to establish connection: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("AMQP: unable to establish channel: %v", err)
	}

	return conn, ch, nil
}
