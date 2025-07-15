package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	C "neofrp/common/constant"
	"neofrp/common/multidialer"
	"neofrp/common/safemap"

	"github.com/charmbracelet/log"
)

type TCPPortListener struct {
	TaggedPort C.TaggedPort
	PortMap    *safemap.SafeMap[C.TaggedPort, C.SessionIndexCompound]
}

func (l *TCPPortListener) Start() error {
	listener, err := net.Listen("tcp", net.JoinHostPort("0.0.0.0", fmt.Sprint(l.TaggedPort.Port)))
	if err != nil {
		return err
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Warnf("Failed to accept connection for port %s: %v", l.TaggedPort.String(), err)
				continue
			}
			comp, ac := l.PortMap.Get(l.TaggedPort)
			if !ac {
				log.Warnf("Failed to get session for port %s: %v", l.TaggedPort.String(), err)
				conn.Close()
				continue
			}
			log.Infof("Accepted connection for port %s from %s", l.TaggedPort.String(), conn.RemoteAddr())
			if comp.Session == nil {
				log.Warnf("Session is nil for port %s, closing connection", l.TaggedPort.String())
				conn.Close()
				continue
			}
			// Handle the connection with the session
			newConn, err := (*comp.Session).OpenStream(context.Background())
			if err != nil {
				log.Errorf("Failed to open stream for session %v: %v", comp.Session, err)
				conn.Close()
				continue
			}
			// First write in where to send the data
			_, err = newConn.Write(l.TaggedPort.Bytes())
			if err != nil {
				log.Errorf("Failed to write tagged port to new connection: %v", err)
				newConn.Close()
				conn.Close()
				continue
			}
			// Now use connection copy to relay data
			go io.Copy(newConn, conn)
			go io.Copy(conn, newConn)
		}
	}()
	return nil
}

type UDPPortListener struct {
	TaggedPort C.TaggedPort
	PortMap    *safemap.SafeMap[C.TaggedPort, C.SessionIndexCompound]
	SourceMap  *safemap.SafeMap[*net.UDPAddr, *multidialer.Stream]
}

func (l *UDPPortListener) Start() error {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: int(l.TaggedPort.Port),
	})
	if err != nil {
		return fmt.Errorf("failed to listen on UDP port %s: %w", l.TaggedPort.String(), err)
	}
	go func() {
		buf := make([]byte, 4096)
		for {
			n, addr, err := conn.ReadFromUDP(buf)
			if err != nil {
				log.Warnf("Failed to read from UDP port %s: %v", l.TaggedPort.String(), err)
				continue
			}
			log.Infof("Received %d bytes from %s on UDP port %s", n, addr.String(), l.TaggedPort.String())
			// Handle the received data
			// First, check whether an existing stream exists for this address
			stream, exists := l.SourceMap.Get(addr)
			if exists {
				_, err = (*stream).Write(buf[:n])
				if err != nil {
					log.Errorf("Failed to write to existing stream for %s: %v", addr.String(), err)
					// Clean up the failed stream
					(*stream).Close()
					l.SourceMap.Delete(addr)
					continue
				}
			} else {
				comp, ac := l.PortMap.Get(l.TaggedPort)
				if !ac {
					log.Warnf("Failed to get session for port %s: %v", l.TaggedPort.String(), err)
					continue
				}
				if comp.Session == nil {
					log.Warnf("Session is nil for port %s, ignoring data from %s", l.TaggedPort.String(), addr.String())
					continue
				}
				// Open a new stream for this address
				newStream, err := (*comp.Session).OpenStream(context.Background())
				if err != nil {
					log.Errorf("Failed to open stream for session %v: %v", comp.Session, err)
					continue
				}
				// Write the tagged port to the new stream
				_, err = newStream.Write(l.TaggedPort.Bytes())
				if err != nil {
					log.Errorf("Failed to write tagged port to new stream: %v", err)
					newStream.Close()
					continue
				}
				// Store the new stream in the source map
				l.SourceMap.Set(addr, &newStream)
				log.Infof("Opened new stream for %s on UDP port %s", addr.String(), l.TaggedPort.String())

				// Start a goroutine to handle responses from the stream back to the UDP client
				go l.handleUDPStreamResponse(conn, addr, &newStream)

				time.Sleep(C.RelayInterval)
				// Now write the data to the new stream
				_, err = newStream.Write(buf[:n])
				if err != nil {
					log.Errorf("Failed to write data to new stream for %s: %v", addr.String(), err)
					newStream.Close()
					l.SourceMap.Delete(addr)
					continue
				}
			}
		}
	}()
	return nil
}

// handleUDPStreamResponse handles responses from the stream back to the UDP client
func (l *UDPPortListener) handleUDPStreamResponse(conn *net.UDPConn, clientAddr *net.UDPAddr, stream *multidialer.Stream) {
	defer func() {
		(*stream).Close()
		l.SourceMap.Delete(clientAddr)
		log.Infof("Closed stream for %s on UDP port %s", clientAddr.String(), l.TaggedPort.String())
	}()

	buf := make([]byte, 4096)
	lastActivity := time.Now()
	timeout := 60 * time.Second // 60 second timeout for idle streams

	for {
		// Check for timeout
		if time.Since(lastActivity) > timeout {
			log.Infof("Stream timeout for %s on UDP port %s", clientAddr.String(), l.TaggedPort.String())
			return
		}

		// Set a read deadline to prevent blocking indefinitely
		deadline := time.Now().Add(1 * time.Second)
		if netConn, ok := (*stream).(interface{ SetReadDeadline(time.Time) error }); ok {
			netConn.SetReadDeadline(deadline)
		}

		n, err := (*stream).Read(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // Read timeout, check for overall timeout and continue
			}
			if err != io.EOF {
				log.Errorf("Failed to read response from stream for %s: %v", clientAddr.String(), err)
			}
			return
		}

		if n > 0 {
			lastActivity = time.Now()
			// Send the response back to the original UDP client
			_, err = conn.WriteToUDP(buf[:n], clientAddr)
			if err != nil {
				log.Errorf("Failed to send response to UDP client %s: %v", clientAddr.String(), err)
				return
			}
			log.Infof("Sent %d bytes response to %s on UDP port %s", n, clientAddr.String(), l.TaggedPort.String())
		}
	}
}
