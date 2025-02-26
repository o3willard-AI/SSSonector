package tunnel

import (
	"io"
	"net"
	"reflect"
	"sync"
	"sync/atomic"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"go.uber.org/zap"
)

// Transfer handles data transfer between two connections
type Transfer struct {
	conn1  io.ReadWriteCloser
	conn2  io.ReadWriteCloser
	config *types.AppConfig
	logger *zap.Logger
	wg     sync.WaitGroup
	done   chan struct{}
	bytes  atomic.Uint64
}

// NewTransfer creates a new transfer
func NewTransfer(conn1, conn2 io.ReadWriteCloser, cfg *types.AppConfig, logger *zap.Logger) *Transfer {
	return &Transfer{
		conn1:  conn1,
		conn2:  conn2,
		config: cfg,
		logger: logger,
		done:   make(chan struct{}),
	}
}

// Start starts the transfer
func (t *Transfer) Start() error {
	if t.logger != nil {
		t.logger.Debug("Starting transfer")
	}

	// Start bidirectional copy between TCP and TUN
	t.wg.Add(2)
	go t.copy(t.conn1, t.conn2) // TCP -> TUN
	go t.copy(t.conn2, t.conn1) // TUN -> TCP

	// Wait for both copies to complete
	t.wg.Wait()

	// Use a mutex to prevent race conditions when closing the channel
	select {
	case <-t.done:
		// Channel is already closed, do nothing
	default:
		close(t.done)
	}

	if t.logger != nil {
		t.logger.Debug("Transfer complete",
			zap.Uint64("bytes_transferred", t.bytes.Load()),
		)
	}

	return nil
}

