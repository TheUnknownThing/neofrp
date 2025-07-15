package protocol

import (
	// "encoding/binary"
	"io"

	C "neofrp/common/constant"

	"github.com/charmbracelet/log"
)

const (
	ActionKeepAlive = byte(0x00) // Keep-alive message
	ActionNewCh     = byte(0x01) // Create new channel message
	ActionClose     = byte(0x02) // Close connection message
)

const (
	ReturnCodeAccepted          = 0x00
	ReturnCodeUnrecognizedToken = 0x01
	ReturnCodePortInUse         = 0x03
	ReturnCodeOtherError        = 0x04
)

func DialKeepAlive(conn io.Writer) {
	if conn == nil {
		return
	}
	// Send a keep-alive message
	if _, err := conn.Write([]byte{ActionKeepAlive}); err != nil {
		// Handle error (e.g., log it)
		log.Error("Failed to send keep-alive message", "error", err)
		return
	}
}

func DialNewCh(conn io.Writer, id C.ChannelIdentifier) {
	if conn == nil {
		return
	}
	// Send a new channel message
	log.Debug("Dialing new channel", "channel_id", id)
	// if _, err := conn.Write(append([]byte{ActionNewCh}, /*id: uint16*/)); err != nil {
	if _, err := conn.Write(append([]byte{ActionNewCh}, byte(id>>8), byte(id))); err != nil {
		// Handle error (e.g., log it)
		log.Error("Failed to send new channel message", "error", err)
		return
	}
}

func DialClose(conn io.Writer) {
	if conn == nil {
		return
	}
	// Send a close connection message
	if _, err := conn.Write([]byte{ActionClose}); err != nil {
		// Handle error (e.g., log it)
		log.Error("Failed to send close connection message", "error", err)
		return
	}
}
