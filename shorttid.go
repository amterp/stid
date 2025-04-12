package shorttid

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
	Epoch           time.Time       // The starting point for the time component (UTC recommended).
	TimeGranularity TimeGranularity // The granularity of the time component.
	Alphabet        string          // The alphabet used for encoding timestamp and random parts.
	RandomChars     int             // The number of random characters to append.
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
		panic("shorttid: failed to initialize default generator: " + err.Error())
	}
}

// DefaultConfig returns a default configuration:
// - Epoch: Unix Epoch (1970-01-01 00:00:00 UTC)
// - TimeGranularity: Millisecond (1ms)
// - Alphabet: Base62
// - RandomChars: 5
func DefaultConfig() Config {
	return Config{
		Epoch:           DefaultEpoch,
		TimeGranularity: Millisecond,
		Alphabet:        DefaultAlphabet,
		RandomChars:     5,
	}
}

func NewConfig() Config {
	return DefaultConfig()
}

// SetEpoch sets the epoch for the generator.
func (c *Config) SetEpoch(epoch time.Time) *Config {
	c.Epoch = epoch
	return c
}

// SetTimeGranularity sets the time granularity for the generator.
func (c *Config) SetTimeGranularity(granularity TimeGranularity) *Config {
	c.TimeGranularity = granularity
	return c
}

// SetAlphabet sets the alphabet for the generator.
func (c *Config) SetAlphabet(alphabet string) *Config {
	c.Alphabet = alphabet
	return c
}

// SetRandomChars sets the number of random characters for the generator.
func (c *Config) SetRandomChars(randomChars int) *Config {
	c.RandomChars = randomChars
	return c
}

// NewGenerator creates a new Generator instance with the given configuration.
// It validates the configuration upon creation.
func NewGenerator(config Config) (*Generator, error) {
	if config.TimeGranularity <= 0 {
		return nil, errors.New("granularity must be positive")
	}

	if len(config.Alphabet) < 2 {
		return nil, errors.New("alphabet must contain at least 2 characters")
	}

	if config.RandomChars < 0 {
		return nil, errors.New("number of random characters cannot be negative")
	}

	return &Generator{
		config: config,
		base:   len(config.Alphabet),
	}, nil
}

// Generate creates a new short TID using the generator's configuration.
func (g *Generator) Generate() (string, error) {
	// 1. Calculate timestamp ticks since configured epoch
	now := time.Now().UTC() // Use UTC for consistent comparison with epoch

	// Check if current time is before the configured epoch. This must be done
	// here, as 'now' is only known at generation time. Allow generation at epoch time.
	if now.Before(g.config.Epoch) {
		return "", errors.New("current time is before the configured epoch")
	}

	delta := now.Sub(g.config.Epoch)
	ticks := uint64(delta.Milliseconds() / int64(g.config.TimeGranularity))

	// 2. Encode timestamp ticks
	encodedTimestamp, err := g.encodeBaseN(ticks)
	if err != nil {
		return "", err
	}

	// 3. Generate random part
	randomPart := ""
	if g.config.RandomChars > 0 {
		randomPart, err = g.randomChars(g.config.RandomChars)
		if err != nil {
			return "", err
		}
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
		panic("shorttid: default generator not initialized")
	}
	return defaultGenerator.Generate()
}

// encodeBaseN encodes a non-negative integer using the generator's alphabet.
func (g *Generator) encodeBaseN(number uint64) (string, error) {
	if number == 0 {
		return string(g.config.Alphabet[0]), nil
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
		buf[i] = g.config.Alphabet[remainder]
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
				bytes[i] = g.config.Alphabet[int(randomByte)%g.base]
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
