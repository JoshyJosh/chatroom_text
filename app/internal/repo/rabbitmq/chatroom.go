package rabbitmq

import (
	"chatroom_text/internal/models"
	"chatroom_text/internal/repo"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/google/uuid"
)

type RabbitMQBroker struct {
	channel          *amqp.Channel
	queue            *amqp.Queue
	enteredChatrooms *sync.Map
	userUUID         uuid.UUID
	msgChan          chan models.WSTextMessageBytes
}

// Client is
var rabbitMQConn *amqp.Connection
var rabbitMQURL string

// Separate users use separate rabbitmq channels for communication. The node uses the same connection.
var channelMap sync.Map

const CHANNEL_LOGS_EXCHANGE = "channel_logs"
const CHANNEL_USERS_EXCHANGE = "channel_users"

func InitRabbitMQClient() error {
	slog.Info("initializing rabbitMQ")
	defer slog.Info("initialized rabbitMQ")

	rabbitMQURL = os.Getenv("RABBITMQ_URL")
	var err error
	rabbitMQConn, err = amqp.Dial(rabbitMQURL)
	if err != nil {
		err = errors.Wrap(err, "failed to connect to RabbitMQ")
		slog.Error(fmt.Sprint(err))
		return err
	}

	channel, err := rabbitMQConn.Channel()
	if err != nil {
		err = errors.Wrap(err, "failed to connect to RabbitMQ channel")
		slog.Error(fmt.Sprint(err))
		return err
	}

	err = channel.ExchangeDeclare(
		CHANNEL_LOGS_EXCHANGE, // name
		"topic",               // type
		false,                 // durable
		true,                  // auto-deleted
		false,                 // internal
		false,                 // no-wait
		nil,                   // arguments
	)
	if err != nil {
		err = errors.Wrapf(err, "failed to declare %s exchange", CHANNEL_LOGS_EXCHANGE)
		slog.Error(fmt.Sprint(err))
		return err
	}

	err = channel.ExchangeDeclare(
		CHANNEL_USERS_EXCHANGE, // name
		"topic",                // type
		false,                  // durable
		true,                   // auto-deleted
		false,                  // internal
		false,                  // no-wait
		nil,                    // arguments
	)
	if err != nil {
		err = errors.Wrapf(err, "failed to declare %s exchange", CHANNEL_USERS_EXCHANGE)
		slog.Error(fmt.Sprint(err))
		return err
	}

	return nil
}

func CloseRabbitMQClient() {
	if rabbitMQConn != nil {
		rabbitMQConn.Close()
	}
}

func getChannel(userUUID uuid.UUID) (*amqp.Channel, error) {
	var channel *amqp.Channel

	channelInterface, ok := channelMap.Load(userUUID)
	if !ok {
		var err error
		channel, err = rabbitMQConn.Channel()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get connection for user %s", userUUID)
		}
	} else {
		channel = channelInterface.(*amqp.Channel)
	}

	return channel, nil
}

func declareQueue(userUUID uuid.UUID, channel *amqp.Channel) (*amqp.Queue, error) {
	queue, err := channel.QueueDeclare(
		userUUID.String(), // name
		false,             // durable
		true,              // delete when unused
		true,              // exclusive
		false,             // no-wait
		nil,               // arguments
	)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to declare a queue")
	}

	slog.Debug(fmt.Sprintf("Connected user %s", userUUID))
	return &queue, nil
}

func GetChatroomMessageBroker(user models.User) (repo.ChatroomMessageBroker, error) {
	// @todo fix issue of channel failing on reconnect
	channel, err := getChannel(user.ID)
	if err != nil {
		return nil, err
	}

	queue, err := declareQueue(user.ID, channel)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create connection")
	}

	broker := RabbitMQBroker{
		channel:          channel,
		queue:            queue,
		userUUID:         user.ID,
		enteredChatrooms: &sync.Map{},
		msgChan:          make(chan models.WSTextMessageBytes),
	}

	return broker, nil
}

func (r RabbitMQBroker) AddUser(chatroomID uuid.UUID) error {
	err := r.channel.QueueBind(
		r.queue.Name,          // name
		chatroomID.String(),   // routing key
		CHANNEL_LOGS_EXCHANGE, // exchange
		false,                 // noWait
		nil,                   // args
	)
	if err != nil {
		return errors.Wrapf(err, "failed to bind log queue for %s", chatroomID)
	}

	err = r.channel.QueueBind(
		r.queue.Name,           // name
		chatroomID.String(),    // routing key
		CHANNEL_USERS_EXCHANGE, // exchange
		false,                  // noWait
		nil,                    // args
	)
	if err != nil {
		return errors.Wrapf(err, "failed to bind users queue for %s", chatroomID)
	}

	return nil
}

func (r RabbitMQBroker) RemoveUser(chatroomID uuid.UUID) error {
	err := r.channel.QueueUnbind(
		r.queue.Name,          // name
		chatroomID.String(),   // routing key
		CHANNEL_LOGS_EXCHANGE, // exchange
		nil,                   // args
	)
	if err != nil {
		return errors.Wrapf(err, "failed to bind queue for %s", chatroomID)
	}

	return nil
}

// @todo make the message distribute to private channels.
// DistributeMessage Distributes messages to all users in chatroom.
func (r RabbitMQBroker) DistributeMessage(ctx context.Context, chatroomID uuid.UUID, msgBytes models.WSTextMessageBytes) error {
	sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	slog.Info(fmt.Sprintf("distributing message: %s", msgBytes))

	err := r.channel.PublishWithContext(
		sendCtx,
		CHANNEL_LOGS_EXCHANGE,        // exchange
		models.MainChatUUID.String(), // routing key
		false,                        // mandatory
		false,                        // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        msgBytes,
		},
	)

	return errors.Wrapf(err, "failed to publish message: %s", msgBytes)
}

func (r RabbitMQBroker) Listen(ctx context.Context, msgBytesChan chan<- models.WSTextMessageBytes) {
	deliveryChan, err := r.channel.Consume(
		r.queue.Name,        // queue
		r.userUUID.String(), // consumer name
		false,               // autoAck
		false,               // exclusive
		false,               // noLocal
		false,               // noWait
		nil,                 // args
	)

	if err != nil {
		slog.Error(fmt.Sprint(errors.Wrapf(err, "failed to consume messages for user %s", r.userUUID)))
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case delivery := <-deliveryChan:
			msgBytesChan <- delivery.Body
			r.channel.Ack(delivery.DeliveryTag, false)
		}
	}
}

// @todo make this a thing in the interface.
func (r RabbitMQBroker) DistributeUserEntryMessage(ctx context.Context, chatroomID uuid.UUID, msgBytes models.WSUserEntry) error {
	sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	msgRawBytes, err := json.Marshal(msgBytes)
	if err != nil {
		return errors.Wrapf(err, "failed to distribute user message: %s", msgBytes.Name)
	}

	slog.Info("publishing raw bytes: %s", msgBytes)

	err = r.channel.PublishWithContext(
		sendCtx,
		CHANNEL_USERS_EXCHANGE, // exchange
		r.queue.Name,           // routing key
		false,                  // mandatory
		false,                  // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        msgRawBytes,
		},
	)
	if err != nil {
		return errors.Wrapf(err, "failed to publish message: %s", msgBytes)
	}

	return nil
}
