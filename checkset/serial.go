// serial.go - encoding and decoding of CheckSets over io streams

package checkset

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
)

// bump when protocol breaks compatibility
const ProtocolVersion = 0

// binary representation of a CheckSet entry
// everything is encoded in little-endian

// header occuring once per CheckSet
type CheckSetHeader struct {
	Magic [8]uint8
	Version uint16
}
// header and data occuring once per CheckInfo
type CheckPackHeader struct {
	NameLength uint16
	InfoLength uint16
}
type CheckPackInfo struct {
	Target Platform
	Mode uint16
	Hash [HashSize]uint8
	Future [0]uint8
}
type CheckPack struct {
	Header CheckPackHeader
	Name []uint8
	Info CheckPackInfo
}

var CurrentVersionHeader = CheckSetHeader {
	[8]uint8{'R', 'S', 'P', 'C', 'H', 'E', 'C', 'K'},
	ProtocolVersion,
}

func EncodeCheckPack(path string, info CheckInfo) CheckPack {
	packinfo := CheckPackInfo {
		info.Target,
		uint16(info.Mode),
		[HashSize]uint8(info.Hash),
		[0]byte{},
	}
	header := CheckPackHeader {
		uint16(len(path)),
		uint16(binary.Size(&packinfo)),
	}
	return CheckPack {
		header,
		[]uint8(path),
		packinfo,
	}
}

func DecodeCheckPack(pack *CheckPack) (string, CheckInfo) {
	name := string(pack.Name)
	mode := os.FileMode(pack.Info.Mode)
	return name, CheckInfo {
		pack.Info.Target,
		mode,
		pack.Info.Hash,
	}
}

func ReadCheckPack(stream io.Reader) (CheckPack, error) {
	var pack CheckPack
	err := binary.Read(stream, binary.LittleEndian, &pack.Header)
	if err != nil {
		return pack, err
	}
	name := make([]uint8, pack.Header.NameLength)
	_, err = io.ReadFull(stream, name)
	if err != nil {
		return pack, err
	}
	pack.Name = name
	err = binary.Read(stream, binary.LittleEndian, &pack.Info)
	if err != nil {
		return pack, err
	}
	extra := binary.Size(&pack.Info) - int(pack.Header.InfoLength)
	if extra > 0 {
		ignore := make([]uint8, extra)
		_, err = io.ReadFull(stream, ignore)
	}
	return pack, err
}
func WriteCheckPack(stream io.Writer, pack CheckPack) error {
	err := binary.Write(stream, binary.LittleEndian, pack.Header)
	if err != nil {
		return err
	}
	_, err = stream.Write(pack.Name)
	if err != nil {
		return err
	}
	err = binary.Write(stream, binary.LittleEndian, pack.Info)
	return err
}

var BadMagic = errors.New("Bad magic number for checkset")
var BadVersion = errors.New("Mismatched protocol version")

func Read(stream io.Reader) (CheckSet, error) {
	var err error
	var version CheckSetHeader
	cset := make(CheckSet)
	err = binary.Read(stream, binary.LittleEndian, &version)
	if version.Magic != CurrentVersionHeader.Magic {
		return cset, BadMagic
	} else if version.Version != CurrentVersionHeader.Version {
		return cset, BadVersion
	}
	for {
		var pack CheckPack
		pack, err = ReadCheckPack(stream)
		if err != nil {
			break
		}
		name, info := DecodeCheckPack(&pack)
		cset[name] = info
	}
	if err == io.EOF {
		err = nil
	}
	return cset, err
}

func Write(cset CheckSet, stream io.Writer) error {
	binary.Write(stream, binary.LittleEndian, CurrentVersionHeader)
	for path, info := range cset {
		pack := EncodeCheckPack(path, info)
		err := WriteCheckPack(stream, pack)
		if err != nil {
			return err
		}
	}
	return nil
}
