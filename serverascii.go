package mbserver

import (
	"bytes"
	"github.com/grid-x/serial"
	"io"
	"log"
)

// ListenAscii starts the Ascii Modbus server listening to a serial device as slave with given address.
// For example:  err := s.ListenAscii(&serial.Config{Address: "/dev/ttyUSB0"}, 1)
func (s *Server) ListenAscii(serialConfig *serial.Config, slaveAddress byte) (err error) {
	port, err := serial.Open(serialConfig)
	if err != nil {
		log.Fatalf("failed to open %s: %v\n", serialConfig.Address, err)
	}

	// flush port buffers by reading until single line
	for {
		var buffer [512]byte
		n, err := port.Read(buffer[:])
		if err != nil || bytes.Count(buffer[:n], []byte("\r\n")) <= 1 {
			break
		}
	}

	s.ports = append(s.ports, port)

	s.portsWG.Add(1)
	go func() {
		defer s.portsWG.Done()
		s.acceptAsciiRequests(port, slaveAddress)
	}()

	return err
}

func (s *Server) acceptAsciiRequests(port serial.Port, slaveAddress byte) {
SkipFrameError:
	for {
		select {
		case <-s.portsCloseChan:
			return
		default:
		}

		buffer := make([]byte, 512)

		bytesRead, err := port.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Printf("serial read error %v\n", err)
			}
			return
		}

		if bytesRead != 0 {

			// Set the length of the packet to the number of read bytes.
			packet := buffer[:bytesRead]

			frame, err := NewAsciiFrame(packet)
			if err != nil {
				log.Printf("bad serial frame error %v\n", err)
				//The next line prevents RTU server from exiting when it receives a bad frame. Simply discard the erroneous
				//frame and wait for next frame by jumping back to the beginning of the 'for' loop.
				log.Printf("Keep the RTU server running!!\n")
				continue SkipFrameError
				//return
			}

			if frame.slaveAddress != slaveAddress {
				// not for us -> ignore
				continue SkipFrameError
			}

			request := &Request{port, frame}

			// not using requestChan here as raspi seems to have synchronization issues if not doing so
			// maybe reading from serial while sending on other goroutine causes problems
			// or because of single cpu on raspi
			//s.requestChan <- request

			response := s.handle(request)
			request.conn.Write(response.Bytes())
		}
	}
}
