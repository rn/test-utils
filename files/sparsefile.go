package main

// This program
// - Creates a large sparse file with ftruncate()
// - Writes "blocks" of checksummed data either sequentially or random (default)
// - Scans the entire sparse file for non-zero "blocks" and verifies the checksum
//
// To run:
// go run sparsefile.go <arguments>
//
// There are three modes:
// - rand:   Use random offsets to write each block
// - seq:    Sequentially write blocks from start of file
// - stream: Write random length "streams" (multiple blocks) to random offsets

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"hash/adler32"
	"log"
	"math/rand"
	"os"
	"syscall"
)

func main() {
	argBlockSz := flag.Int64("blocksize", 4096, "Size of 'blocks' to write")
	argNumBlocks := flag.Int("blocks", 1000, "Number of blocks to write")
	syncBlocks := flag.Int("sync", 0, "Call sync() every n blocks")
	mode := flag.String("mode", "rand", "Mode: rand, seq, stream")
	minBlocks := flag.Int("stream-min", 5, "In stream mode, minimum number of blocks")
	maxBlocks := flag.Int("stream-max", 30, "In stream mode, maximum number of blocks")
	fileName := flag.String("file", "disk.img", "File name to use for sparse file")
	argFileSz := flag.Int64("size", 30*1024*1024*1024, "File size")
	seed := flag.Int64("seed", 42, "Seed for the random number generator")
	verbose := flag.Bool("v", false, "Enable verbose output (loads)")
	flag.Parse()

	// Having to do pointer de-reference is tedious
	blockSz := *argBlockSz
	numBlocks := *argNumBlocks
	fileSz := *argFileSz
	totalBlocks := fileSz / blockSz

	if blockSz >= fileSz {
		log.Fatal("Blocksize should be smaller than filesize")
	}
	if *minBlocks >= *maxBlocks {
		log.Fatal("stream-min should be less than stream-max")
	}

	if *mode == "seq" {
		if int64(numBlocks) > totalBlocks {
			log.Fatal("Filesize too small")
		}
	}

	r := rand.New(rand.NewSource(*seed))

	fmt.Println("Create file:", *fileName)
	f, err := os.Create(*fileName)
	if err != nil {
		log.Fatal("Open file:", err)
	}
	defer f.Close()

	fd := f.Fd()
	fmt.Println("Ftruncate file")
	if err := syscall.Ftruncate(int(fd), fileSz); err != nil {
		log.Fatal("Ftruncate:", err)
	}

	fmt.Println("Write data")
	var off int64
	var streamBlocks int64
	for i := 0; i < numBlocks; i++ {
		b := fillBuf(int(blockSz))

		switch *mode {
		case "rand":
			off = r.Int63n(totalBlocks) * blockSz
		case "seq":
			off = int64(i) * blockSz
		case "stream":
			if streamBlocks == 0 {
				// new stream
				streamBlocks = int64(*minBlocks)
				streamBlocks += rand.Int63n(int64(*maxBlocks - *minBlocks + 1))
				var offBlock int64
				for {
					offBlock = r.Int63n(totalBlocks)
					if offBlock+streamBlocks <= totalBlocks {
						break
					}
				}
				off = offBlock * blockSz
			} else {
				off += blockSz
			}
			streamBlocks--
		}
		if *verbose {
			fmt.Printf("w: %08x %08x\n", off, len(b))
		}
		_, err := f.WriteAt(b, off)
		if err != nil {
			log.Fatal("write failed:", err)
		}

		if *syncBlocks != 0 && i%*syncBlocks == 0 {
			if err := f.Sync(); err != nil {
				log.Fatal("write failed:", err)
			}
		}
	}

	fmt.Println("Verify the file")
	b := make([]byte, blockSz)
	empty := make([]byte, blockSz)
	count := 0
	for i := int64(0); i < totalBlocks; i++ {
		off := i * blockSz
		if *verbose {
			fmt.Printf("r: %08x %08x\n", off, len(b))
		}
		_, err = f.ReadAt(b, off)
		if err != nil {
			log.Printf("read at offset %d failed with %v", off, err)
			continue
		}
		// If it is all zeroes we didn't write to it. Skip
		if bytes.Equal(b, empty) {
			continue
		}
		count++
		if *verbose {
			fmt.Printf("v: %08x %08x\n", off, len(b))
		}
		if !verifyBuf(b) {
			fmt.Printf("\nXXX: Block %d did not verify\n", i)
			printBuf(b)
		}
	}
	fmt.Printf("Verified %d blocks\n", count)
	if count > numBlocks {
		log.Fatalf("Verified more non-zero than blocks we wrote %d > %d\n", count, numBlocks)
	}
}

// fillBuf creates a buffer of size n filled with random data and a 4B checksum in the last word
func fillBuf(n int) []byte {
	if n < 8 {
		log.Fatal("Buffer needs at least 8B")
	}
	if n%4 != 0 {
		log.Fatal("Buffer needs to be multiple of 4B")
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(rand.Intn(255))
	}

	// Calculate checksum and add it
	csum := adler32.Checksum(b[:len(b)-4])
	a := make([]byte, 4)
	binary.LittleEndian.PutUint32(a, csum)
	for i := range a {
		b[len(b)-4+i] = a[i]
	}
	return b
}

// verifyBuf verifies the content of the buffer by checking the included checksum
func verifyBuf(b []byte) bool {
	csum1 := adler32.Checksum(b[:len(b)-4])
	csum2 := binary.LittleEndian.Uint32(b[len(b)-4:])
	return csum1 == csum2
}

// printBuf prints the buffer in hexdump kinda way
func printBuf(b []byte) {
	for i := 0; i < len(b); i += 16 {
		hex := ""
		str := ""
		for j := 0; j < 16; j++ {
			if b[i+j] < 32 || b[i+j] >= 127 {
				str += " "
			} else {
				str += string(b[i+j])
			}
			hex += fmt.Sprintf("%02x ", b[i+j])
		}
		fmt.Printf("%06x: %s %s\n", i, hex, str)
	}
}
