package serialization

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	. "github.com/elastos/Elastos.ELA.SPV/common"
)

var ErrRange = errors.New("value out of range")
var ErrEof = errors.New("got EOF, can not get the next byte")

//Serializable describe the data need be serialized.
type Serializable interface {
	//Write data to writer
	Serialize(w io.Writer) error

	//read data to reader
	Deserialize(r io.Reader) error
}

/*
 ******************************************************************************
 * public func for outside calling
 ******************************************************************************
 * 1. WriteVarUint func, depend on the inpute number's Actual number size,
 *    serialize to bytes.
 *      uint8  =>  (LittleEndian)num in 1 byte                 = 1bytes
 *      uint16 =>  0xfd(1 byte) + (LittleEndian)num in 2 bytes = 3bytes
 *      uint32 =>  0xfe(1 byte) + (LittleEndian)num in 4 bytes = 5bytes
 *      uint64 =>  0xff(1 byte) + (LittleEndian)num in 8 bytes = 9bytes
 * 2. ReadVarUint  func, this func will read the first byte to determined
 *    the num length to read.and retrun the uint64
 *      first byte = 0xfd, read the next 2 bytes as uint16
 *      first byte = 0xfe, read the next 4 bytes as uint32
 *      first byte = 0xff, read the next 8 bytes as uint64
 *      other else,        read this byte as uint8
 * 3. WriteVarBytes func, this func will output two item as serialization.
 *      length of bytes (uint8/uint16/uint32/uint64)  +  bytes
 * 4. WriteVarString func, this func will output two item as serialization.
 *      length of string(uint8/uint16/uint32/uint64)  +  bytes(string)
 * 5. ReadVarBytes func, this func will first read a uint to identify the
 *    length of bytes, and use it to get the next length's bytes to return.
 * 6. ReadVarString func, this func will first read a uint to identify the
 *    length of string, and use it to get the next bytes as a string.
 * 7. GetVarUintSize func, this func will return the length of a uint when it
 *    serialized by the WriteVarUint func.
 * 8. ReadBytes func, this func will read the specify lenth's bytes and retun.
 * 9. ReadUint8,16,32,64 read uint with fixed length
 * 10.WriteUint8,16,32,64 Write uint with fixed length
 * 11.ToArray Serializable to ToArray() func.
 ******************************************************************************
 */

func WriteVarUint(writer io.Writer, value uint64) error {
	var buf [9]byte
	var len = 0
	if value < 0xFD {
		buf[0] = uint8(value)
		len = 1
	} else if value <= 0xFFFF {
		buf[0] = 0xFD
		binary.LittleEndian.PutUint16(buf[1:], uint16(value))
		len = 3
	} else if value <= 0xFFFFFFFF {
		buf[0] = 0xFE
		binary.LittleEndian.PutUint32(buf[1:], uint32(value))
		len = 5
	} else {
		buf[0] = 0xFF
		binary.LittleEndian.PutUint64(buf[1:], uint64(value))
		len = 9
	}
	_, err := writer.Write(buf[:len])
	return err
}

func ReadVarUint(reader io.Reader, maxint uint64) (uint64, error) {
	var res uint64
	if maxint == 0x00 {
		maxint = math.MaxUint64
	}
	var fb [9]byte
	_, err := reader.Read(fb[:1])
	if err != nil {
		return 0, err
	}

	if fb[0] == byte(0xfd) {
		_, err := reader.Read(fb[1:3])
		if err != nil {
			return 0, err
		}
		res = uint64(binary.LittleEndian.Uint16(fb[1:3]))
	} else if fb[0] == byte(0xfe) {
		_, err := reader.Read(fb[1:5])
		if err != nil {
			return 0, err
		}
		res = uint64(binary.LittleEndian.Uint32(fb[1:5]))
	} else if fb[0] == byte(0xff) {
		_, err := reader.Read(fb[1:9])
		if err != nil {
			return 0, err
		}
		res = uint64(binary.LittleEndian.Uint64(fb[1:9]))
	} else {
		res = uint64(fb[0])
	}
	if res > maxint {
		return 0, ErrRange
	}
	return res, nil
}

func WriteVarBytes(writer io.Writer, value []byte) error {
	err := WriteVarUint(writer, uint64(len(value)))
	if err != nil {
		return err
	}
	_, err = writer.Write(value)
	return err
}

func WriteVarString(writer io.Writer, value string) error {
	err := WriteVarUint(writer, uint64(len(value)))
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte(value))
	if err != nil {
		return err
	}
	return nil
}

