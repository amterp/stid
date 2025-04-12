# Go STID: Short Time IDs

Generate short string IDs with a time and random component. Also referred to as "STIDs": **S**hort **T**ime **ID**s.

Useful for when you want to *guarantee* 0 collisions between two points in time, while minimizing collisions for
generated IDs *within* that time.

**For example:**

> Generate base-62 IDs with a time granularity of 1 millisecond, and with 5 extra random characters at the end.

## Installation

```sh
go get github.com/amterp/stid
```

## Usage

### Basic

You can use the default generator, which uses default settings.

```go
import "github.com/amterp/stid"

id := stid.MustGenerate()
anotherId, err := stdin.Generate()
```

### Advanced (Custom Settings)

You can create your own `Generator` by passing it your own `Config` object.

```go
import "github.com/amterp/stid"

// NewConfig creates with defaults.
// You can then chain With methods to customize settings.
config := stid.NewConfig()
	WithTimeGranularity(stid.TimeGranularity(1000)).
	WithRandomChars(6).
	WithAlphabet(stid.Base16LowerAlphabet)

// Create the generator with the config.
generator, err := stid.NewGenerator(config)

// Use it to generate ids.
id := generator.MustGenerate()
```

## How does it work?

It's simple!

TLDR: Given an alphabet, encode a ticking epoch timestamp, and append random characters from the alphabet.

### Time Component

Every ID begins with an encoded epoch timestamp. You can specify the granularity.

For example, if you specify a 100ms granularity, this timestamp-derived segment will "tick over" every 100 milliseconds,
guaranteeing uniqueness across 100 millisecond increments. This "time" component also gets shorter at lower
granularities.

Some examples of how **just this beginning segment** varies depending on the granularity:

| Granularity        | Example   |
|--------------------|-----------|
| Millisecond        | `Ui8NksP` |
| Decisecond (100ms) | `J2YxUE`  |
| Second             | `1u3UkZ`  |
| Hour               | `223a`    |
| Day                | `5Fe`     |

By default, the epoch start time is the traditional UNIX epoch of 1970-01-01. You can override this, however, to reduce
the size of the time component.

The following table compares IDs generated off the UNIX epoch vs. a 2025-01-01 epoch, generated in 2025-04.

| Granularity        | UNIX (1970) Epoch Example | 2025-01 Epoch Example | Character reduction |
|--------------------|---------------------------|-----------------------|---------------------|
| Millisecond        | `Ui8NksP`                 | `9YGDBT`              | -1                  |
| Decisecond (100ms) | `J2YxUE`                  | `5vCer`               | -1                  |
| Second             | `1u3UkZ`                  | `aifP`                | -2                  |
| Hour               | `223a`                    | `dC`                  | -2                  |
| Day                | `5Fe`                     | `1d`                  | -1                  |

Note, that by starting with the time component, IDs generated with the same granularity are chronologically sortable.

### Random Component

The time component, by itself, guarantees uniqueness *across* time granularity ticks. However, to avoid collisions
*within* a time tick, we add a "random component". Simply put, we randomly select characters from a given alphabet.

Using the default base-62 as an example, each appended character reduces the likelihood of collision by a factor of 62.
If we use 5 random characters, that's `62^5 = 916,132,832` unique possibilities. So, for any two IDs generated in the same granularity tick, the odds of them
colliding is 1 in 916,132,832.

That said, be aware of the [Birthday Problem](https://en.wikipedia.org/wiki/Birthday_problem) and
the [Pigeonhole](https://en.wikipedia.org/wiki/Pigeonhole_principle) principle.

## ID Examples

Below are some examples of IDs generated with different settings.

| Settings                                                   | Example             |
|------------------------------------------------------------|---------------------|
| Base-62, UNIX epoch, millisecond, 5 random chars (default) | `Ui8TX1zB2Avb`      |
| Base-36, UNIX epoch, millisecond, 5 random chars           | `m9dw96b1y9gtx`     |
| Base-62, UNIX epoch, decisecond, 5 random chars            | `J2Z31IPSxtl`       |
| Base-62, 2025-01 epoch, decisecond, 5 random chars         | `5vHWk2ayCn`        |
| Base-64, UNIX epoch, hour, 5 random chars                  | `B2TXxM3k1`         |
| Base-16, UNIX epoch, millisecond, 6 random chars           | `19628e9e59adc559d` |
