package stid

import (
	"crypto/rand"
	"errors"
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

// TimeGranularity represents the time granularity in milliseconds.
type TimeGranularity int

// Common TimeGranularity values in milliseconds.
const (
	Millisecond TimeGranularity = 1
	Centisecond TimeGranularity = 10
	Decisecond  TimeGranularity = 100
	Second      TimeGranularity = 1000
	Minute      TimeGranularity = 60000
	Hour        TimeGranularity = 3600000
	Day         TimeGranularity = 86400000
)

// Config holds the configuration for generating short TIDs.
type Config struct {
	epoch           time.Time       // The starting point for the time component (UTC recommended).
	timeGranularity TimeGranularity // The granularity of the time component.
	alphabet        string          // The alphabet used for encoding timestamp and random parts.
	randomChars     int             // The number of random characters to append.
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
	defaultGenerator, err = NewGenerator(DefaultConfig())
	if err != nil {
		panic("stid: failed to initialize default generator: " + err.Error())
	}
}

// DefaultConfig returns a default configuration:
// - epoch: Unix epoch (1970-01-01 00:00:00 UTC)
// - TimeGranularity: Millisecond (1ms)
// - alphabet: Base62
// - randomChars: 5
func DefaultConfig() Config {
	return Config{
		epoch:           DefaultEpoch,
		timeGranularity: Millisecond,
		alphabet:        DefaultAlphabet,
		randomChars:     5,
	}
}

func NewConfig() Config {
	return DefaultConfig()
}

// WithEpoch sets the epoch for the generator.
func (c Config) WithEpoch(epoch time.Time) Config {
	c.epoch = epoch
	return c
}

// WithTimeGranularity sets the time granularity for the generator.
func (c Config) WithTimeGranularity(granularity TimeGranularity) Config {
	c.timeGranularity = granularity
	return c
}

// WithAlphabet sets the alphabet for the generator.
func (c Config) WithAlphabet(alphabet string) Config {
	c.alphabet = alphabet
	return c
}

// WithRandomChars sets the number of random characters for the generator.
func (c Config) WithRandomChars(randomChars int) Config {
	c.randomChars = randomChars
	return c
}

// NewGenerator creates a new Generator instance with the given configuration.
// It validates the configuration upon creation.
func NewGenerator(config Config) (*Generator, error) {
	if len(config.alphabet) < 2 {
		return nil, errors.New("alphabet must contain at least 2 characters")
	}

	if config.randomChars < 0 {
		return nil, errors.New("number of random characters cannot be negative")
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
	now := time.Now().UTC() // Use UTC for consistent comparison with epoch

	// Check if current time is before the configured epoch. This must be done
	// here, as 'now' is only known at generation time. Allow generation at epoch time.
	if now.Before(g.config.epoch) {
		return "", errors.New("current time is before the configured epoch")
	}

	// 2. Encode timestamp ticks (if applicable)
	encodedTimestamp := ""
	if g.config.timeGranularity > 0 {
		delta := now.Sub(g.config.epoch)
		ticks := uint64(delta.Milliseconds() / int64(g.config.timeGranularity))
		encoded, err := g.encodeBaseN(ticks)
		if err != nil {
			return "", err
		}
		encodedTimestamp = encoded
	}

	// 3. Generate random part
	randomPart := ""
	if g.config.randomChars > 0 {
		chars, err := g.randomChars(g.config.randomChars)
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

// randomChars generates a cryptographically secure random string of the specified length
// using the generator's alphabet, avoiding modulo bias via rejection sampling.
func (g *Generator) randomChars(length int) (string, error) {
	if length == 0 {
		return "", nil
	}

	bytes := make([]byte, length)       // Buffer for resulting characters
	randomBytes := make([]byte, length) // Temporary buffer for OS random bytes
	maxValidByte := byte((256/g.base)*g.base - 1)

	for i := 0; i < length; {
		if _, err := io.ReadFull(rand.Reader, randomBytes); err != nil {
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