func ReadVarBytes(reader io.Reader) ([]byte, error) {
	val, err := ReadVarUint(reader, 0)
	if err != nil {
		return nil, err
	}
	str, err := byteXReader(reader, val)
	if err != nil {
		return nil, err
	}
	return str, nil
}

func ReadVarString(reader io.Reader) (string, error) {
	val, err := ReadVarBytes(reader)
	if err != nil {
		return "", err
	}
	return string(val), nil
}

func ReadBytes(reader io.Reader, length uint64) ([]byte, error) {
	str, err := byteXReader(reader, length)
	if err != nil {
		return nil, err
	}
	return str, nil
}

func ReadUint8(reader io.Reader) (uint8, error) {
	var p [1]byte
	n, err := reader.Read(p[:])
	if n <= 0 || err != nil {
		return 0, ErrEof
	}
	return uint8(p[0]), nil
}

func ReadUint16(reader io.Reader) (uint16, error) {
	var p [2]byte
	n, err := reader.Read(p[:])
	if n <= 0 || err != nil {
		return 0, ErrEof
	}
	return binary.LittleEndian.Uint16(p[:]), nil
}

func ReadUint32(reader io.Reader) (uint32, error) {
	var p [4]byte
	n, err := reader.Read(p[:])
	if n <= 0 || err != nil {
		return 0, ErrEof
	}
	return binary.LittleEndian.Uint32(p[:]), nil
}

func ReadUint64(reader io.Reader) (uint64, error) {
	var p [8]byte
	n, err := reader.Read(p[:])
	if n <= 0 || err != nil {
		return 0, ErrEof
	}
	return binary.LittleEndian.Uint64(p[:]), nil
}

func WriteUint8(writer io.Writer, val uint8) error {
	var p [1]byte
	p[0] = byte(val)
	_, err := writer.Write(p[:])
	return err
}

func WriteUint16(writer io.Writer, val uint16) error {
	var p [2]byte
	binary.LittleEndian.PutUint16(p[:], val)
	_, err := writer.Write(p[:])
	return err
}

func WriteUint32(writer io.Writer, val uint32) error {
	var p [4]byte
	binary.LittleEndian.PutUint32(p[:], val)
	_, err := writer.Write(p[:])
	return err
}

func WriteUint64(writer io.Writer, val uint64) error {
	var p [8]byte
	binary.LittleEndian.PutUint64(p[:], val)
	_, err := writer.Write(p[:])
	return err
}

//**************************************************************************
//**    internal func                                                    ***
//**************************************************************************
//** 2.byteXReader: read x byte and return []byte.
//** 3.byteToUint8: change byte -> uint8 and return.
//**************************************************************************

func byteXReader(reader io.Reader, x uint64) ([]byte, error) {
	p := make([]byte, x)
	n, err := reader.Read(p)
	if n > 0 {
		return p[:], nil
	}
	return p, err
}

func WriteElements(writer io.Writer, elements ...interface{}) error {
	for _, e := range elements {
		err := WriteElement(writer, e)
		if err != nil {
			return err
		}
	}
	return nil
}

func WriteElement(writer io.Writer, element interface{}) (err error) {
	switch e := element.(type) {
	case Uint256:
		err = e.Serialize(writer)
	case []Uint256:
		for _, v := range e {
			err = WriteElement(writer, v)
			if err != nil {
				return err
			}
		}
	case []*Uint256:
		for _, v := range e {
			err = WriteElement(writer, *v)
			if err != nil {
				return err
			}
		}
	case []byte:
		err = WriteVarBytes(writer, e)
	default:
		err = binary.Write(writer, binary.LittleEndian, e)
	}
	return err
}

func ReadElements(reader io.Reader, elements ...interface{}) error {
	for _, e := range elements {
		err := ReadElement(reader, e)
		if err != nil {
			return err
		}
	}
	return nil
}

func ReadElement(reader io.Reader, element interface{}) (err error) {
	switch e := element.(type) {
	case *Uint256:
		err = (*e).Deserialize(reader)
	case *[]Uint256:
		for i, v := range *e {
			err = v.Deserialize(reader)
			if err != nil {
				return err
			}
			(*e)[i] = v
		}
	case *[]*Uint256:
		for i, v := range *e {
			v = new(Uint256)
			err = (*v).Deserialize(reader)
			if err != nil {
				return err
			}
			(*e)[i] = v
		}
	case *[]byte:
		*e, err = ReadVarBytes(reader)
	default:
		err = binary.Read(reader, binary.LittleEndian, e)
	}
	return err
}