// copy copies data between TCP and TUN interfaces
func (t *Transfer) copy(dst io.Writer, src io.Reader) {
	defer t.wg.Done()

	// Use MTU from network config for buffer size
	mtu := t.config.Config.Network.MTU
	if mtu <= 0 {
		mtu = 1500 // Default MTU
	}

	// Create buffer for reading
	buf := make([]byte, mtu)

	// Get source and destination types for logging
	srcType := getConnType(src)
	dstType := getConnType(dst)

	// Log direction of data flow
	if t.logger != nil {
		t.logger.Info("Starting data transfer",
			zap.String("direction", srcType+" -> "+dstType),
			zap.Int("mtu", mtu),
		)
	}

	// Packet counters for detailed logging
	var packetCount, totalBytes uint64

	// Copy data
	for {
		select {
		case <-t.done:
			if t.logger != nil {
				t.logger.Info("Transfer stopped",
					zap.String("direction", srcType+" -> "+dstType),
					zap.Uint64("packets_transferred", packetCount),
					zap.Uint64("bytes_transferred", totalBytes),
				)
			}
			return
		default:
			// Read from source
			n, err := src.Read(buf)
			if err != nil {
				if err != io.EOF && t.logger != nil {
					t.logger.Error("Read failed",
						zap.Error(err),
						zap.String("src_type", srcType),
						zap.String("dst_type", dstType),
						zap.Uint64("packets_transferred", packetCount),
						zap.Uint64("bytes_transferred", totalBytes),
					)
				}
				return
			}

			if n > 0 {
				packetCount++
				totalBytes += uint64(n)

				// Log packet details
				if t.logger != nil {
					// Determine if this is an IP packet and log details
					if n >= 20 && (buf[0]>>4) == 4 { // IPv4
						version := buf[0] >> 4
						ihl := buf[0] & 0x0F
						totalLength := uint16(buf[2])<<8 | uint16(buf[3])
						protocol := buf[9]
						srcIP := net.IPv4(buf[12], buf[13], buf[14], buf[15])
						dstIP := net.IPv4(buf[16], buf[17], buf[18], buf[19])

						t.logger.Debug("IPv4 packet",
							zap.Uint8("version", version),
							zap.Uint8("ihl", ihl),
							zap.Uint16("total_length", totalLength),
							zap.Uint8("protocol", protocol),
							zap.String("src_ip", srcIP.String()),
							zap.String("dst_ip", dstIP.String()),
							zap.String("direction", srcType+" -> "+dstType),
							zap.Uint64("packet_number", packetCount),
							zap.Int("bytes_read", n),
						)

						// Log protocol-specific details
						switch protocol {
						case 1: // ICMP
							if n >= 24 {
								icmpType := buf[20]
								icmpCode := buf[21]
								t.logger.Debug("ICMP packet",
									zap.Uint8("type", icmpType),
									zap.Uint8("code", icmpCode),
								)
							}
						case 6: // TCP
							if n >= 40 {
								srcPort := uint16(buf[20])<<8 | uint16(buf[21])
								dstPort := uint16(buf[22])<<8 | uint16(buf[23])
								seqNum := uint32(buf[24])<<24 | uint32(buf[25])<<16 | uint32(buf[26])<<8 | uint32(buf[27])
								ackNum := uint32(buf[28])<<24 | uint32(buf[29])<<16 | uint32(buf[30])<<8 | uint32(buf[31])
								flags := buf[33]
								t.logger.Debug("TCP packet",
									zap.Uint16("src_port", srcPort),
									zap.Uint16("dst_port", dstPort),
									zap.Uint32("seq_num", seqNum),
									zap.Uint32("ack_num", ackNum),
									zap.Uint8("flags", flags),
								)
							}
						case 17: // UDP
							if n >= 28 {
								srcPort := uint16(buf[20])<<8 | uint16(buf[21])
								dstPort := uint16(buf[22])<<8 | uint16(buf[23])
								length := uint16(buf[24])<<8 | uint16(buf[25])
								t.logger.Debug("UDP packet",
									zap.Uint16("src_port", srcPort),
									zap.Uint16("dst_port", dstPort),
									zap.Uint16("length", length),
								)
							}
						}
					} else if n >= 40 && (buf[0]>>4) == 6 { // IPv6
						t.logger.Debug("IPv6 packet",
							zap.Int("bytes_read", n),
							zap.String("direction", srcType+" -> "+dstType),
							zap.Uint64("packet_number", packetCount),
						)
					} else {
						t.logger.Debug("Non-IP packet or fragment",
							zap.Int("bytes_read", n),
							zap.String("direction", srcType+" -> "+dstType),
							zap.Uint64("packet_number", packetCount),
							zap.Binary("packet_header", buf[:min(n, 16)]), // Log first 16 bytes for debugging
						)
					}
				}

				// Write immediately without buffering
				written := 0
				for written < n {
					w, err := dst.Write(buf[written:n])
					if err != nil {
						if t.logger != nil {
							t.logger.Error("Write failed",
								zap.Error(err),
								zap.Int("buffer_size", n),
								zap.Int("bytes_written", written),
								zap.String("src_type", srcType),
								zap.String("dst_type", dstType),
								zap.Uint64("packet_number", packetCount),
							)
						}
						return
					}

					written += w
					t.bytes.Add(uint64(w))
				}

				// Log successful write
				if t.logger != nil {
					t.logger.Debug("Write successful",
						zap.Int("bytes_written", n),
						zap.String("src_type", srcType),
						zap.String("dst_type", dstType),
						zap.Uint64("packet_number", packetCount),
					)
				}

				// Log periodic statistics
				if packetCount%100 == 0 && t.logger != nil {
					t.logger.Info("Transfer statistics",
						zap.String("direction", srcType+" -> "+dstType),
						zap.Uint64("packets_transferred", packetCount),
						zap.Uint64("bytes_transferred", totalBytes),
					)
				}
			}
		}
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Stop stops the transfer
func (t *Transfer) Stop() error {
	// Use a mutex to prevent race conditions when closing the channel
	select {
	case <-t.done:
		// Channel is already closed, do nothing
	default:
		close(t.done)
	}
	t.wg.Wait()
	return nil
}

// getConnType returns a string describing the type of connection
func getConnType(conn interface{}) string {
	if conn == nil {
		return "nil"
	}
	t := reflect.TypeOf(conn)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.String()
}
