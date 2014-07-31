package ice

import "io"

// Reference: https://github.com/Grumbel/rfactortools/blob/master/other/quickbms/src/compression/prs.cpp
// (modification: long copy sizes are readByte() + 10, not +1 as that source mentions)

const prsBufferSize = 0x10000
const prsBufferThreshold = prsBufferSize - 0x400
const prsBufferLookbehind = 0x2000

type prsReader struct {
	reader io.ReadSeeker
	controlPos, controlByte uint8

	buffer [prsBufferSize]uint8
	byteBuffer [1]uint8
	outputBufferPosition int
	outputPosition int
	size, position int64
	err error
}

func (s *prsReader) readByte() (ret uint8, err error) {
	_, err = s.reader.Read(s.byteBuffer[:])
	ret = s.byteBuffer[0]
	return
}

func (s *prsReader) consumeControlStream() (ret bool, err error) {
	s.controlPos--
	if s.controlPos == 0 {
		s.controlByte, err = s.readByte()
		s.controlPos = 8
	}

	ret = (s.controlByte & 1) == 1
	s.controlByte >>= 1
	return
}

func (s *prsReader) decompress() error {
	if flag, err := s.consumeControlStream(); err == nil && flag {
		// Read byte directly from bytestream

		b, err := s.readByte()
		if err != nil {
			return err
		} else {
			s.queueOutput(b, true)
		}
	} else if err == nil {
		// Copy from sliding output window

		var offset int
		var size int
		if flag, err = s.consumeControlStream(); err == nil && flag {
			// Long copy
			var b, lsb, msb uint8

			lsb, err = s.readByte()
			msb, err = s.readByte()
			offset = int((uint16(msb) << 8) | uint16(lsb))

			if err != nil {
				return err
			}

			if offset == 0 {
				return io.EOF
			}

			size = int(lsb & 0x07)
			offset = int(int32(uint32(offset >> 3) | 0xffffe000))

			if size == 0 {
				b, err = s.readByte()
				size = int(b) + 10
			} else {
				size += 2
			}
		} else if err == nil {
			// Short copy
			for i := 0; i < 2; i++ {
				size <<= 1
				if flag, err = s.consumeControlStream(); flag {
					size |= 1
				}
			}
			size += 2

			var b uint8
			b, err = s.readByte()
			offset = int(int32(uint32(b) | 0xffffff00))
		}

		if err == nil {
			bufferPos := s.outputBufferPosition
			for i := 0; i < size; i++ {
				var b uint8 = 0
				pos := offset + i + bufferPos
				if pos < s.outputBufferPosition {
					b = s.buffer[offset + i + bufferPos]
				}
				s.queueOutput(b, false)
			}
			s.flushOutput()
		} else {
			return err
		}
	} else {
		return err
	}

	return nil
}

func (s *prsReader) queueOutput(b uint8, flush bool) {
	s.buffer[s.outputBufferPosition] = b
	s.outputBufferPosition++

	if flush {
		s.flushOutput()
	}
}

func (s *prsReader) flushOutput() {
	// Flush our buffer every 16KB
	if s.outputBufferPosition > prsBufferThreshold {
		diff := s.outputBufferPosition - prsBufferLookbehind

		copy(s.buffer[:prsBufferLookbehind], s.buffer[diff:])
		s.outputPosition -= diff
		s.outputBufferPosition = prsBufferLookbehind
	}
}

func (s *prsReader) Read(p []byte) (n int, err error) {
	err = s.err

	if s.position >= s.size {
		err = io.EOF
	}

	// Buffer decompressed output
	for len(p) > 0 && err == nil {
		if s.outputBufferPosition <= s.outputPosition {
			if err = s.decompress(); err != nil {
				break
			}
		}

		if int64(len(p)) > s.size - s.position {
			p = p[:int(s.size - s.position)]
		}

		read := copy(p, s.buffer[s.outputPosition:s.outputBufferPosition])
		n += read
		s.outputPosition += read
		s.position += int64(read)
		p = p[read:]

		if s.position >= s.size {
			break
		}
	}

	s.err = err

	if err == io.EOF && s.position < s.size {
		read := int(s.size - s.position)
		if read > len(p) {
			read = len(p)
		}
		p = p[:read]

		for i := range p {
			p[i] = 0
		}

		n += read
		s.position += int64(read)

		err = nil
	}

	return
}

func (s *prsReader) Seek(offset int64, whence int) (pos int64, err error) {
	// Determine offset...
	switch whence {
		case 1:
			offset += int64(s.position)
		case 2:
			offset += s.Size()
	}

	// Rewind to beginning...
	if offset < int64(s.position) {
		s.reader.Seek(0, 0)
		s.controlPos = 1
		s.outputBufferPosition = 0
		s.outputPosition = 0
		s.position = 0
		s.err = nil
	}

	// Then read forward until we reach our destination
	for offset > int64(s.position) && err == nil {
		diff := offset - int64(s.position)
		var buffer [0x1000]uint8

		if diff > int64(len(buffer)) {
			diff = int64(len(buffer))
		}

		_, err = s.Read(buffer[:diff])
	}

	pos = int64(s.position)
	return
}

func (s *prsReader) Size() int64 {
	return s.size
}

func newPrsReader(reader io.ReadSeeker, size int64) *prsReader {
	return &prsReader{reader, 1, 0, [prsBufferSize]uint8{}, [1]uint8{}, 0, 0, size, 0, nil}
}
