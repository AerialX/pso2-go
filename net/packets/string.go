package packets

import (
	"bytes"
	"unicode/utf16"
	"encoding/binary"
)

func EncodeVariableString(v string, xor, sub int) []uint8 {
	data := utf16.Encode([]rune(v))

	var odata bytes.Buffer
	end := binary.LittleEndian
	binary.Write(&odata, end, uint32(xor) ^ (uint32(len(data) + 1) + uint32(sub)))
	binary.Write(&odata, end, data)
	binary.Write(&odata, end, uint16(0))
	return odata.Bytes()
}

func DecodeString(v []uint16) string {
	return string(utf16.Decode(v))
}

func EncodeString(v string, buffer []uint16) {
	raw := utf16.Encode([]rune(v))
	copy(buffer, raw)
	for i := len(raw); i < len(buffer); i++ {
		buffer[i] = 0
	}
}
