package transcoding

import (
	"sort"
	"testing"
)

func TestExtractSegmentNumber(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     int
	}{
		{
			name:     "standard segment",
			filename: "chunk-000.m4s",
			want:     0,
		},
		{
			name:     "segment 1",
			filename: "chunk-001.m4s",
			want:     1,
		},
		{
			name:     "segment 99",
			filename: "chunk-099.m4s",
			want:     99,
		},
		{
			name:     "segment 100",
			filename: "chunk-100.m4s",
			want:     100,
		},
		{
			name:     "segment 999",
			filename: "chunk-999.m4s",
			want:     999,
		},
		{
			name:     "segment 1000 (4 digits)",
			filename: "chunk-1000.m4s",
			want:     1000,
		},
		{
			name:     "segment 2331 (4 digits)",
			filename: "chunk-2331.m4s",
			want:     2331,
		},
		{
			name:     "5-digit segment pattern",
			filename: "chunk-00000.m4s",
			want:     0,
		},
		{
			name:     "5-digit segment 1000",
			filename: "chunk-01000.m4s",
			want:     1000,
		},
		{
			name:     "5-digit segment large",
			filename: "chunk-99999.m4s",
			want:     99999,
		},
		{
			name:     "no dash",
			filename: "chunk.m4s",
			want:     0,
		},
		{
			name:     "no number after dash",
			filename: "chunk-.m4s",
			want:     0,
		},
		{
			name:     "invalid number",
			filename: "chunk-abc.m4s",
			want:     0,
		},
		{
			name:     "wrong extension returns 0",
			filename: "chunk-001.mp4",
			want:     0, // extractSegmentNumber requires .m4s suffix
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSegmentNumber(tt.filename)
			if got != tt.want {
				t.Errorf("extractSegmentNumber(%q) = %d, want %d", tt.filename, got, tt.want)
			}
		})
	}
}

func TestSegmentFileSorting_NumericVsAlphabetical(t *testing.T) {
	// This test demonstrates the bug that was fixed:
	// Alphabetical sorting puts chunk-1000.m4s between chunk-099.m4s and chunk-100.m4s

	// Simulate segment files with >1000 segments (like Dune with 2331 segments)
	segments := []string{
		"chunk-097.m4s",
		"chunk-098.m4s",
		"chunk-099.m4s",
		"chunk-100.m4s",
		"chunk-101.m4s",
		"chunk-999.m4s",
		"chunk-1000.m4s",
		"chunk-1001.m4s",
		"chunk-2330.m4s",
	}

	// Test alphabetical sort (the buggy behavior)
	alphabetical := make([]string, len(segments))
	copy(alphabetical, segments)
	sort.Strings(alphabetical)

	// With alphabetical sort, chunk-1000.m4s comes BEFORE chunk-101.m4s
	// because "1000" < "101" when comparing character by character (0 < 1 at position 3)
	// This means segment 1000 would be processed before segment 101!
	t.Run("alphabetical sort is wrong", func(t *testing.T) {
		// Find positions
		pos1000 := -1
		pos101 := -1
		for i, s := range alphabetical {
			if s == "chunk-1000.m4s" {
				pos1000 = i
			}
			if s == "chunk-101.m4s" {
				pos101 = i
			}
		}

		// With alphabetical sort, chunk-1000 appears before chunk-101 (wrong!)
		if pos1000 >= pos101 {
			t.Errorf("Alphabetical sort should incorrectly place chunk-1000 (pos=%d) before chunk-101 (pos=%d)", pos1000, pos101)
		}
	})

	// Test numeric sort (the correct behavior)
	numeric := make([]string, len(segments))
	copy(numeric, segments)
	sort.Slice(numeric, func(i, j int) bool {
		return extractSegmentNumber(numeric[i]) < extractSegmentNumber(numeric[j])
	})

	t.Run("numeric sort is correct", func(t *testing.T) {
		expected := []string{
			"chunk-097.m4s",
			"chunk-098.m4s",
			"chunk-099.m4s",
			"chunk-100.m4s",
			"chunk-101.m4s",
			"chunk-999.m4s",
			"chunk-1000.m4s",
			"chunk-1001.m4s",
			"chunk-2330.m4s",
		}

		for i, want := range expected {
			if numeric[i] != want {
				t.Errorf("numeric[%d] = %q, want %q", i, numeric[i], want)
			}
		}
	})
}

func TestSegmentFileSorting_MixedPadding(t *testing.T) {
	// Test that sorting works correctly even with mixed padding
	// (e.g., old 3-digit and new 5-digit patterns in the same list)
	segments := []string{
		"chunk-00001.m4s", // 5-digit
		"chunk-002.m4s",   // 3-digit
		"chunk-00000.m4s", // 5-digit
		"chunk-001.m4s",   // 3-digit
		"chunk-000.m4s",   // 3-digit
	}

	sorted := make([]string, len(segments))
	copy(sorted, segments)
	sort.Slice(sorted, func(i, j int) bool {
		return extractSegmentNumber(sorted[i]) < extractSegmentNumber(sorted[j])
	})

	expected := []string{
		"chunk-000.m4s",
		"chunk-00000.m4s", // Both are 0, so order depends on original position (stable-ish)
		"chunk-001.m4s",
		"chunk-00001.m4s", // Both are 1
		"chunk-002.m4s",
	}

	// For segments with the same number, we just verify they're grouped together
	// (exact order within the group doesn't matter for correctness)
	t.Run("segments grouped by number", func(t *testing.T) {
		// Check that segment numbers are in ascending order
		lastNum := -1
		for _, s := range sorted {
			num := extractSegmentNumber(s)
			if num < lastNum {
				t.Errorf("segment %q (num=%d) appears after segment with num=%d", s, num, lastNum)
			}
			lastNum = num
		}
	})

	// Verify both 0-segments and both 1-segments are in the list
	t.Run("all segments present", func(t *testing.T) {
		if len(sorted) != len(expected) {
			t.Errorf("got %d segments, want %d", len(sorted), len(expected))
		}

		for _, want := range expected {
			found := false
			for _, got := range sorted {
				if got == want {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("missing segment %q", want)
			}
		}
	})
}
