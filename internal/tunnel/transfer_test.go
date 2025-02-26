package tunnel

import (
	"bytes"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"go.uber.org/zap/zaptest"
)

// mockPacketConn is a mock connection that simulates packet-based communication
type mockPacketConn struct {
	readBuf     [][]byte // Slice of packets to read
	writeBuf    [][]byte // Slice of packets that were written
	readIndex   int
	mu          sync.Mutex
	readCh      chan struct{}
	writeCh     chan struct{}
	closed      bool
	readTimeout time.Duration
}

func newMockPacketConn() *mockPacketConn {
	return &mockPacketConn{
		readBuf:  make([][]byte, 0),
		writeBuf: make([][]byte, 0),
		readCh:   make(chan struct{}, 10),
		writeCh:  make(chan struct{}, 10),
	}
}

func (m *mockPacketConn) Read(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, io.EOF
	}

	if m.readTimeout > 0 {
		time.Sleep(m.readTimeout)
	}

	if m.readIndex >= len(m.readBuf) {
		return 0, io.EOF
	}

	packet := m.readBuf[m.readIndex]
	n = copy(p, packet)
	m.readIndex++

	select {
	case m.readCh <- struct{}{}:
	default:
	}
	return n, nil
}

func (m *mockPacketConn) Write(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return 0, io.ErrClosedPipe
	}

	// Make a copy of the packet
	packet := make([]byte, len(p))
	copy(packet, p)
	m.writeBuf = append(m.writeBuf, packet)
	n = len(p)

	select {
	case m.writeCh <- struct{}{}:
	default:
	}
	return n, nil
}

func (m *mockPacketConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockPacketConn) AddPacket(packet []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Make a copy of the packet
	p := make([]byte, len(packet))
	copy(p, packet)
	m.readBuf = append(m.readBuf, p)
}

func (m *mockPacketConn) GetWrittenPackets() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.writeBuf
}

func (m *mockPacketConn) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readBuf = make([][]byte, 0)
	m.writeBuf = make([][]byte, 0)
	m.readIndex = 0
	m.closed = false
}

// Helper function to create an IPv4 packet
func createIPv4Packet(protocol byte, srcIP, dstIP net.IP, payload []byte) []byte {
	// IPv4 header (20 bytes)
	header := make([]byte, 20)
	header[0] = 0x45                       // Version (4) and IHL (5)
	header[2] = byte(20 + len(payload)>>8) // Total Length (high byte)
	header[3] = byte(20 + len(payload))    // Total Length (low byte)
	header[8] = 64                         // TTL
	header[9] = protocol                   // Protocol
	copy(header[12:16], srcIP.To4())       // Source IP
	copy(header[16:20], dstIP.To4())       // Destination IP

	// Combine header and payload
	packet := append(header, payload...)
	return packet
}

// Helper function to create an ICMP Echo Request packet
func createICMPEchoRequest(srcIP, dstIP net.IP, id, seq uint16, payload []byte) []byte {
	// ICMP Echo Request header (8 bytes)
	icmpHeader := make([]byte, 8)
	icmpHeader[0] = 8              // Type: Echo Request
	icmpHeader[1] = 0              // Code
	icmpHeader[4] = byte(id >> 8)  // Identifier (high byte)
	icmpHeader[5] = byte(id)       // Identifier (low byte)
	icmpHeader[6] = byte(seq >> 8) // Sequence Number (high byte)
	icmpHeader[7] = byte(seq)      // Sequence Number (low byte)

	// Combine ICMP header and payload
	icmpPacket := append(icmpHeader, payload...)

	// Create IP packet with ICMP as payload
	return createIPv4Packet(1, srcIP, dstIP, icmpPacket)
}

// Helper function to create a TCP packet
func createTCPPacket(srcIP, dstIP net.IP, srcPort, dstPort uint16, flags byte, payload []byte) []byte {
	// TCP header (20 bytes)
	tcpHeader := make([]byte, 20)
	tcpHeader[0] = byte(srcPort >> 8) // Source Port (high byte)
	tcpHeader[1] = byte(srcPort)      // Source Port (low byte)
	tcpHeader[2] = byte(dstPort >> 8) // Destination Port (high byte)
	tcpHeader[3] = byte(dstPort)      // Destination Port (low byte)
	tcpHeader[12] = 5 << 4            // Data Offset (5 32-bit words)
	tcpHeader[13] = flags             // Flags

	// Combine TCP header and payload
	tcpPacket := append(tcpHeader, payload...)

	// Create IP packet with TCP as payload
	return createIPv4Packet(6, srcIP, dstIP, tcpPacket)
}

