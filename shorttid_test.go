package stid

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

// Test the DefaultConfig values
func Test_DefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	expectedEpoch := time.Unix(0, 0).UTC()

	if !cfg.epoch.Equal(expectedEpoch) {
		t.Errorf("DefaultConfig epoch got %v, want %v", cfg.epoch, expectedEpoch)
	}
	if cfg.timeGranularity != Millisecond {
		t.Errorf("DefaultConfig timeGranularity got %d, want %d (Millisecond)", cfg.timeGranularity, Millisecond)
	}
	if cfg.alphabet != DefaultAlphabet {
		t.Errorf("DefaultConfig alphabet got %q, want %q", cfg.alphabet, DefaultAlphabet)
	}
	if cfg.randomChars != 5 {
		t.Errorf("DefaultConfig randomChars got %d, want 5", cfg.randomChars)
	}
}

// Test NewGenerator validation logic
func Test_NewGenerator_Validation(t *testing.T) {
	testCases := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{"Valid Default", DefaultConfig(), false},
		{
			name: "Valid Custom",
			config: Config{
				epoch:           time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
				timeGranularity: Second,
				alphabet:        "abc",
				randomChars:     3,
			},
			expectError: false,
		},
		{
			name: "Invalid alphabet Empty",
			config: Config{
				epoch:           DefaultEpoch,
				timeGranularity: Millisecond,
				alphabet:        "", // Invalid
				randomChars:     5,
			},
			expectError: true,
		},
		{
			name: "Invalid alphabet Single Char",
			config: Config{
				epoch:           DefaultEpoch,
				timeGranularity: Millisecond,
				alphabet:        "a", // Invalid
				randomChars:     5,
			},
			expectError: true,
		},
		{
			name: "Invalid randomChars Negative",
			config: Config{
				epoch:           DefaultEpoch,
				timeGranularity: Millisecond,
				alphabet:        DefaultAlphabet,
				randomChars:     -1, // Invalid
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gen, err := NewGenerator(tc.config)
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
				if gen == nil {
					t.Errorf("Expected a generator instance, but got nil")
				}
				// Check if nil location epoch was defaulted to UTC
				if tc.config.epoch.Location() == nil && gen != nil {
					if gen.config.epoch.Location() != time.UTC {
						t.Errorf("epoch location was nil, expected generator to default to UTC, but got %v", gen.config.epoch.Location())
					}
				}
			}
		})
	}
}

