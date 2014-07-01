package ice

import "io"

// TODO: This is incorrect. Need to buffer writes until the control byte is full

type prsWriter struct {
	writer io.Writer
	controlPos, controlByte uint8
}

func (s *prsWriter) writeByte(b uint8) (err error) {
	_, err = s.writer.Write([]uint8 { b })
	return
}

func (s *prsWriter) writeControlStream(b, save bool) error {
	s.controlByte >>= 1
	if b {
		s.controlByte |= 0x80
	}
	s.controlPos++

	if save {
		return s.saveControlStream()
	}

	return nil
}

func (s *prsWriter) saveControlStream() (err error) {
	if s.controlPos >= 8 {
		err = s.writeByte(s.controlByte)

		s.controlPos = 0
		s.controlByte = 0
	}

	return
}

func (s *prsWriter) Write(p []byte) (n int, err error) {
	for _, b := range p {
		s.writeControlStream(true, false)
		err = s.writeByte(b)
		if err == nil {
			err = s.saveControlStream()
		}

		if err != nil {
			break
		} else {
			n++
		}
	}

	return
}

func (s *prsWriter) Close() (err error) {
	err = s.writeControlStream(false, true)
	err = s.writeControlStream(true, true)
	if s.controlPos > 0 {
		s.controlByte = (s.controlByte << s.controlPos) >> 8
	}
	err = s.writeByte(0)
	err = s.writeByte(0)

	if c, ok := s.writer.(io.Closer); ok && err == nil {
		err = c.Close()
	}

	return
}

func newPrsWriter(writer io.Writer) *prsWriter {
	return &prsWriter{writer, 0, 0}
}
