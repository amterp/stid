package stid

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math" // Import needed for the comment explanation
	"strings"
	"time"
)

// DefaultAlphabet is the standard base62 alphabet (0-9, A-Z, a-z).
const (
	DefaultAlphabet = Base62Alphabet

	Base62Alphabet      = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	Base36Alphabet      = "0123456789abcdefghijklmnopqrstuvwxyz"
	Base16LowerAlphabet = "0123456789abcdef"
	Base16UpperAlphabet = "0123456789ABCDEF"
	Base64UrlAlphabet   = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	// CrockfordBase32Alphabet is designed for human readability and is case-insensitive (excludes I, L, O, U).
	CrockfordBase32Alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
)

// Common tick size durations.
const (
	Nanosecond  = time.Nanosecond
	Microsecond = 1000 * Nanosecond
	Millisecond = 1000 * Microsecond
	Centisecond = 10 * Millisecond
	Decisecond  = 100 * Millisecond
	Second      = 1000 * Millisecond
	Minute      = 60 * Second
	Hour        = 60 * Minute
	Day         = 24 * Hour
)

// Config holds the configuration for generating short TIDs.
type Config struct {
	epoch          time.Time        // The starting point for the time component (UTC recommended).
	tickSize       time.Duration    // The tick size of the time component.
	alphabet       string           // The alphabet used for encoding timestamp and random parts.
	numRandomChars int              // The number of random characters to append.
	timeProvider   func() time.Time // Function to provide the current time (for testing).
	randomSource   io.Reader        // Source of randomness (for testing).
}

// Generator is responsible for generating TIDs based on a fixed configuration.
type Generator struct {
	config Config
	base   int // Cache the base (length of alphabet)
}

var (
	DefaultEpoch     = time.Unix(0, 0).UTC()
	defaultGenerator *Generator
)

func init() {
	var err error
	defaultGenerator, err = NewGenerator(NewConfig())
	if err != nil {
		panic("stid: failed to initialize default generator: " + err.Error())
	}
}

// NewConfig returns a default configuration:
// - epoch: Unix epoch (1970-01-01 00:00:00 UTC)
// - TickSize: Millisecond
// - alphabet: Base62
// - numRandomChars: 5
func NewConfig() Config {
	return Config{
		epoch:          DefaultEpoch,
		tickSize:       Millisecond,
		alphabet:       DefaultAlphabet,
		numRandomChars: 5,
		randomSource:   rand.Reader,
		timeProvider:   time.Now,
	}
}

// WithEpoch sets the epoch for the generator.
func (c Config) WithEpoch(epoch time.Time) Config {
	c.epoch = epoch
	return c
}

// WithTickSize sets the tick size for the time component of the generator.
func (c Config) WithTickSize(tickSize time.Duration) Config {
	c.tickSize = tickSize
	return c
}

// WithAlphabet sets the alphabet for the generator.
func (c Config) WithAlphabet(alphabet string) Config {
	c.alphabet = alphabet
	return c
}

// WithNumRandomChars sets the number of random characters for the generator.
func (c Config) WithNumRandomChars(numRandomChars int) Config {
	c.numRandomChars = numRandomChars
	return c
}

// WithRandomSource sets the random source for the generator.
func (c Config) WithRandomSource(randomSource io.Reader) Config {
	c.randomSource = randomSource
	return c
}

// WithTimeProvider sets the time provider for the generator.
func (c Config) WithTimeProvider(timeProvider func() time.Time) Config {
	c.timeProvider = timeProvider
	return c
}

// NewGenerator creates a new Generator instance with the given configuration.
// It validates the configuration upon creation.
func NewGenerator(config Config) (*Generator, error) {
	if len(config.alphabet) < 2 {
		return nil, errors.New("alphabet must contain at least 2 characters")
	}

	if config.numRandomChars < 0 {
		return nil, errors.New("number of random characters cannot be negative")
	}

	err := validateAlphabet(config.alphabet)
	if err != nil {
		return nil, err
	}

	return &Generator{
		config: config,
		base:   len(config.alphabet),
	}, nil
}

