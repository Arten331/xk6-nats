package nats

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/grafana/sobek" // Switched back to sobek
	"github.com/mitchellh/mapstructure"
	natsio "github.com/nats-io/nats.go"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
)

func init() {
	modules.Register("k6/x/nats", new(RootModule))
}

type RootModule struct{}

type Nats struct {
	conn    *natsio.Conn
	vu      modules.VU
	exports map[string]interface{}
}

// Ensure the interfaces are implemented correctly.
var (
	_ modules.Instance = &Nats{}
	_ modules.Module   = &RootModule{}
)

func (r *RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	mi := &Nats{
		vu:      vu,
		exports: make(map[string]interface{}),
	}

	mi.exports["Nats"] = mi.client

	return mi
}

func (mi *Nats) Exports() modules.Exports {
	return modules.Exports{
		Named: mi.exports,
	}
}

func (n *Nats) client(c sobek.ConstructorCall) *sobek.Object {
	rt := n.vu.Runtime()

	var cfg Configuration
	// Use mapstructure to decode the JS object (exported as a map) into the Go struct.
	err := mapstructure.Decode(c.Argument(0).Export(), &cfg)
	if err != nil {
		common.Throw(rt, fmt.Errorf("Nats constructor expects a Configuration object as its argument: %w", err))
	}

	natsOptions := natsio.GetDefaultOptions()
	natsOptions.Servers = cfg.Servers

	switch cfg.Auth.Strategy {
	case "unsafe":
		if cfg.Auth.Unsafe {
			natsOptions.TLSConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
	case "user_password":
		if cfg.Auth.Username == "" || cfg.Auth.Password == "" {
			common.Throw(rt, fmt.Errorf("username and password are required for 'user_password' strategy"))
		}
		natsOptions.User = cfg.Auth.Username
		natsOptions.Password = cfg.Auth.Password
	case "token":
		if cfg.Auth.Token == "" {
			common.Throw(rt, fmt.Errorf("token is required for 'token' strategy"))
		}
		natsOptions.Token = cfg.Auth.Token
	default:
		if cfg.Auth.Unsafe {
			natsOptions.TLSConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
	}

	conn, err := natsOptions.Connect()
	if err != nil {
		common.Throw(rt, err)
	}

	return rt.ToValue(&Nats{
		vu:   n.vu,
		conn: conn,
	}).ToObject(rt)
}

func (n *Nats) Close() {
	if n.conn != nil {
		n.conn.Close()
	}
}

func (n *Nats) PublishWithHeaders(topic, message string, headers map[string]string) error {
	if n.conn == nil {
		return fmt.Errorf("the connection is not valid")
	}

	h := natsio.Header{}
	for k, v := range headers {
		h.Add(k, v)
	}

	return n.conn.PublishMsg(&natsio.Msg{
		Subject: topic,
		Reply:   "",
		Data:    []byte(message),
		Header:  h,
	})
}

func (n *Nats) PublishMsg(msg *Message) error {
	if n.conn == nil {
		return fmt.Errorf("the connection is not valid")
	}

	raw := msg.Raw
	if raw == nil {
		raw = []byte(msg.Data)
	}

	h := natsio.Header{}
	for k, v := range msg.Header {
		h.Add(k, v)
	}

	return n.conn.PublishMsg(&natsio.Msg{
		Subject: msg.Topic,
		Reply:   "",
		Data:    raw,
		Header:  h,
	})
}

func (n *Nats) Publish(topic, message string) error {
	if n.conn == nil {
		return fmt.Errorf("the connection is not valid")
	}

	return n.conn.Publish(topic, []byte(message))
}

func (n *Nats) Subscribe(topic string, handler MessageHandler) (*Subscription, error) {
	if n.conn == nil {
		return nil, fmt.Errorf("the connection is not valid")
	}

	sub, err := n.conn.Subscribe(topic, func(msg *natsio.Msg) {
		msg.Ack()
		h := make(map[string]string)
		for k := range msg.Header {
			h[k] = msg.Header.Get(k)
		}

		message := Message{
			Raw:    msg.Data,
			Data:   string(msg.Data),
			Topic:  msg.Subject,
			Header: h,
		}
		handler(message)
	})

	if err != nil {
		return nil, err
	}

	subscription := Subscription{
		Close: func() error {
			return sub.Unsubscribe()
		},
	}

	return &subscription, err
}

func (n *Nats) JetStreamSetup(streamConfig *natsio.StreamConfig) error {
	if n.conn == nil {
		return fmt.Errorf("the connection is not valid")
	}

	js, err := n.conn.JetStream()
	if err != nil {
		return fmt.Errorf("cannot acquire jetstream context: %w", err)
	}

	stream, _ := js.StreamInfo(streamConfig.Name)
	if stream == nil {
		_, err = js.AddStream(streamConfig)
	} else {
		_, err = js.UpdateStream(streamConfig)
	}

	return err
}

func (n *Nats) JetStreamDelete(name string) error {
	if n.conn == nil {
		return fmt.Errorf("the connection is not valid")
	}

	js, err := n.conn.JetStream()
	if err != nil {
		return fmt.Errorf("cannot acquire jetstream context: %w", err)
	}

	return js.DeleteStream(name)
}

func (n *Nats) JetStreamPublish(topic string, message string) error {
	if n.conn == nil {
		return fmt.Errorf("the connection is not valid")
	}

	js, err := n.conn.JetStream()
	if err != nil {
		return fmt.Errorf("cannot acquire jetstream context: %w", err)
	}

	_, err = js.Publish(topic, []byte(message))

	return err
}

func (n *Nats) JetStreamPublishWithHeaders(topic, message string, headers map[string]string) error {
	if n.conn == nil {
		return fmt.Errorf("the connection is not valid")
	}

	h := natsio.Header{}
	for k, v := range headers {
		h.Add(k, v)
	}

	js, err := n.conn.JetStream()
	if err != nil {
		return fmt.Errorf("cannot acquire jetstream context: %w", err)
	}

	_, err = js.PublishMsg(&natsio.Msg{
		Subject: topic,
		Reply:   "",
		Data:    []byte(message),
		Header:  h,
	})

	return err
}

func (n *Nats) JetStreamPublishMsg(msg *Message) error {
	if n.conn == nil {
		return fmt.Errorf("the connection is not valid")
	}

	raw := msg.Raw
	if raw == nil {
		raw = []byte(msg.Data)
	}

	h := natsio.Header{}
	for k, v := range msg.Header {
		h.Add(k, v)
	}

	js, err := n.conn.JetStream()
	if err != nil {
		return fmt.Errorf("cannot acquire jetstream context: %w", err)
	}

	_, err = js.PublishMsg(&natsio.Msg{
		Subject: msg.Topic,
		Reply:   "",
		Data:    raw,
		Header:  h,
	})

	return err
}

func (n *Nats) JetStreamSubscribe(topic string, handler MessageHandler) (*Subscription, error) {
	if n.conn == nil {
		return nil, fmt.Errorf("the connection is not valid")
	}

	js, err := n.conn.JetStream()
	if err != nil {
		return nil, fmt.Errorf("cannot acquire jetstream context: %w", err)
	}

	sub, err := js.Subscribe(topic, func(msg *natsio.Msg) {
		msg.Ack()
		h := make(map[string]string)
		for k := range msg.Header {
			h[k] = msg.Header.Get(k)
		}

		message := Message{
			Raw:    msg.Data,
			Data:   string(msg.Data),
			Topic:  msg.Subject,
			Header: h,
		}
		handler(message)
	})

	if err != nil {
		return nil, err
	}

	subscription := Subscription{
		Close: func() error {
			return sub.Unsubscribe()
		},
	}

	return &subscription, err
}

func (n *Nats) Request(subject, data string, headers map[string]string) (Message, error) {
	if n.conn == nil {
		return Message{}, fmt.Errorf("the connection is not valid")
	}

	msg := &natsio.Msg{
		Subject: subject,
		Data:    []byte(data),
	}

	if len(headers) > 0 {
		msg.Header = natsio.Header{}
		for k, v := range headers {
			msg.Header.Add(k, v)
		}
	}

	resp, err := n.conn.RequestMsg(msg, 5*time.Second)
	if err != nil {
		return Message{}, err
	}

	respHeaders := make(map[string]string)
	for k := range resp.Header {
		respHeaders[k] = resp.Header.Get(k)
	}

	return Message{
		Raw:    resp.Data,
		Data:   string(resp.Data),
		Topic:  resp.Subject,
		Header: respHeaders,
	}, nil
}

type Configuration struct {
	Servers []string
	Auth    Auth
}

type Auth struct {
	Unsafe   bool
	Strategy string
	Token    string
	Username string
	Password string
}

type Message struct {
	Raw    []byte
	Data   string
	Topic  string
	Header map[string]string
}

type Subscription struct {
	Close func() error
}

type MessageHandler func(Message)