// Test Generate basic functionality
func Test_Generate(t *testing.T) {
	id1, err1 := Generate()
	id2, err2 := Generate()

	if err1 != nil {
		t.Fatalf("Generate() #1 failed: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("Generate() #2 failed: %v", err2)
	}

	if id1 == "" {
		t.Error("Generate() #1 produced empty ID")
	}
	if id2 == "" {
		t.Error("Generate() #2 produced empty ID")
	}
	if id1 == id2 {
		t.Error("Generate() produced identical IDs sequentially (highly unlikely)")
	}

	// Check if characters are from the default Base62 alphabet
	if !containsOnly(id1, DefaultAlphabet) {
		t.Errorf("Default ID %q contains characters outside the default alphabet %q", id1, DefaultAlphabet)
	}

	// Check approximate length based on defaults (will grow over time)
	// Expecting ~8 chars timestamp + 5 random = ~13 chars initially from 1970
	// This is a loose check as timestamp part length grows.
	if len(id1) < 10 || len(id1) > 20 {
		t.Logf("Default ID length is %d. This is expected to grow over time.", len(id1))
	}

}

// Test basic generation, uniqueness, and character set
func Test_Generator_Generate_Basic(t *testing.T) {
	cfg := Config{
		epoch:           time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		timeGranularity: Decisecond,
		alphabet:        "abcdef0123456789", // Hex Lowercase
		randomChars:     6,
	}
	gen, err := NewGenerator(cfg)
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

	const numIDs = 100
	generatedIDs := make(map[string]struct{}, numIDs)
	expectedLen := -1

	for i := 0; i < numIDs; i++ {
		id, err := gen.Generate()
		if err != nil {
			t.Fatalf("Generate() failed on iteration %d: %v", i, err)
		}
		if id == "" {
			t.Fatalf("Generate() produced empty ID on iteration %d", i)
		}

		// Check character set
		if !containsOnly(id, cfg.alphabet) {
			t.Errorf("ID %q contains characters outside the specified alphabet %q", id, cfg.alphabet)
		}

		// Check for uniqueness
		if _, exists := generatedIDs[id]; exists {
			t.Fatalf("Duplicate ID generated: %q", id)
		}
		generatedIDs[id] = struct{}{}

		// Check length consistency (can vary slightly if timestamp part grows)
		if expectedLen == -1 {
			expectedLen = len(id)
		} else if len(id) < expectedLen || len(id) > expectedLen+1 { // Allow length to increase by 1
			t.Errorf("ID length changed unexpectedly: got %d, expected around %d", len(id), expectedLen)
			// Update expected length if it grows legitimately
			if len(id) > expectedLen {
				expectedLen = len(id)
			}
		}
	}

	if len(generatedIDs) != numIDs {
		t.Errorf("Expected %d unique IDs, but generated %d", numIDs, len(generatedIDs))
	}
}

// Test sortability of generated IDs
func Test_Generator_Generate_Sortability(t *testing.T) {
	cfg := Config{
		epoch:           time.Now().UTC().Add(-10 * time.Second), // Start epoch recently
		timeGranularity: Decisecond,                              // 100ms
		alphabet:        DefaultAlphabet,
		randomChars:     4,
	}
	gen, err := NewGenerator(cfg)
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

	const numIDs = 5
	const delay = 150 * time.Millisecond // Delay > timeGranularity

	originalOrder := make([]string, numIDs)
	sortedOrder := make([]string, numIDs)

	for i := 0; i < numIDs; i++ {
		id, err := gen.Generate()
		if err != nil {
			t.Fatalf("Generate() failed on iteration %d: %v", i, err)
		}
		originalOrder[i] = id
		sortedOrder[i] = id
		if i < numIDs-1 {
			time.Sleep(delay) // Wait longer than granularity
		}
	}

	// Sort the copied slice
	slices.Sort(sortedOrder)

	// Compare original order with sorted order
	if !slices.Equal(originalOrder, sortedOrder) {
		t.Errorf("IDs generated with delay are not lexicographically sorted:")
		t.Logf(" Original: %v", originalOrder)
		t.Logf(" Sorted:   %v", sortedOrder)
	}
}

// Test error when generating before the configured epoch
func Test_Generator_Generate_BeforeEpoch(t *testing.T) {
	futureEpoch := time.Now().UTC().Add(1 * time.Hour)
	gen, err := NewGenerator(Config{
		epoch:           futureEpoch,
		timeGranularity: Second,
		alphabet:        "01",
		randomChars:     1,
	})
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

	_, err = gen.Generate()
	if err == nil {
		t.Error("Expected an error when generating before epoch, but got nil")
	} else if !strings.Contains(err.Error(), "before the configured epoch") {
		t.Errorf("Expected 'before epoch' error, but got: %v", err)
	}
}

// Test generation with zero random characters
func Test_Generator_Generate_ZeroRandomChars(t *testing.T) {
	gen, err := NewGenerator(Config{
		epoch:           DefaultEpoch,
		timeGranularity: Millisecond,
		alphabet:        "0123456789",
		randomChars:     0, // No random part
	})
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

	id1, err1 := gen.Generate()
	id2, err2 := gen.Generate() // Generate immediately after

	if err1 != nil || err2 != nil {
		t.Fatalf("Generate() failed: %v, %v", err1, err2)
	}

	// Without random part, IDs generated within the same granularity tick should be identical
	if id1 != id2 {
		t.Errorf("Expected identical IDs with zero random chars within same tick, got %q and %q", id1, id2)
	}

	if !containsOnly(id1, "0123456789") {
		t.Errorf("ID %q contains characters outside the specified decimal alphabet", id1)
	}

	// Test after granularity tick passes
	time.Sleep(time.Duration(2) * time.Millisecond) // Sleep for longer than granularity
	id3, err3 := gen.Generate()
	if err3 != nil {
		t.Fatalf("Generate() failed: %v", err3)
	}
	if id1 == id3 {
		t.Errorf("Expected different IDs after granularity tick, but got identical %q", id1)
	}
}

func Test_NoTimeComponent(t *testing.T) {
	gen := MustNewGenerator(NewConfig().WithTimeGranularity(0).WithRandomChars(10))

	id1 := gen.MustGenerate()
	id2 := gen.MustGenerate()

	if id1 == id2 {
		t.Errorf("Expected different IDs, but got identical %q", id1)
	}
}

// Test internal encodeBaseN function
func Test_EncodeBaseN(t *testing.T) {
	// Use a dummy generator for testing encodeBaseN directly
	gen, _ := NewGenerator(DefaultConfig()) // Error check not needed for dummy

	testCases := []struct {
		name     string
		number   uint64
		alphabet string
		expected string
	}{
		{"Zero Base62", 0, DefaultAlphabet, "0"},
		{"Small Base62", 10, DefaultAlphabet, "A"},
		{"Larger Base62", 62, DefaultAlphabet, "10"},
		{"Large Base62", 1234567890, DefaultAlphabet, "1LY7VK"},
		{"Zero Base16", 0, "0123456789abcdef", "0"},
		{"Number Base16", 255, "0123456789abcdef", "ff"},
		{"Number Base16", 4096, "0123456789abcdef", "1000"},
		{"Zero Base2", 0, "01", "0"},
		{"Number Base2", 10, "01", "1010"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Temporarily modify generator's config for the test case
			gen.config.alphabet = tc.alphabet
			gen.base = len(tc.alphabet)

			result, err := gen.encodeBaseN(tc.number)
			if err != nil {
				t.Fatalf("encodeBaseN failed: %v", err)
			}
			if result != tc.expected {
				t.Errorf("encodeBaseN(%d, %q) = %q, want %q", tc.number, tc.alphabet, result, tc.expected)
			}
		})
	}
	// Restore default alphabet for subsequent tests if gen were reused (it isn't here)
	gen.config.alphabet = DefaultAlphabet
	gen.base = len(DefaultAlphabet)
}