func MustNewGenerator(config Config) *Generator {
	generator, err := NewGenerator(config)
	if err != nil {
		panic("stid: failed to create generator: " + err.Error())
	}
	return generator
}

// Generate creates a new short TID using the generator's configuration.
func (g *Generator) Generate() (string, error) {
	// 1. Calculate timestamp ticks since configured epoch
	now := g.config.timeProvider().UTC()

	// Check if current time is before the configured epoch. This must be done
	// here, as 'now' is only known at generation time. Allow generation at epoch time.
	if now.Before(g.config.epoch) {
		return "", errors.New("current time is before the configured epoch")
	}

	// 2. Encode timestamp ticks (if applicable)
	encodedTimestamp := ""
	if g.config.tickSize > 0 {
		delta := now.Sub(g.config.epoch)
		ticks := uint64(delta.Nanoseconds() / int64(g.config.tickSize))
		encoded, err := g.encodeBaseN(ticks)
		if err != nil {
			return "", err
		}
		encodedTimestamp = encoded
	}

	// 3. Generate random part
	randomPart := ""
	if g.config.numRandomChars > 0 {
		chars, err := g.generateRandomChars(g.config.numRandomChars)
		if err != nil {
			return "", err
		}
		randomPart = chars
	}

	// 4. Combine parts
	var sb strings.Builder
	sb.WriteString(encodedTimestamp)
	sb.WriteString(randomPart)
	return sb.String(), nil
}

// Generate generates a TID using the default configuration.
// It panics if the internal default generator failed to initialize.
func Generate() (string, error) {
	if defaultGenerator == nil {
		panic("stid: default generator not initialized")
	}
	return defaultGenerator.Generate()
}

func MustGenerate() string {
	id, err := Generate()
	if err != nil {
		panic("stid: failed to generate TID: " + err.Error())
	}
	return id
}

func (g *Generator) MustGenerate() string {
	id, err := g.Generate()
	if err != nil {
		panic("stid: failed to generate TID: " + err.Error())
	}
	return id
}

// encodeBaseN encodes a non-negative integer using the generator's alphabet.
func (g *Generator) encodeBaseN(number uint64) (string, error) {
	if number == 0 {
		return string(g.config.alphabet[0]), nil
	}

	// Estimate buffer size: log_base(number). Rough estimate is fine.
	bufSize := int(64/math.Log2(float64(g.base))) + 2 // Add 2 for safety
	buf := make([]byte, bufSize)
	i := bufSize - 1

	for number > 0 {
		if i < 0 {
			return "", errors.New("buffer size estimation failed in encodeBaseN")
		}
		remainder := number % uint64(g.base)
		buf[i] = g.config.alphabet[remainder]
		number /= uint64(g.base)
		i--
	}

	return string(buf[i+1:]), nil
}

// generateRandomChars generates a cryptographically secure random string of the specified length
// using the generator's alphabet, avoiding modulo bias via rejection sampling.
func (g *Generator) generateRandomChars(length int) (string, error) {
	if length == 0 {
		return "", nil
	}

	bytes := make([]byte, length)       // Buffer for resulting characters
	randomBytes := make([]byte, length) // Temporary buffer for OS random bytes
	maxValidByte := byte((256/g.base)*g.base - 1)

	for i := 0; i < length; {
		if _, err := io.ReadFull(g.config.randomSource, randomBytes); err != nil {
			return "", errors.New("failed to read random bytes: " + err.Error())
		}

		for _, randomByte := range randomBytes {
			if randomByte <= maxValidByte {
				bytes[i] = g.config.alphabet[int(randomByte)%g.base]
				i++
				if i == length {
					break // Got all required characters
				}
			}
			// Discard biased byte (randomByte > maxValidByte) and continue
		}
	}

	return string(bytes), nil
}

// ensure alphabet doesn't contain duplicates
func validateAlphabet(alphabet string) error {
	seen := make(map[rune]struct{})
	for _, ch := range alphabet {
		if _, exists := seen[ch]; exists {
			return fmt.Errorf("alphabet contains duplicate character: %c", ch)
		}
		seen[ch] = struct{}{}
	}
	return nil
}
