package snappy

import (
	"bytes"
	"encoding/binary"

	master "github.com/golang/snappy"
	"io"
)

var xerialHeader = []byte{130, 83, 78, 65, 80, 80, 89, 0}

// Encode encodes data as snappy with no framing header.
func Encode(src []byte) []byte {
	return master.Encode(nil, src)
}

// Encode encodes data as snappy with xerial framing format
func EncodeXerialFramingFormat(src []byte) []byte {
	buffer := bytes.NewBuffer(nil)
	writer := NewWriter(buffer)
	writer.Write(src)
	writer.Flush()
	return buffer.Bytes()
}

// Decode decodes snappy data whether it is traditional unframed
// or includes the xerial framing format.
func Decode(src []byte) ([]byte, error) {
	if !bytes.Equal(src[:8], xerialHeader) {
		return master.Decode(nil, src)
	}

	var (
		pos   = uint32(16)
		max   = uint32(len(src))
		dst   = make([]byte, 0, len(src))
		chunk []byte
		err   error
	)
	for pos < max {
		size := binary.BigEndian.Uint32(src[pos : pos+4])
		pos += 4

		chunk, err = master.Decode(chunk, src[pos:pos+size])
		if err != nil {
			return nil, err
		}
		pos += size
		dst = append(dst, chunk...)
	}
	return dst, nil
}

// 'copy' from org.xerial.snappy.SnappyOutputStream
type Writer struct {
	w            io.Writer
	outputSize   int
	blockSize    int
	outputBuffer blockWriter
	inputBuffer  blockWriter
}

var (
	currentHeader = []byte{130, 83, 78, 65, 80, 80, 89, 0, 0, 0, 0, 1, 0, 0, 0, 1}

	HeaderSize = len(currentHeader)

	MinBlockSize = 1 * 1024

	DefaultBlockSize = 32 * 1024
)

func NewWriter(w io.Writer) *Writer {
	return NewWriterWithBlockSize(w, DefaultBlockSize)
}

func NewWriterWithBlockSize(w io.Writer, blockSize int) *Writer {
	if blockSize < MinBlockSize {
		blockSize = MinBlockSize
	}
	writer := &Writer{w: w, blockSize: blockSize}
	writer.inputBuffer.block = make([]byte, blockSize)
	writer.outputBuffer.block = make([]byte, HeaderSize+4+master.MaxEncodedLen(blockSize))

	writer.outputBuffer.Write(currentHeader)
	return writer
}

func (w *Writer) Write(data []byte) (n int, err error) {
	for {
		num, _ := w.inputBuffer.Write(data)
		if num == 0 {
			if err = w.compressInput(); err != nil {
				return
			}
		}
		data = data[num:]
		n += num
		if len(data) == 0 {
			return
		}
	}
}

func (w *Writer) Flush() error {
	err := w.compressInput()
	if err != nil {
		return err
	}
	return w.dumpOutput()
}

func (w *Writer) compressInput() error {
	block := w.inputBuffer.Reset()
	if !w.hasSufficientOutputBufferFor(len(block)) {
		if err := w.dumpOutput(); err != nil {
			return err
		}
	}
	encodedLen := len(master.Encode(w.outputBuffer.block[w.outputBuffer.cursor+4:], block))
	binary.Write(&w.outputBuffer, binary.BigEndian, int32(encodedLen))
	w.outputBuffer.cursor += encodedLen
	return nil
}

func (w *Writer) hasSufficientOutputBufferFor(inputSize int) bool {
	maxCompressedSize := master.MaxEncodedLen(inputSize)
	return maxCompressedSize < len(w.outputBuffer.block)-w.outputBuffer.cursor-4
}

func (w *Writer) dumpOutput() error {
	_, err := w.w.Write(w.outputBuffer.Reset())
	return err
}

type blockWriter struct {
	block  []byte
	cursor int
}

func (b *blockWriter) Write(data []byte) (int, error) {
	writeSize := copy(b.block[b.cursor:], data)
	b.cursor += writeSize
	return writeSize, nil
}

func (b *blockWriter) Reset() []byte {
	tmp := b.block[:b.cursor]
	b.cursor = 0
	return tmp
}
