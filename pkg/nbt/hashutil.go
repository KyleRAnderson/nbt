package nbt

import (
	"encoding/binary"
	"fmt"
	"hash"
)

func HashWriteNum[X int | int32 | int64 | uint | uint32 | uint64 | float32 | float64](h hash.Hash64, num X) {
	if err := binary.Write(h, binary.LittleEndian, num); err != nil {
		/* The write to the hash should not fail, if it does something's wrong. */
		panic(fmt.Errorf(`nbt.HashWriteNum: failed to write number: %w`, err))
	}
}
