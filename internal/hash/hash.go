package hash

import (
	"crypto/md5"
	"io"
	"math"
	"os"
)

const (
	// ChunkSize is the size of chunks to read when hashing files
	ChunkSize = 1024 * 1024 // 1 MiB

	// MinPartialSize is the minimum file size to use partial hashing
	MinPartialSize = 3 * ChunkSize // 3 MiB

	// PartialOffset is the offset for partial hashing
	PartialOffset = 0x4000 // 16 KiB

	// PartialSize is the size to read for partial hashing
	PartialSize = 0x4000 // 16 KiB
)

// HashFile calculates the hash of an entire file
func HashFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hasher := md5.New()
	buffer := make([]byte, ChunkSize)

	for {
		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n == 0 {
			break
		}

		hasher.Write(buffer[:n])
	}

	return hasher.Sum(nil), nil
}

// HashFilePartial calculates a partial hash of a file
// It reads data from PartialOffset with size PartialSize
func HashFilePartial(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Seek to the partial offset
	_, err = file.Seek(PartialOffset, 0)
	if err != nil {
		return nil, err
	}

	// Read the partial data
	buffer := make([]byte, PartialSize)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, err
	}

	// Hash the partial data
	hasher := md5.New()
	hasher.Write(buffer[:n])

	return hasher.Sum(nil), nil
}

// HashFileSamples calculates hash from samples at different positions in the file
// This is useful for large files where full hashing would be too slow
func HashFileSamples(path string, fileSize int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hasher := md5.New()
	buffer := make([]byte, ChunkSize/10)

	// Sample at 25%, 50%, and 75% of the file
	samplePositions := []float64{0.25, 0.50, 0.75}

	for _, pos := range samplePositions {
		offset := int64(math.Floor(float64(fileSize) * pos))
		_, err = file.Seek(offset, 0)
		if err != nil {
			return nil, err
		}

		n, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n == 0 {
			continue
		}

		hasher.Write(buffer[:n])
	}

	return hasher.Sum(nil), nil
}
