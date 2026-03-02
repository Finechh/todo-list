package consumer

import (
	"context"
	"log"
	handlerAudit "todo_list/internal/kafka/audit/handlers"

	"github.com/IBM/sarama"
)

type Handler interface {
	Handle(ctx context.Context, data []byte) error
}

type Consumer struct {
	group   sarama.ConsumerGroup
	handler Handler
	gpoudID string
	ctx     context.Context
}

func New(brokers []string, groupID string, h *handlerAudit.Handler, ctx context.Context) (*Consumer, error) {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_8_0_0
	cfg.Consumer.Offsets.Initial = sarama.OffsetNewest

	g, err := sarama.NewConsumerGroup(brokers, groupID, cfg)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		group:   g,
		handler: h,
		gpoudID: groupID,
		ctx:     ctx,
	}, nil
}

func (c *Consumer) Run(topic string) error {
	for {
		if err := c.group.Consume(c.ctx, []string{topic}, c); err != nil {
			log.Println("kafka consume error", err)
			return err
		}
		if c.ctx.Err() != nil {
			log.Println("Context expired, stopping consumer")
			return c.ctx.Err()
		}
	}
}

func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	log.Printf("kafka consumer group %s stoped", c.gpoudID)
	return nil
}

func (c *Consumer) Setup(sarama.ConsumerGroupSession) error {
	log.Printf("kafka consumer group %s started", c.gpoudID)
	return nil
}

func (c *Consumer) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		if err := c.handler.Handle(c.ctx, msg.Value); err != nil {
			log.Println("handler error:", err)
		}
		sess.MarkMessage(msg, "")
	}
	return nil
}
