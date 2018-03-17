package msg

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"

	"SPVWallet/config"
	"SPVWallet/core"
	"SPVWallet/log"
)

const (
	CMDLEN      = 12
	CMDOFFSET   = 4
	CHECKSUMLEN = 4
	HEADERLEN   = 24
)

var (
	EmptyMsgSum = []byte{0x5d, 0xf6, 0xe0, 0xe2}
)

type Header struct {
	Magic    uint32
	CMD      [CMDLEN]byte
	Length   uint32
	Checksum [CHECKSUMLEN]byte
}

func NewHeader(cmd string, checksum []byte, length int) *Header {
	header := new(Header)
	// Write Magic
	header.Magic = config.Config().Magic
	// Write CMD
	copy(header.CMD[:len(cmd)], cmd)
	// Write length
	header.Length = uint32(length)
	// Write checksum
	copy(header.Checksum[:], checksum[:CHECKSUMLEN])

	return header
}

func BuildHeader(cmd string, msg []byte) *Header {
	// Calculate checksum
	checksum := core.Sha256D(msg)
	return NewHeader(cmd, checksum[:], len(msg))
}

func BuildMessage(cmd string, body []byte) ([]byte, error) {
	hdr, err := BuildHeader(cmd, body).Serialize()
	if err != nil {
		return nil, err
	}

	log.Info("Build message:", cmd)
	return append(hdr, body...), nil
}

func (header *Header) Verify(buf []byte) error {
	// Verify magic
	if header.Magic != config.Config().Magic {
		return errors.New(fmt.Sprint("Unmatched magic number ", header.Magic))
	}

	sum := core.Sha256D(buf)
	checksum := sum[:CHECKSUMLEN]
	if !bytes.Equal(header.Checksum[:], checksum) {
		return errors.New(
			fmt.Sprintf("Unmatched checksum, expecting %s get $s",
				hex.EncodeToString(checksum),
				hex.EncodeToString(header.Checksum[:])))
	}

	return nil
}

func (header *Header) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, header)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (header *Header) Deserialize(buf []byte) error {
	// Check CMD
	cmd := buf[CMDOFFSET:CMDOFFSET+CMDLEN]
	end := bytes.IndexByte(cmd, 0)
	if end < 0 || end >= CMDLEN {
		return errors.New("Unexpected length of CMD")
	}

	hdr := bytes.NewBuffer(buf[:HEADERLEN])
	return binary.Read(hdr, binary.LittleEndian, header)
}

func (header *Header) GetCMD() string {
	end := bytes.IndexByte(header.CMD[:], 0)
	return string(header.CMD[:end])
}