// Test internal randomChars function
func Test_RandomChars(t *testing.T) {
	gen, _ := NewGenerator(DefaultConfig()) // Use default Base62 alphabet

	testCases := []struct {
		name     string
		length   int
		alphabet string
	}{
		{"Len 0", 0, DefaultAlphabet},
		{"Len 1 Base62", 1, DefaultAlphabet},
		{"Len 10 Base62", 10, DefaultAlphabet},
		{"Len 10 Base16", 10, "0123456789abcdef"},
		{"Len 5 Base2", 5, "01"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set generator alphabet for test case
			gen.config.alphabet = tc.alphabet
			gen.base = len(tc.alphabet)

			s1, err1 := gen.randomChars(tc.length)
			s2, err2 := gen.randomChars(tc.length) // Generate a second one

			if err1 != nil {
				t.Fatalf("randomChars(%d) #1 failed: %v", tc.length, err1)
			}
			if err2 != nil {
				t.Fatalf("randomChars(%d) #2 failed: %v", tc.length, err2)
			}

			if len(s1) != tc.length {
				t.Errorf("randomChars(%d) produced string of length %d, want %d", tc.length, len(s1), tc.length)
			}
			if !containsOnly(s1, tc.alphabet) {
				t.Errorf("randomChars produced string %q with characters outside alphabet %q", s1, tc.alphabet)
			}

			// Probabilistic check for randomness (should be different unless len=0)
			if tc.length > 0 && s1 == s2 {
				t.Logf("randomChars produced identical strings %q twice (rare but possible)", s1)
			} else if tc.length == 0 && (s1 != "" || s2 != "") {
				t.Errorf("randomChars(0) produced non-empty string: %q, %q", s1, s2)
			}
		})
	}
}

// Example showing how granularity affects the timestamp part
func Test_TimeGranularityEffect(t *testing.T) {
	epoch := time.Now().UTC().Add(-5 * time.Second) // Recent epoch

	genSec, _ := NewGenerator(Config{epoch: epoch, timeGranularity: Second, alphabet: "0123456789", randomChars: 2})
	genMs, _ := NewGenerator(Config{epoch: epoch, timeGranularity: Millisecond, alphabet: "0123456789", randomChars: 2})

	// Generate multiple quickly within the same second but different milliseconds
	idSec1, _ := genSec.Generate()
	time.Sleep(5 * time.Millisecond)
	idMs1, _ := genMs.Generate()
	time.Sleep(5 * time.Millisecond)
	idSec2, _ := genSec.Generate()
	time.Sleep(5 * time.Millisecond)
	idMs2, _ := genMs.Generate()

	// Extract timestamp parts (assuming random part has fixed length here)
	tsSec1 := idSec1[:len(idSec1)-genSec.config.randomChars]
	tsSec2 := idSec2[:len(idSec2)-genSec.config.randomChars]
	tsMs1 := idMs1[:len(idMs1)-genMs.config.randomChars]
	tsMs2 := idMs2[:len(idMs2)-genMs.config.randomChars]

	if tsSec1 != tsSec2 {
		t.Errorf("Second granularity: Expected same timestamp part for IDs generated within the same second, got %q and %q", tsSec1, tsSec2)
	}
	if tsMs1 == tsMs2 {
		t.Errorf("Millisecond granularity: Expected different timestamp parts for IDs generated ~5ms apart, got identical %q", tsMs1)
	}

	// Generate after more than a second
	time.Sleep(1100 * time.Millisecond)
	idSec3, _ := genSec.Generate()
	tsSec3 := idSec3[:len(idSec3)-genSec.config.randomChars]

	if tsSec1 == tsSec3 {
		t.Errorf("Second granularity: Expected different timestamp parts after >1sec delay, got identical %q", tsSec1)
	}
}

func Test_Sandbox(t *testing.T) {
	gen, _ := NewGenerator(Config{
		epoch: DefaultEpoch,
		//epoch:          time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		timeGranularity: Millisecond,
		alphabet:        Base16LowerAlphabet,
		randomChars:     6,
	})

	ids := make(map[string]struct{})
	for i := 0; i < 2; i++ {
		id, _ := gen.Generate()
		ids[id] = struct{}{}
		fmt.Printf("%s ", id)
		if i%10 == 9 {
			fmt.Printf("\n")
		}
		time.Sleep(time.Duration(1) * time.Millisecond * 5)
	}
	fmt.Printf("\n")
	fmt.Printf("Generated %d unique IDs\n", len(ids))
}

func containsOnly(s string, alphabet string) bool {
	for _, r := range s {
		if !strings.ContainsRune(alphabet, r) {
			return false
		}
	}
	return true
}
