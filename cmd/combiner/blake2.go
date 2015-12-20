package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/codahale/blake2"
)

func (foot *Footer) FillHashes(cfg *CombinerConfig) error {

	copy(foot.MagicFooterNumber1[:], MAGIC1[:])
	copy(foot.MagicFooterNumber2[:], MAGIC2[:])

	var hash []byte
	var sz int64
	var err error

	hash, sz, err = foot.Blake2HashFile(cfg.ExecutablePath)
	if err != nil {
		return err
	}
	copy(foot.ExecutableBlake2Checksum[:], hash)
	foot.ExecutableLengthBytes = sz

	hash, sz, err = foot.Blake2HashFile(cfg.ZipfilePath)
	if err != nil {
		return err
	}
	copy(foot.ZipfileBlake2Checksum[:], hash)
	foot.ZipfileLengthBytes = sz

	// fill FooterChecksum
	foot.FooterLengthBytes = 256

	ser := foot.ToBytes()

	h := blake2.New(nil)
	h.Write(ser)
	hash = h.Sum(nil)

	copy(foot.ExecutableBlake2Checksum[:], hash)
	return nil
}

func (f *Footer) Blake2HashFile(path string) (hash []byte, length int64, err error) {
	if !FileExists(path) {
		return nil, 0, fmt.Errorf("no such file: '%s'", path)
	}

	of, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("Blake2HashFile() error during opening file '%s': '%s'", path, err)
	}
	defer of.Close()

	h := blake2.New(nil)
	length, err = io.Copy(h, of)
	if err != nil {
		return nil, 0, fmt.Errorf("Blake2HashFile() error during reading from file '%s': '%s'", path, err)
	}
	hash = h.Sum(nil)
	fmt.Printf("hash = '%s' for file '%s'\n", hash, path)
	return hash, length, nil
}

func (f *Footer) ToBytes() []byte {
	// Create a struct and write it.
	buf := &bytes.Buffer{}
	err := binary.Write(buf, binary.BigEndian, f)
	if err != nil {
		panic(err)
	}
	fmt.Println(buf.Bytes())
	return buf.Bytes()
}

func (f *Footer) FromBytes(by []byte) {
	// Read into an empty struct.
	*f = Footer{}
	err := binary.Read(bytes.NewBuffer(by), binary.BigEndian, f)
	if err != nil {
		panic(err)
	}
	fmt.Printf("f = '%#v'\n", *f)
}
