package cdclient

import (
	"errors"
	"net"
	"sync"
	"time"
)

type UDPMode byte

const (
	UDPPlainText UDPMode = iota
	UDPSign
	UDPEncrypt
)

type UDPClientOptions struct {
	// Mode determines whether data is signed, encrypted or sent
	// in plain text. When zero no encryption is used.
	Mode UDPMode
	// Username and password for the "Sign" and "Encrypt" modes.
	Username, Password string
	// Size of the send buffer. When zero, DefaultBufferSize is used.
	BufferSize int
}

// A udp client that buffers metrics to write complete udp packets.
// The client is safe to use from multiple goroutines concurrently.
type UDPClient struct {
	lock      sync.Mutex
	conn      *net.UDPConn
	packet    Packet
	tmpValues []float64
}

// Dial connects to the collectd server at address. "address" must be a network
// address accepted by net.Dial().
func DialUDP(address string, opts UDPClientOptions) (*UDPClient, error) {
	c := &UDPClient{}
	err := c.Reconnect(address, opts)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *UDPClient) Reconnect(address string, opts UDPClientOptions) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.conn != nil {
		// best effort flush
		_ = c.flush()
	}

	var err error
	var packet Packet

	if opts.BufferSize == 0 {
		opts.BufferSize = DefaultBufferSize
	}

	switch opts.Mode {
	case UDPPlainText:
		packet = NewPlainTextPacketSize(opts.BufferSize)
	case UDPSign:
		packet, err = NewSignedPacketSize(opts.Username, opts.Password, opts.BufferSize)
		if err != nil {
			return err
		}
	case UDPEncrypt:
		packet, err = NewEncryptedPacketSize(opts.Username, opts.Password, opts.BufferSize)
		if err != nil {
			return err
		}
	default:
		return errors.New("unsupport client mode")
	}

	conn, err := net.Dial("udp", address)
	if err != nil {
		return err
	}

	if c.conn != nil {
		_ = c.conn.Close()
	}

	c.conn = conn.(*net.UDPConn)
	c.packet = packet
	return nil
}

func (c *UDPClient) AddValues(m *Metric, t time.Time, values ...float64) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	// This copy allows the go compiler to avoid an allocation.
	c.tmpValues = append(c.tmpValues[:0], values...)
	return c.addValueList(ValueList{
		Metric: m,
		Time:   t,
		Values: c.tmpValues,
	})
}

func (c *UDPClient) AddValueList(v ValueList) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.addValueList(v)
}

func (c *UDPClient) addValueList(v ValueList) error {
	err := c.packet.AddValueList(v)
	if errors.Is(err, ErrPacketFull) {
		err = c.flush()
		if err != nil {
			c.conn.Close()
			return err
		}
		return c.packet.AddValueList(v)
	}
	return err
}

func (c *UDPClient) flush() error {
	buf := c.packet.Finalize()
	if len(buf) == 0 {
		return nil
	}
	_, err := c.conn.Write(buf)
	// unconditionally reset the packet state,
	// it is the same as dropping the packet
	// on the wire.
	c.packet.Reset()
	return err
}

func (c *UDPClient) Flush() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.flush()
}

func (c *UDPClient) Close() error {
	return c.conn.Close()
}
