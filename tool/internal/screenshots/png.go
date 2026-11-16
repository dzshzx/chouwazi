package screenshots

import (
	"bytes"
	"compress/zlib"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"hash/crc32"
	"image"
	"image/draw"
	"image/png"
	"io"
	"os"
)

var pngSignature = []byte("\x89PNG\r\n\x1a\n")

// colorChunks are the ancillary chunks that define how the pixels look; they are
// carried over unchanged. Other ancillary chunks (text, resolution) are dropped.
var colorChunks = map[string]bool{"iCCP": true, "sRGB": true, "gAMA": true, "cHRM": true}

type chunk struct {
	kind string
	data []byte
}

func readChunks(data []byte) ([]chunk, error) {
	if !bytes.HasPrefix(data, pngSignature) {
		return nil, errors.New("not a PNG")
	}
	var chunks []chunk
	for rest := data[len(pngSignature):]; len(rest) > 0; {
		if len(rest) < 12 {
			return nil, errors.New("truncated PNG chunk")
		}
		length := binary.BigEndian.Uint32(rest)
		if uint64(length)+12 > uint64(len(rest)) {
			return nil, errors.New("truncated PNG chunk")
		}
		chunks = append(chunks, chunk{string(rest[4:8]), rest[8 : 8+length]})
		rest = rest[12+length:]
	}
	if len(chunks) == 0 || chunks[0].kind != "IHDR" || len(chunks[0].data) != 13 {
		return nil, errors.New("PNG has no header")
	}
	return chunks, nil
}

type header struct {
	width, height int
	depth, color  byte
}

func readHeader(chunks []chunk) header {
	data := chunks[0].data
	return header{int(binary.BigEndian.Uint32(data)), int(binary.BigEndian.Uint32(data[4:])), data[8], data[9]}
}

// iccProfile returns the decompressed embedded color profile, or nil when there is none.
func iccProfile(chunks []chunk) ([]byte, error) {
	for _, item := range chunks {
		if item.kind != "iCCP" {
			continue
		}
		name := bytes.IndexByte(item.data, 0)
		if name < 0 || name+2 > len(item.data) {
			return nil, errors.New("invalid iCCP chunk")
		}
		reader, err := zlib.NewReader(bytes.NewReader(item.data[name+2:]))
		if err != nil {
			return nil, err
		}
		return io.ReadAll(reader)
	}
	return nil, nil
}

func writeChunk(out *bytes.Buffer, item chunk) {
	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(item.data)))
	out.Write(length[:])
	checksum := crc32.NewIEEE()
	checksum.Write([]byte(item.kind))
	checksum.Write(item.data)
	out.WriteString(item.kind)
	out.Write(item.data)
	var crc [4]byte
	binary.BigEndian.PutUint32(crc[:], checksum.Sum32())
	out.Write(crc[:])
}

// fullyOpaque reports whether every pixel's 8-bit alpha is 255.
func fullyOpaque(source image.Image) bool {
	bounds := source.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if _, _, _, alpha := source.At(x, y).RGBA(); alpha>>8 != 0xff {
				return false
			}
		}
	}
	return true
}

func checkSource(raw []byte, width, height int) ([]chunk, image.Image, error) {
	chunks, err := readChunks(raw)
	if err != nil {
		return nil, nil, Error("Screenshot has an unexpected format or pixel size")
	}
	head := readHeader(chunks)
	if err := require(head.width == width && head.height == height, "Screenshot has an unexpected format or pixel size"); err != nil {
		return nil, nil, err
	}
	if err := require((head.color == 2 || head.color == 6) && (head.depth == 8 || head.depth == 16),
		"Screenshot has an unexpected pixel format"); err != nil {
		return nil, nil, err
	}
	source, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, nil, err
	}
	if head.color == 6 {
		if err := require(fullyOpaque(source), "Native screenshot has transparent pixels; refusing to change its appearance"); err != nil {
			return nil, nil, err
		}
	}
	return chunks, source, nil
}

// OpaquePNG removes only a redundant, fully opaque alpha channel, without compositing.
// PNG encoding is lossless: 8-bit RGB values, the entire screen and the color
// chunks (iCCP, sRGB, gAMA, cHRM) stay intact. It returns the SHA-256 of the new file.
func OpaquePNG(path string, width, height int) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	chunks, source, err := checkSource(raw, width, height)
	if err != nil {
		return "", err
	}
	profile, err := iccProfile(chunks)
	if err != nil {
		return "", err
	}
	rgb := image.NewRGBA(source.Bounds())
	draw.Draw(rgb, rgb.Bounds(), source, source.Bounds().Min, draw.Src)
	var encoded bytes.Buffer
	// An opaque RGBA image is written as 8-bit truecolor without alpha.
	if err := png.Encode(&encoded, rgb); err != nil {
		return "", err
	}
	written, err := readChunks(encoded.Bytes())
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	out.Write(pngSignature)
	writeChunk(&out, written[0])
	for _, item := range chunks {
		if colorChunks[item.kind] {
			writeChunk(&out, item)
		}
	}
	for _, item := range written[1:] {
		writeChunk(&out, item)
	}
	if err := os.WriteFile(path, out.Bytes(), 0o644); err != nil {
		return "", err
	}
	return verifySaved(path, width, height, profile)
}

func verifySaved(path string, width, height int, profile []byte) (string, error) {
	saved, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	chunks, err := readChunks(saved)
	if err != nil {
		return "", err
	}
	head := readHeader(chunks)
	if err := require(head.color == 2 && head.depth == 8 && head.width == width && head.height == height,
		"Saved PNG must be RGB without alpha"); err != nil {
		return "", err
	}
	savedProfile, err := iccProfile(chunks)
	if err != nil {
		return "", err
	}
	if err := require(bytes.Equal(savedProfile, profile) && (savedProfile == nil) == (profile == nil),
		"Saved PNG must retain its native color profile"); err != nil {
		return "", err
	}
	sum := sha256.Sum256(saved)
	return hex.EncodeToString(sum[:]), nil
}
