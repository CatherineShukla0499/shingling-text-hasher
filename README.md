# shingling-text-hasher

Split text into overlapping word n-grams and hash them for near-duplicate document detection.

```go
package main

import (
	"fmt"

	shinglingtexthasher "example.com/shingling-text-hasher"
)

func main() {
	h, err := shinglingtexthasher.New(5)
	if err != nil {
		panic(err)
	}

	a := h.HashSet("the quick brown fox jumps over the lazy dog")
	b := h.HashSet("the quick brown fox leaps over the lazy dog")

	fmt.Printf("similarity: %.3f\n", shinglingtexthasher.Jaccard(a, b))
}
```

## Why this exists

Near-duplicate detection usually starts with shingling: turn a document into the
set of its overlapping word n-grams, hash each n-gram, and compare the resulting
sets. The trade-off here is that we hash the raw words rather than using a
rolling hash over characters. A rolling hash is faster and needs less memory,
but it is sensitive to tokenization choices and is harder to reason about when
the input mixes scripts. Hashing the joined words costs one allocation per
shingle and gives a result that is trivial to reproduce in another language or
another process.

## API

- `New(size int) (*Hasher, error)` — creates a hasher for shingles of `size`
  words. `size` must be at least 1.
- `(*Hasher) Size() int` — returns the shingle size.
- `(*Hasher) Hashes(text string) [][32]byte` — returns the SHA-256 hash of each
  shingle in document order. Returns `nil` when the text has fewer than `size`
  words.
- `(*Hasher) HashSet(text string) map[[32]byte]struct{}` — returns the distinct
  shingle hashes as a set.
- `Jaccard(a, b map[[32]byte]struct{}) float64` — Jaccard similarity of two hash
  sets. Two empty sets are defined as similarity 1.
- `HashBytes(hash [32]byte) [8]byte` — folds a 32-byte hash into 8 bytes for
  compact storage.
- `HashUint64(hash [32]byte) uint64` — the folded hash as a little-endian
  `uint64`.
- `ErrInvalidShingleSize` — returned by `New` when `size < 1`.
- `DefaultShingleSize` — 5.

## The awkward edge

Tokenization is deliberately simple: a word is a maximal run of Unicode letters
and digits, lowercased. Everything else is a separator. That means `don't`
becomes two words (`don`, `t`) and `foo-bar` becomes two words (`foo`, `bar`).
If your documents contain hyphenated terms that should stay together, you will
need to pre-process them before calling `Hashes` or `HashSet`; the library does
not try to guess.

Another edge: a document with fewer words than the shingle size produces no
hashes at all. That is intentional, because there is no partial shingle that
would be comparable to a full one. Callers that need to compare very short
documents should use a smaller shingle size.
