# Go FlexID: Flexible IDs

A Go library for generating short, configurable string IDs with a time and random component.
Also referred to as "FIDs": **F**lexible **ID**s.

Useful for when you want to *guarantee* 0 collisions between two points in time, while minimizing collisions for
generated IDs *within* that time.

**For example:**

> Generate base-62 IDs with a tick size of 100 milliseconds, and with 5 extra random characters at the end.

May generate `J3GzHY02O6S`.

## Features ✨

- **Highly Configurable:**
  - Set your own **epoch** (start date/time).
  - Adjust the **tick size** (milliseconds, seconds, minutes, etc.).
  - Choose different **alphabets** (Base62, Base16 (hex), Base64URL, Crockford Base32, or custom).
  - Control the **length** of the random part. Reduce for shorter IDs, increase for greater collision resistance.
- **Short:** Generates compact IDs using configurable character sets (alphabets).
- **Collision Resistant:** Cryptographically secure random suffix minimizes collision probability.
- **Easy to Use:** Get started with sensible defaults or create fine-tuned generators.

## Installation 🚀

```sh
go get github.com/amterp/flexid
```

## Usage 🔨

### Basic

You can use the default generator, which uses some sensible default settings.

```go
import fid "github.com/amterp/flexid"

id := fid.MustGenerate()
anotherId, err := fid.Generate()
```

### Advanced (Custom Settings)

You can create your own `Generator` by passing it your own `Config` object.

```go
import fid "github.com/amterp/flexid"

// NewConfig creates with defaults.
// You can then chain With methods to customize settings.
config := fid.NewConfig().
	WithTickSize(fid.Second).
	WithNumRandomChars(6).
	WithAlphabet(fid.Base16LowerAlphabet)

// Create the generator with the config.
generator, err := fid.NewGenerator(config)
if err != nil {
    panic(err)
}

// Use it to generate ids.
id := generator.MustGenerate()
```

## How does it work? 🤔

It's simple!

TLDR: Given an alphabet, encode an epoch timestamp, and append random characters from the alphabet.

### Time Component

Every ID begins with an encoded epoch timestamp. This timestamp is measured in *ticks*. The tick size is configurable
and defaults to one millisecond.

For example, if you specify a 100ms tick size, the time component will be the number of 100ms ticks since epoch.
On every new tick, the time component changes, guaranteeing uniqueness across each increment of your tick size (100ms in this example).
The larger your tick size, the shorter the time component will be when encoded (as the number of ticks since epoch will go down).

Some examples of how **just this beginning segment** varies depending on the tick size:

| Tick Size          | Example   |
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

Note, that by starting with the time component, IDs generated with the same tick size are chronologically sortable.

### Random Component

The time component, by itself, guarantees uniqueness between ticks. However, to avoid collisions
*within* the same tick, we add a "random component". Simply put, we randomly select characters from a given alphabet.

Using the default base-62 as an example, each appended character reduces the likelihood of collision by a factor of 62.
If we use 5 random characters, that's `62^5 = 916,132,832` unique possibilities. So, for any two IDs generated in the same granularity tick, the odds of them
colliding is 1 in 916,132,832.

That said, be aware of the [Birthday Problem](https://en.wikipedia.org/wiki/Birthday_problem) and
the [Pigeonhole](https://en.wikipedia.org/wiki/Pigeonhole_principle) principle.

## ID Examples 📗

Below are some examples of IDs generated with different settings.

| Settings                                                   | Example             |
|------------------------------------------------------------|---------------------|
| Base-62, UNIX epoch, millisecond, 5 random chars (default) | `Ui8TX1zB2Avb`      |
| Base-36, UNIX epoch, millisecond, 5 random chars           | `m9dw96b1y9gtx`     |
| Base-62, UNIX epoch, decisecond, 5 random chars            | `J2Z31IPSxtl`       |
| Base-62, 2025-01 epoch, decisecond, 5 random chars         | `5vHWk2ayCn`        |
| Base-64, UNIX epoch, hour, 5 random chars                  | `B2TXxM3k1`         |
| Base-16, UNIX epoch, millisecond, 6 random chars           | `19628e9e59adc559d` |

## Why FlexIDs?

There are alternatives like UUIDs, NanoIDs, ULIDs, etc, so what do FIDs offer over these?

- **Configurability:** Fine-tune the epoch, tick size, alphabet, and random suffix length to precisely balance ID length, sortability, and collision resistance for your specific needs.
  - Need short IDs for a system with a known limited lifespan? Adjust the epoch and tick size.
  - Need higher collision resistance within a tick? Increase the random length.
- **Brevity:** By configuring the epoch and tick size appropriately, FlexID can generate significantly shorter IDs than alternatives like ULID or UUID, while retaining chronological sortability.
- **Simplicity:** The underlying concept (time prefix + random suffix) is straightforward and easy to reason about.

## Performance

Generating FIDs is very fast! There's no state or locking - they'll generate as fast as your CPU can go!

Benchmarking on an Apple M2 Pro, I get ~235 nanoseconds / op, or around 4-5 million IDs per second.

## Contributing 🙏

Contributions are welcome! Please feel free to open an issue or submit a pull request.

## License 📜

This library is licensed under the [MIT license](./LICENSE).
