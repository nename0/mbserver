package mbserver

import (
	"encoding/hex"
	"fmt"
	"github.com/grid-x/modbus"
)

// AsciiFrame is the Ascii Modbus frame.
type AsciiFrame struct {
	modbus.ProtocolDataUnit
	slaveAddress byte
}

const asciiStartSlave = '>' // must use > as start character

// NewAsciiFrame converts a packet to a Ascii Modbus frame.
func NewAsciiFrame(packet []byte) (*AsciiFrame, error) {
	decoder := modbus.ASCIIClientHandler{}

	// decoder is meant for clients/masters and compares the received response to the send request
	// however this is a server/slave so we have only received a request yet.
	// Passing the packet as both request and response, makes it do all necessary checks for us.
	err := decoder.Verify(packet, packet)
	if err != nil {
		return nil, err
	}
	// parse the slaveAddress as Decode does not store it
	slaveAddress, err := readHex(packet[1:])
	if err != nil {
		return nil, err
	}
	pdu, err := decoder.Decode(packet)
	if err != nil {
		return nil, err
	}

	return &AsciiFrame{
		ProtocolDataUnit: *pdu,
		slaveAddress:     slaveAddress,
	}, nil
}

func (frame *AsciiFrame) Copy() Framer {
	copy := *frame
	return &copy
}

func (frame *AsciiFrame) Bytes() []byte {
	encoder := modbus.ASCIIClientHandler{}
	encoder.SlaveID = frame.slaveAddress

	bytes, err := encoder.Encode(&frame.ProtocolDataUnit)
	if err != nil {
		fmt.Println("Errror encoding modbus ascii frame", err)
		return make([]byte, 0)
	}
	bytes[0] = asciiStartSlave

	return bytes
}

func (frame *AsciiFrame) GetData() []byte {
	return frame.Data
}

func (frame *AsciiFrame) GetFunction() uint8 {
	return frame.FunctionCode
}

func (frame *AsciiFrame) SetData(data []byte) {
	frame.Data = data
}

func (frame *AsciiFrame) SetException(exception *Exception) {
	frame.FunctionCode = frame.FunctionCode | 0x80
	frame.Data = []byte{byte(*exception)}
}

// readHex decodes hex string to byte, e.g. "8C" => 0x8C.
func readHex(data []byte) (value byte, err error) {
	var dst [1]byte
	if _, err = hex.Decode(dst[:], data[0:2]); err != nil {
		return
	}
	value = dst[0]
	return
}
