package shinglingtexthasher

import (
	"math"
	"testing"
)

func mustNew(t *testing.T, size int) *Hasher {
	t.Helper()
	h, err := New(size)
	if err != nil {
		t.Fatalf("New(%d) returned error: %v", size, err)
	}
	return h
}

func TestNewRejectsNonPositiveSize(t *testing.T) {
	for _, size := range []int{0, -1, -100} {
		if _, err := New(size); err == nil {
			t.Errorf("New(%d) expected error, got nil", size)
		}
	}
}

func TestNewAcceptsPositiveSize(t *testing.T) {
	h := mustNew(t, 3)
	if h.Size() != 3 {
		t.Fatalf("Size() = %d, want 3", h.Size())
	}
}

func TestHashesCountForSimpleSentence(t *testing.T) {
	h := mustNew(t, 3)
	// "the quick brown fox" has 4 words, so 4-3+1 = 2 shingles.
	hashes := h.Hashes("the quick brown fox")
	if len(hashes) != 2 {
		t.Fatalf("len(Hashes) = %d, want 2", len(hashes))
	}
}

func TestHashesAreDeterministic(t *testing.T) {
	h := mustNew(t, 2)
	text := "alpha beta gamma delta"
	first := h.Hashes(text)
	second := h.Hashes(text)
	if len(first) != len(second) {
		t.Fatalf("lengths differ: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("hash %d differs between runs", i)
		}
	}
}

func TestHashesIgnorePunctuationAndCase(t *testing.T) {
	h := mustNew(t, 2)
	a := h.Hashes("Hello, world!")
	b := h.Hashes("hello world")
	if len(a) != len(b) {
		t.Fatalf("lengths differ: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("hash %d differs for punctuation/case variant", i)
		}
	}
}

func TestHashesEmptyWhenFewerWordsThanSize(t *testing.T) {
	h := mustNew(t, 5)
	if got := h.Hashes("only three words"); got != nil {
		t.Fatalf("Hashes = %v, want nil", got)
	}
}

func TestHashesEmptyForEmptyString(t *testing.T) {
	h := mustNew(t, 1)
	if got := h.Hashes(""); got != nil {
		t.Fatalf("Hashes = %v, want nil", got)
	}
}

func TestHashesSizeOneProducesOnePerWord(t *testing.T) {
	h := mustNew(t, 1)
	hashes := h.Hashes("one two three")
	if len(hashes) != 3 {
		t.Fatalf("len(Hashes) = %d, want 3", len(hashes))
	}
}

func TestHashesOrderMatters(t *testing.T) {
	h := mustNew(t, 2)
	a := h.Hashes("alpha beta")
	b := h.Hashes("beta alpha")
	if len(a) != 1 || len(b) != 1 {
		t.Fatalf("unexpected lengths: %d, %d", len(a), len(b))
	}
	if a[0] == b[0] {
		t.Fatalf("word order should change the shingle hash")
	}
}

func TestHashSetDeduplicates(t *testing.T) {
	h := mustNew(t, 2)
	// "a b" appears twice, so the set should have 2 entries, not 3.
	set := h.HashSet("a b a b a")
	if len(set) != 2 {
		t.Fatalf("len(HashSet) = %d, want 2", len(set))
	}
}

func TestJaccardIdenticalSets(t *testing.T) {
	h := mustNew(t, 2)
	set := h.HashSet("alpha beta gamma")
	got := Jaccard(set, set)
	if math.Abs(got-1.0) > 1e-12 {
		t.Fatalf("Jaccard(identical) = %v, want 1", got)
	}
}

func TestJaccardDisjointSets(t *testing.T) {
	h := mustNew(t, 2)
	a := h.HashSet("alpha beta")
	b := h.HashSet("gamma delta")
	got := Jaccard(a, b)
	if math.Abs(got-0.0) > 1e-12 {
		t.Fatalf("Jaccard(disjoint) = %v, want 0", got)
	}
}

func TestJaccardEmptySetsAreSimilar(t *testing.T) {
	empty := map[[32]byte]struct{}{}
	got := Jaccard(empty, empty)
	if math.Abs(got-1.0) > 1e-12 {
		t.Fatalf("Jaccard(empty, empty) = %v, want 1", got)
	}
}

func TestJaccardPartialOverlap(t *testing.T) {
	h := mustNew(t, 1)
	a := h.HashSet("a b c")
	b := h.HashSet("b c d")
	// Intersection {b, c} = 2, union {a, b, c, d} = 4, so 0.5.
	got := Jaccard(a, b)
	if math.Abs(got-0.5) > 1e-12 {
		t.Fatalf("Jaccard = %v, want 0.5", got)
	}
}

func TestHashBytesIsStable(t *testing.T) {
	h := mustNew(t, 1)
	hashes := h.Hashes("hello")
	if len(hashes) != 1 {
		t.Fatalf("len(Hashes) = %d, want 1", len(hashes))
	}
	first := HashBytes(hashes[0])
	second := HashBytes(hashes[0])
	if first != second {
		t.Fatalf("HashBytes is not stable: %v vs %v", first, second)
	}
}

func TestHashUint64MatchesHashBytes(t *testing.T) {
	h := mustNew(t, 1)
	hashes := h.Hashes("hello")
	if len(hashes) != 1 {
		t.Fatalf("len(Hashes) = %d, want 1", len(hashes))
	}
	folded := HashBytes(hashes[0])
	var want uint64
	for i := 7; i >= 0; i-- {
		want = want<<8 | uint64(folded[i])
	}
	got := HashUint64(hashes[0])
	if got != want {
		t.Fatalf("HashUint64 = %d, want %d", got, want)
	}
}

func TestHashUint64DiffersForDifferentText(t *testing.T) {
	h := mustNew(t, 1)
	a := h.Hashes("hello")
	b := h.Hashes("world")
	if len(a) != 1 || len(b) != 1 {
		t.Fatalf("unexpected lengths: %d, %d", len(a), len(b))
	}
	if HashUint64(a[0]) == HashUint64(b[0]) {
		t.Fatalf("different text produced the same folded hash")
	}
}

func TestUnicodeLettersAreWords(t *testing.T) {
	h := mustNew(t, 2)
	// "café" and "über" are single words, so this has one shingle.
	hashes := h.Hashes("café über")
	if len(hashes) != 1 {
		t.Fatalf("len(Hashes) = %d, want 1", len(hashes))
	}
}

func TestDigitsAreWords(t *testing.T) {
	h := mustNew(t, 2)
	hashes := h.Hashes("version 2 release")
	if len(hashes) != 2 {
		t.Fatalf("len(Hashes) = %d, want 2", len(hashes))
	}
}