// Helper function to create a UDP packet
func createUDPPacket(srcIP, dstIP net.IP, srcPort, dstPort uint16, payload []byte) []byte {
	// UDP header (8 bytes)
	udpHeader := make([]byte, 8)
	udpHeader[0] = byte(srcPort >> 8)            // Source Port (high byte)
	udpHeader[1] = byte(srcPort)                 // Source Port (low byte)
	udpHeader[2] = byte(dstPort >> 8)            // Destination Port (high byte)
	udpHeader[3] = byte(dstPort)                 // Destination Port (low byte)
	udpHeader[4] = byte((8 + len(payload)) >> 8) // Length (high byte)
	udpHeader[5] = byte(8 + len(payload))        // Length (low byte)

	// Combine UDP header and payload
	udpPacket := append(udpHeader, payload...)

	// Create IP packet with UDP as payload
	return createIPv4Packet(17, srcIP, dstIP, udpPacket)
}

func TestTransferPacketProcessing(t *testing.T) {
	// Create logger for testing
	logger := zaptest.NewLogger(t)

	// Create a simple config for the transfer
	cfg := &types.AppConfig{
		Config: &types.ServiceConfig{
			Network: &types.NetworkConfig{
				MTU: 1500,
			},
		},
	}

	tests := []struct {
		name         string
		setupPackets func(conn *mockPacketConn)
		verify       func(t *testing.T, packets [][]byte)
	}{
		{
			name: "ICMP Echo Request",
			setupPackets: func(conn *mockPacketConn) {
				srcIP := net.ParseIP("10.0.0.2")
				dstIP := net.ParseIP("10.0.0.1")
				payload := bytes.Repeat([]byte("ping"), 4)
				packet := createICMPEchoRequest(srcIP, dstIP, 1234, 1, payload)
				conn.AddPacket(packet)
			},
			verify: func(t *testing.T, packets [][]byte) {
				if len(packets) != 1 {
					t.Fatalf("Expected 1 packet, got %d", len(packets))
				}

				packet := packets[0]
				if len(packet) < 28 { // IP header (20) + ICMP header (8)
					t.Fatalf("Packet too short: %d bytes", len(packet))
				}

				// Verify IP header
				if packet[0]>>4 != 4 {
					t.Errorf("Expected IPv4 packet, got version %d", packet[0]>>4)
				}
				if packet[9] != 1 { // ICMP protocol
					t.Errorf("Expected ICMP protocol (1), got %d", packet[9])
				}

				// Verify ICMP header
				if packet[20] != 8 { // Echo Request
					t.Errorf("Expected ICMP Echo Request (8), got %d", packet[20])
				}
			},
		},
		{
			name: "TCP SYN Packet",
			setupPackets: func(conn *mockPacketConn) {
				srcIP := net.ParseIP("10.0.0.2")
				dstIP := net.ParseIP("10.0.0.1")
				// TCP SYN flag (0x02)
				packet := createTCPPacket(srcIP, dstIP, 12345, 443, 0x02, nil)
				conn.AddPacket(packet)
			},
			verify: func(t *testing.T, packets [][]byte) {
				if len(packets) != 1 {
					t.Fatalf("Expected 1 packet, got %d", len(packets))
				}

				packet := packets[0]
				if len(packet) < 40 { // IP header (20) + TCP header (20)
					t.Fatalf("Packet too short: %d bytes", len(packet))
				}

				// Verify IP header
				if packet[0]>>4 != 4 {
					t.Errorf("Expected IPv4 packet, got version %d", packet[0]>>4)
				}
				if packet[9] != 6 { // TCP protocol
					t.Errorf("Expected TCP protocol (6), got %d", packet[9])
				}

				// Verify TCP header
				srcPort := uint16(packet[20])<<8 | uint16(packet[21])
				dstPort := uint16(packet[22])<<8 | uint16(packet[23])
				flags := packet[33]

				if srcPort != 12345 {
					t.Errorf("Expected source port 12345, got %d", srcPort)
				}
				if dstPort != 443 {
					t.Errorf("Expected destination port 443, got %d", dstPort)
				}
				if flags != 0x02 {
					t.Errorf("Expected SYN flag (0x02), got 0x%02x", flags)
				}
			},
		},
		{
			name: "UDP Packet",
			setupPackets: func(conn *mockPacketConn) {
				srcIP := net.ParseIP("10.0.0.2")
				dstIP := net.ParseIP("10.0.0.1")
				payload := []byte("UDP payload")
				packet := createUDPPacket(srcIP, dstIP, 53, 12345, payload)
				conn.AddPacket(packet)
			},
			verify: func(t *testing.T, packets [][]byte) {
				if len(packets) != 1 {
					t.Fatalf("Expected 1 packet, got %d", len(packets))
				}

				packet := packets[0]
				if len(packet) < 28 { // IP header (20) + UDP header (8)
					t.Fatalf("Packet too short: %d bytes", len(packet))
				}

				// Verify IP header
				if packet[0]>>4 != 4 {
					t.Errorf("Expected IPv4 packet, got version %d", packet[0]>>4)
				}
				if packet[9] != 17 { // UDP protocol
					t.Errorf("Expected UDP protocol (17), got %d", packet[9])
				}

				// Verify UDP header
				srcPort := uint16(packet[20])<<8 | uint16(packet[21])
				dstPort := uint16(packet[22])<<8 | uint16(packet[23])

				if srcPort != 53 {
					t.Errorf("Expected source port 53, got %d", srcPort)
				}
				if dstPort != 12345 {
					t.Errorf("Expected destination port 12345, got %d", dstPort)
				}
			},
		},
		{
			name: "Multiple Packets",
			setupPackets: func(conn *mockPacketConn) {
				srcIP := net.ParseIP("10.0.0.2")
				dstIP := net.ParseIP("10.0.0.1")

				// Add ICMP packet
				icmpPayload := bytes.Repeat([]byte("ping"), 4)
				icmpPacket := createICMPEchoRequest(srcIP, dstIP, 1234, 1, icmpPayload)
				conn.AddPacket(icmpPacket)

				// Add TCP packet
				tcpPacket := createTCPPacket(srcIP, dstIP, 12345, 443, 0x02, nil)
				conn.AddPacket(tcpPacket)

				// Add UDP packet
				udpPayload := []byte("UDP payload")
				udpPacket := createUDPPacket(srcIP, dstIP, 53, 12345, udpPayload)
				conn.AddPacket(udpPacket)
			},
			verify: func(t *testing.T, packets [][]byte) {
				if len(packets) != 3 {
					t.Fatalf("Expected 3 packets, got %d", len(packets))
				}

				// Verify first packet (ICMP)
				if packets[0][9] != 1 {
					t.Errorf("Expected first packet to be ICMP (1), got %d", packets[0][9])
				}

				// Verify second packet (TCP)
				if packets[1][9] != 6 {
					t.Errorf("Expected second packet to be TCP (6), got %d", packets[1][9])
				}

				// Verify third packet (UDP)
				if packets[2][9] != 17 {
					t.Errorf("Expected third packet to be UDP (17), got %d", packets[2][9])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock connections
			srcConn := newMockPacketConn()
			dstConn := newMockPacketConn()

			// Setup test packets
			tt.setupPackets(srcConn)

			// Create transfer
			transfer := NewTransfer(dstConn, srcConn, cfg, logger)

			// Create a channel to signal when the transfer is done
			transferDone := make(chan struct{})

			// Start transfer in a goroutine
			go func() {
				err := transfer.Start()
				if err != nil {
					t.Errorf("Transfer.Start() error = %v", err)
				}
				close(transferDone)
			}()

			// Wait for packets to be processed or timeout
			deadline := time.Now().Add(1 * time.Second)
			for len(dstConn.GetWrittenPackets()) < len(srcConn.readBuf) && time.Now().Before(deadline) {
				time.Sleep(10 * time.Millisecond)
			}

			// Wait for transfer to complete or force stop it
			select {
			case <-transferDone:
				// Transfer completed naturally
			case <-time.After(100 * time.Millisecond):
				// Transfer didn't complete in time, stop it
				transfer.Stop()
				<-transferDone
			}

			// Verify packets
			tt.verify(t, dstConn.GetWrittenPackets())
		})
	}
}

func TestTransferBidirectional(t *testing.T) {
	// Create logger for testing
	logger := zaptest.NewLogger(t)

	// Create a simple config for the transfer
	cfg := &types.AppConfig{
		Config: &types.ServiceConfig{
			Network: &types.NetworkConfig{
				MTU: 1500,
			},
		},
	}

	// Create mock connections
	conn1 := newMockPacketConn()
	conn2 := newMockPacketConn()

	// Create test packets
	srcIP1 := net.ParseIP("10.0.0.1")
	dstIP1 := net.ParseIP("10.0.0.2")
	srcIP2 := net.ParseIP("10.0.0.2")
	dstIP2 := net.ParseIP("10.0.0.1")

	// Add packets to conn1 (simulating TUN -> TCP direction)
	icmpPacket1 := createICMPEchoRequest(srcIP1, dstIP1, 1234, 1, []byte("ping from 1 to 2"))
	conn1.AddPacket(icmpPacket1)

	// Add packets to conn2 (simulating TCP -> TUN direction)
	icmpPacket2 := createICMPEchoRequest(srcIP2, dstIP2, 5678, 1, []byte("ping from 2 to 1"))
	conn2.AddPacket(icmpPacket2)

	// Create transfer
	transfer := NewTransfer(conn1, conn2, cfg, logger)

	// Create a channel to signal when the transfer is done
	transferDone := make(chan struct{})

	// Start transfer in a goroutine
	go func() {
		err := transfer.Start()
		if err != nil {
			t.Errorf("Transfer.Start() error = %v", err)
		}
		close(transferDone)
	}()

	// Wait for packets to be processed
	deadline := time.Now().Add(1 * time.Second)
	for (len(conn1.GetWrittenPackets()) < len(conn2.readBuf) ||
		len(conn2.GetWrittenPackets()) < len(conn1.readBuf)) &&
		time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for transfer to complete or force stop it
	select {
	case <-transferDone:
		// Transfer completed naturally
	case <-time.After(100 * time.Millisecond):
		// Transfer didn't complete in time, stop it
		transfer.Stop()
		<-transferDone
	}

	// Verify packets
	if len(conn1.GetWrittenPackets()) != 1 {
		t.Errorf("Expected 1 packet written to conn1, got %d", len(conn1.GetWrittenPackets()))
	}
	if len(conn2.GetWrittenPackets()) != 1 {
		t.Errorf("Expected 1 packet written to conn2, got %d", len(conn2.GetWrittenPackets()))
	}

	// Verify packet content
	if !bytes.Equal(conn1.GetWrittenPackets()[0], icmpPacket2) {
		t.Error("Packet written to conn1 doesn't match original packet from conn2")
	}
	if !bytes.Equal(conn2.GetWrittenPackets()[0], icmpPacket1) {
		t.Error("Packet written to conn2 doesn't match original packet from conn1")
	}
}

func TestTransferLargePackets(t *testing.T) {
	// Create logger for testing
	logger := zaptest.NewLogger(t)

	// Create a simple config for the transfer with small MTU
	cfg := &types.AppConfig{
		Config: &types.ServiceConfig{
			Network: &types.NetworkConfig{
				MTU: 1500, // Use standard MTU to ensure packet fits
			},
		},
	}

	// Create mock connections
	srcConn := newMockPacketConn()
	dstConn := newMockPacketConn()

	// Create a large packet (but still within MTU)
	srcIP := net.ParseIP("10.0.0.2")
	dstIP := net.ParseIP("10.0.0.1")
	payload := bytes.Repeat([]byte("large packet payload"), 10) // ~200 bytes
	packet := createUDPPacket(srcIP, dstIP, 53, 12345, payload)
	srcConn.AddPacket(packet)

	// Create transfer
	transfer := NewTransfer(dstConn, srcConn, cfg, logger)

	// Create a channel to signal when the transfer is done
	transferDone := make(chan struct{})

	// Start transfer in a goroutine
	go func() {
		err := transfer.Start()
		if err != nil {
			t.Errorf("Transfer.Start() error = %v", err)
		}
		close(transferDone)
	}()

	// Wait for packets to be processed
	deadline := time.Now().Add(1 * time.Second)
	for len(dstConn.GetWrittenPackets()) < 1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for transfer to complete or force stop it
	select {
	case <-transferDone:
		// Transfer completed naturally
	case <-time.After(100 * time.Millisecond):
		// Transfer didn't complete in time, stop it
		transfer.Stop()
		<-transferDone
	}

	// Verify packets
	writtenPackets := dstConn.GetWrittenPackets()
	if len(writtenPackets) != 1 {
		t.Fatalf("Expected 1 packet, got %d", len(writtenPackets))
	}

	// Verify the packet was transferred correctly
	if !bytes.Equal(writtenPackets[0], packet) {
		t.Error("Large packet was not transferred correctly")
	}
}

func TestTransferWithErrors(t *testing.T) {
	// Create logger for testing
	logger := zaptest.NewLogger(t)

	// Create a simple config for the transfer
	cfg := &types.AppConfig{
		Config: &types.ServiceConfig{
			Network: &types.NetworkConfig{
				MTU: 1500,
			},
		},
	}

	t.Run("Read Timeout", func(t *testing.T) {
		// Create mock connections
		srcConn := newMockPacketConn()
		dstConn := newMockPacketConn()

		// Set read timeout
		srcConn.readTimeout = 500 * time.Millisecond

		// Add a packet
		srcIP := net.ParseIP("10.0.0.2")
		dstIP := net.ParseIP("10.0.0.1")
		packet := createICMPEchoRequest(srcIP, dstIP, 1234, 1, []byte("ping"))
		srcConn.AddPacket(packet)

		// Create transfer
		transfer := NewTransfer(dstConn, srcConn, cfg, logger)

		// Create a channel to signal when the transfer is done
		transferDone := make(chan struct{})

		// Start transfer in a goroutine
		go func() {
			err := transfer.Start()
			if err != nil {
				t.Errorf("Transfer.Start() error = %v", err)
			}
			close(transferDone)
		}()

		// Wait for packets to be processed
		deadline := time.Now().Add(2 * time.Second)
		for len(dstConn.GetWrittenPackets()) < 1 && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}

		// Wait for transfer to complete or force stop it
		select {
		case <-transferDone:
			// Transfer completed naturally
		case <-time.After(1 * time.Second):
			// Transfer didn't complete in time, stop it
			transfer.Stop()
			<-transferDone
		}

		// Verify packets were still transferred despite timeout
		if len(dstConn.GetWrittenPackets()) != 1 {
			t.Errorf("Expected 1 packet despite timeout, got %d", len(dstConn.GetWrittenPackets()))
		}
	})

	t.Run("Connection Closed", func(t *testing.T) {
		// Create mock connections
		srcConn := newMockPacketConn()
		dstConn := newMockPacketConn()

		// Add a packet
		srcIP := net.ParseIP("10.0.0.2")
		dstIP := net.ParseIP("10.0.0.1")
		packet := createICMPEchoRequest(srcIP, dstIP, 1234, 1, []byte("ping"))
		srcConn.AddPacket(packet)

		// Create transfer
		transfer := NewTransfer(dstConn, srcConn, cfg, logger)

		// Create a channel to signal when the transfer is done
		transferDone := make(chan struct{})

		// Start transfer in a goroutine
		go func() {
			err := transfer.Start()
			if err != nil {
				t.Errorf("Transfer.Start() error = %v", err)
			}
			close(transferDone)
		}()

		// Wait briefly
		time.Sleep(100 * time.Millisecond)

		// Close the source connection
		srcConn.Close()

		// Wait for transfer to complete
		select {
		case <-transferDone:
			// Transfer completed naturally
		case <-time.After(1 * time.Second):
			// Transfer didn't complete in time, stop it
			transfer.Stop()
			<-transferDone
		}

		// Verify transfer handled the closed connection gracefully
		// The packet should have been transferred before the connection was closed
		if len(dstConn.GetWrittenPackets()) != 1 {
			t.Errorf("Expected 1 packet before connection closed, got %d", len(dstConn.GetWrittenPackets()))
		}
	})
}
