// +build test

package mqtt

import (
	"time"

	cli "github.com/256dpi/gomqtt/client"
	"github.com/256dpi/gomqtt/packet"
	"github.com/pkg/errors"
)

type TestSubscriptionStream struct {
	client   *cli.Client
	messages chan interface{}
}

func (s *TestSubscriptionStream) Close() {
	if s.client != nil {
		_ = s.client.Close()
	}
	if s.messages != nil {
		close(s.messages)
	}
}

type TestSubscriptionStreamInterception func(actual *packet.Message) bool

func (s *TestSubscriptionStream) Intercept(wait time.Duration, interception TestSubscriptionStreamInterception) error {
	var deadline = time.After(wait)
	for {
		select {
		case <-deadline:
			return errors.Errorf("timeout after %v", wait)
		case msg := <-s.messages:
			switch m := msg.(type) {
			case packet.Message:
				if interception(&m) {
					return nil
				}
			case error:
				return errors.Wrapf(m, "received error")
			}
		}
	}
}

func NewTestSubscriptionStream(address, topic string, qos byte) (*TestSubscriptionStream, error) {
	var messages = make(chan interface{})

	var c = cli.New()
	c.Callback = func(msg *packet.Message, err error) error {
		if err != nil {
			messages <- err
		} else {
			messages <- *msg
		}
		return nil
	}

	cf, err := c.Connect(cli.NewConfig(address))
	if err != nil {
		_ = c.Close()
		return nil, errors.Wrap(err, "failed to connect broker")
	}
	err = cf.Wait(10 * time.Second)
	if err != nil {
		_ = c.Close()
		return nil, errors.Wrap(err, "failed to wait connecting")
	}

	sf, err := c.Subscribe(topic, packet.QOS(qos))
	if err != nil {
		_ = c.Close()
		return nil, errors.Wrap(err, "failed to subscribe topic")
	}
	err = sf.Wait(10 * time.Second)
	if err != nil {
		_ = c.Close()
		return nil, errors.Wrap(err, "failed to wait subscribing")
	}

	return &TestSubscriptionStream{
		client:   c,
		messages: messages,
	}, nil
}
