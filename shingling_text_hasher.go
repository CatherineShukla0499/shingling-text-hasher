// Package shinglingtexthasher splits text into overlapping word n-grams
// (shingles) and hashes them for near-duplicate document detection.
//
// The package deliberately uses only the standard library so it can be
// vendored into constrained build environments without a network fetch.
package shinglingtexthasher

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"strings"
	"unicode"
)

// DefaultShingleSize is the number of words in each shingle when the caller
// does not specify a size. Five is a common choice for near-duplicate
// detection: long enough to be specific, short enough to survive minor edits.
const DefaultShingleSize = 5

// ErrInvalidShingleSize is returned by New when size is less than 1.
var ErrInvalidShingleSize = errors.New("shinglingtexthasher: shingle size must be at least 1")

// Hasher turns text into a set of shingle hashes.
//
// The zero value is not usable; call New.
type Hasher struct {
	size int
}

// New returns a Hasher that produces shingles of the given number of words.
//
// size must be at least 1. A size of 1 hashes individual words, which is
// useful for bag-of-words comparisons but loses all local ordering.
func New(size int) (*Hasher, error) {
	if size < 1 {
		return nil, ErrInvalidShingleSize
	}
	return &Hasher{size: size}, nil
}

// Size reports the number of words per shingle.
func (h *Hasher) Size() int {
	return h.size
}

// Hashes splits text into word shingles and returns the SHA-256 hash of each
// shingle as a 32-byte array.
//
// Words are maximal runs of Unicode letters and digits. Everything else is a
// separator, so punctuation and whitespace do not affect the result. A
// document with fewer words than the shingle size yields no hashes: there is
// no sensible partial shingle to compare against.
//
// The returned slice is in document order and may contain duplicates if the
// same shingle occurs more than once. Callers that want a set should dedupe
// the result themselves.
func (h *Hasher) Hashes(text string) [][32]byte {
	words := tokenize(text)
	if len(words) < h.size {
		return nil
	}

	out := make([][32]byte, 0, len(words)-h.size+1)
	for i := 0; i+h.size <= len(words); i++ {
		// Hash the words joined by a single space. The join is unambiguous
		// because tokenize never produces empty words, so the space can
		// only be a separator.
		joined := strings.Join(words[i:i+h.size], " ")
		out = append(out, sha256.Sum256([]byte(joined)))
	}
	return out
}

// HashSet returns the distinct shingle hashes of text as a map keyed by the
// 32-byte hash. This is the representation most useful for Jaccard-style
// near-duplicate comparison.
func (h *Hasher) HashSet(text string) map[[32]byte]struct{} {
	hashes := h.Hashes(text)
	set := make(map[[32]byte]struct{}, len(hashes))
	for _, hash := range hashes {
		set[hash] = struct{}{}
	}
	return set
}

// Jaccard returns the Jaccard similarity of two hash sets in [0, 1].
//
// The Jaccard index is |A ∩ B| / |A ∪ B|. Two empty sets are defined to have
// similarity 1, matching the convention that identical documents are
// maximally similar. The result is exact: no floating-point arithmetic is
// involved beyond the final division, and the integer counts are small enough
// that the quotient is deterministic for a given input.
func Jaccard(a, b map[[32]byte]struct{}) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1
	}

	// Iterate over the smaller set to keep the loop cheap.
	if len(a) > len(b) {
		a, b = b, a
	}

	intersection := 0
	for hash := range a {
		if _, ok := b[hash]; ok {
			intersection++
		}
	}
	union := len(a) + len(b) - intersection
	return float64(intersection) / float64(union)
}

// HashBytes returns a stable 8-byte little-endian representation of a shingle
// hash. It is handy when a caller needs to store hashes in a fixed-width
// column or compare them as integers.
func HashBytes(hash [32]byte) [8]byte {
	// Fold the 32-byte hash down to 8 bytes by XORing each 8-byte chunk.
	// This keeps all input bytes significant while producing a compact key.
	var out [8]byte
	for i := 0; i < 4; i++ {
		chunk := hash[i*8 : i*8+8]
		for j := 0; j < 8; j++ {
			out[j] ^= chunk[j]
		}
	}
	return out
}

// HashUint64 is a convenience wrapper around HashBytes that returns the folded
// hash as a uint64 in little-endian order.
func HashUint64(hash [32]byte) uint64 {
	folded := HashBytes(hash)
	return binary.LittleEndian.Uint64(folded[:])
}

// tokenize splits text into maximal runs of Unicode letters and digits.
//
// The rule is intentionally simple: a letter or digit continues a word, and
// anything else ends it. This treats "don't" as two words ("don", "t") and
// "foo-bar" as two words ("foo", "bar"), which is the behaviour we want for
// near-duplicate detection because punctuation differences should not create
// different shingles.
func tokenize(text string) []string {
	var words []string
	var current []rune

	flush := func() {
		if len(current) > 0 {
			words = append(words, string(current))
			current = current[:0]
		}
	}

	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current = append(current, unicode.ToLower(r))
		} else {
			flush()
		}
	}
	flush()
	return words
}
