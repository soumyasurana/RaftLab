package wal

import "hash/crc32"

// checksum computes the CRC32 checksum of a byte slice.
func checksum(data []byte) uint32 {
	return crc32.ChecksumIEEE(data)
}

// verifyChecksum returns true if the checksum matches.
func verifyChecksum(data []byte, expected uint32) bool {
	return checksum(data) == expected
}
