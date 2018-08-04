// MIT License
//
// Copyright (c) 2018 dualface
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package impl

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/dualface/go-gbc/gbc"
	"strings"
)

const (
	RawMessageMaxLen      = 64 * 1024            // 64KB, 0x010000
	RawMessageMaxLenMask  = RawMessageMaxLen - 1 // 0x00ffff
	RawMessageHeaderLen   = 14                   // (chunkSize uint32, mainCmdId uint16, subCmdId uint16, dataSize uint32, DataType uint16)
	RawMessagePaddingSize = 8                    // 8 bytes
)

type (
	RawMessageImpl struct {
		chunkSize uint32 // length of chunk (exclude chunkSize)
		mainCmdId uint16
		subCmdId  uint16
		dataSize  uint32 // length of valid bytes in data
		dataType  uint16 // Protobuf or Clang struct
		data      []byte
		remains   int
		offset    int
	}
)

func NewRawMessageFromHeaderBuf(buf []byte) (*RawMessageImpl, error) {
	l := len(buf)
	if l < RawMessageHeaderLen {
		return nil, fmt.Errorf("buf is not enough")
	}

	chunkSize := binary.LittleEndian.Uint32(buf[0:4]) & RawMessageMaxLenMask
	mainCmdId := binary.LittleEndian.Uint16(buf[4:6])
	subCmdId := binary.LittleEndian.Uint16(buf[6:8])
	dataSize := binary.LittleEndian.Uint32(buf[8:12])
	dataType := binary.LittleEndian.Uint16(buf[12:14])

	remains := chunkSize - RawMessageHeaderLen + 4 // chunk size not include first 4bytes
	if remains < dataSize {
		return nil, fmt.Errorf("invalid chunk size or data size")
	}

	c := &RawMessageImpl{
		chunkSize: chunkSize,
		mainCmdId: mainCmdId,
		subCmdId:  subCmdId,
		dataSize:  dataSize,
		dataType:  dataType,
		data:      make([]byte, remains),
		remains:   int(remains),
	}
	return c, nil
}

func NewRawMessageFromData(mainCmdId uint16, subCmdId uint16, dataType uint16, data []byte) *RawMessageImpl {
	c := &RawMessageImpl{}
	c.mainCmdId = mainCmdId
	c.subCmdId = subCmdId
	c.dataType = dataType
	c.dataSize = uint32(len(data))

	chunkSize := c.dataSize + RawMessageHeaderLen - 4 // padding not include first 4bytes
	m := chunkSize % RawMessagePaddingSize
	if m > 0 {
		chunkSize = (chunkSize - m) + RawMessagePaddingSize
	}
	alignDataSize := chunkSize - RawMessageHeaderLen + 4
	c.data = make([]byte, alignDataSize)
	copy(c.data[0:], data)

	c.chunkSize = chunkSize
	return c
}

func (m *RawMessageImpl) WriteBytes(b []byte) (int, error) {
	l := len(b)
	if l > m.remains {
		return 0, fmt.Errorf("write failed, remains bytes is %d, try to write %d", m.remains, l)
	}

	copy(m.data[m.offset:], b)
	m.offset += l
	m.remains -= l
	return l, nil
}

func (m *RawMessageImpl) RemainsBytes() int {
	return m.remains
}

func (m *RawMessageImpl) GenBytes() []byte {
	var buf bytes.Buffer

	binary.Write(&buf, binary.LittleEndian, m.chunkSize)
	binary.Write(&buf, binary.LittleEndian, m.mainCmdId)
	binary.Write(&buf, binary.LittleEndian, m.subCmdId)
	binary.Write(&buf, binary.LittleEndian, m.dataSize)
	binary.Write(&buf, binary.LittleEndian, m.DataType)
	buf.Write(m.data)

	return buf.Bytes()
}

// interface RawMessageImpl

func (m *RawMessageImpl) DataType() gbc.MessageType {
	return m.dataType
}

func (m *RawMessageImpl) DataBytes() []byte {
	return m.data[:m.dataSize]
}

func (m *RawMessageImpl) String() string {
	sb := &strings.Builder{}
	fmt.Fprintf(sb, "chunk:%d,", m.chunkSize)
	fmt.Fprintf(sb, "main:%d,", m.mainCmdId)
	fmt.Fprintf(sb, "sub:%d,", m.subCmdId)
	fmt.Fprintf(sb, "size:%d,", m.dataSize)
	fmt.Fprintf(sb, "type:%d [", m.dataType)

	l := int(m.dataSize)
	for i := 0; i < l; i++ {
		fmt.Fprintf(sb, "%02X", m.data[i])
		if i < l-1 {
			sb.WriteByte(' ')
		}
	}

	sb.WriteByte(']')
	return sb.String()
}
