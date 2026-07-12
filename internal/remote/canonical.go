package remote

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"sort"
)

// CanonicalDigest computes a transport-independent sha256 pin over a validated
// tree. The canonical form is a deterministic serialization — files in lexical
// path order, each emitted as:
//
//	path bytes | 0x00 | exec-bit byte | 8-byte big-endian length | file bytes
//
// so the same content pins identically regardless of archive framing, mtimes,
// or ownership. Only the executable bit of the mode is significant.
func CanonicalDigest(t Tree) Digest {
	files := append([]FileEntry(nil), t.Files...)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })

	h := sha256.New()
	var lenbuf [8]byte
	for _, f := range files {
		h.Write([]byte(f.Path))
		h.Write([]byte{0x00})
		if f.Mode&0o111 != 0 {
			h.Write([]byte{1})
		} else {
			h.Write([]byte{0})
		}
		binary.BigEndian.PutUint64(lenbuf[:], uint64(len(f.Data)))
		h.Write(lenbuf[:])
		h.Write(f.Data)
	}
	return Digest{Algo: "sha256", Hex: hex.EncodeToString(h.Sum(nil))}
}
