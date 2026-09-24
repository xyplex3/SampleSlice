# SampleSlice

![Go Version](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go)
![License](https://img.shields.io/badge/license-MIT-blue)
![Release](https://img.shields.io/github/v/release/xyplex3/SampleSlice)
![Build Status](https://img.shields.io/github/actions/workflow/status/xyplex3/SampleSlice/release.yml)

SampleSlice turns long recording sessions into ready-to-use drum samples and
loop libraries. Point it at a WAV file and it automatically detects every hit,
slices the audio apart, and exports the results directly into Akai MPC or
Kurzweil sampler formats. No DAW or manual editing required.

## Table of contents

- [Overview](#overview)
- [Features](#features)
- [Supported WAV input](#supported-wav-input)
- [Installation](#installation)
- [Quick start](#quick-start)
- [Usage](#usage)
- [Configuration](#configuration)
- [Shell completion](#shell-completion)
- [Testing](#testing)
- [About the author](#about-the-author)
- [License](#license)

## Overview

SampleSlice reads a WAV file, locates every hit, attack, or onset using
energy-based transient detection (or divides audio into equal bar-length loops
by tempo), and exports the resulting slices as ready-to-load sampler programs.
It targets Akai MPC hardware and software (`.xpm` format) and Kurzweil K-series
synthesizers (`.krz` format). A single command maps every slice to its own pad
or key, with optional normalization, silence trimming, deduplication, and MIDI
note assignment.

## Features

- Two slicing modes: energy-based transient detection or beat-grid
  (BPM-aligned equal-length bars)
- Multiple output formats: Akai MPC `.xpm` (XML + individual WAV files),
  Kurzweil `.krz`, or both in one pass
- Note mapping: sequential, General MIDI drum layout (`--gm-map`), or
  fully custom per-slice (`--note-map`)
- Per-slice processing: peak-normalize to 0 dBFS (`--normalize`) and strip
  inaudible silence (`--auto-trim`)
- Deduplication: remove near-duplicate slices by waveform energy profile
  (`--dedupe`)
- KRZ envelope shaping: built-in presets or a raw 13-value ADSR envelope
- KRZ VAST tone borrowing: splice a real patch's algorithm/filter/pan/amp
  settings from an existing KRZ file onto generated programs
- Transient report: export slice timing and MIDI metadata as JSON or CSV
- Shell completions: bash, zsh, fish, and PowerShell

## Supported WAV input

| Property | Supported values |
|----------|-----------------|
| Bit depth | 16-bit, 24-bit, 32-bit PCM |
| Channels | Mono or stereo (stereo is downmixed to mono) |
| Sample rate | Any (44100, 48000, 96000, etc.) |

## Installation

### Download a prebuilt binary

Prebuilt binaries for Linux (amd64/arm64), macOS (amd64/arm64), and
Windows (amd64) are published on the
[Releases page](https://github.com/xyplex3/SampleSlice/releases/latest).
Download the archive for your platform, extract it, and run the
`sampleslice` (or `sampleslice.exe`) binary directly — no Go toolchain
required.

### Build from source

#### Prerequisites

- Go 1.26 or higher

```bash
git clone https://github.com/xyplex3/SampleSlice.git
cd SampleSlice
go build -o sampleslice .
```

To embed version and commit metadata in the binary at build time:

```bash
go build -ldflags "-X sampleslice/cmd.Version=1.0.0 \
  -X sampleslice/cmd.Commit=$(git rev-parse --short HEAD) \
  -X sampleslice/cmd.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o sampleslice .
```

### Verification

```bash
./sampleslice version
# SampleSlice v1.0.0
# Commit: abc1234
# Built: 2026-05-11T00:00:00Z
```

## Quick start

Slice a drum loop into an MPC program in one command:

```bash
./sampleslice --input drums.wav
```

SampleSlice creates a `drums_sliced/` directory containing:

- `SampleSlice.xpm`: MPC drum program file
- `Samples/slice_001.wav`, `slice_002.wav`, …: individual slice WAV files

Load the `.xpm` file in MPC software or copy the directory to your MPC's
storage drive.

## Usage

### Transient detection mode (default)

The default mode finds every attack or onset in the audio automatically.

```bash
# Use defaults (sensitivity 0.5, 50ms minimum interval)
./sampleslice --input loop.wav

# More sensitive: catches quieter hits
./sampleslice --input loop.wav --sensitivity 0.2

# Stricter: only the loudest transients
./sampleslice --input loop.wav --sensitivity 0.8

# Require at least 80ms between detections (prevents double-triggering)
./sampleslice --input loop.wav --min-interval 80

# Wider energy-analysis window (default: 10ms)
./sampleslice --input loop.wav --window-size 20
```

#### Sensitivity guide

| Audio type | Recommended sensitivity |
|------------|------------------------|
| Quiet hits, ambient recordings | 0.2–0.3 |
| Standard drum loops | 0.4–0.6 |
| Dense rhythms, breakbeats | 0.6–0.8 |

### Beat-grid mode

When `--bpm` is set, SampleSlice ignores transients and slices the audio into
equal-length loops aligned to the tempo grid. Tail segments shorter than half
a bar are discarded.

```bash
# Slice at 120 BPM into 1-bar loops (default)
./sampleslice --input loop.wav --bpm 120

# 2-bar loops
./sampleslice --input loop.wav --bpm 120 --loop-bars 2

# 3/4 time at 90 BPM
./sampleslice --input loop.wav --bpm 90 --beats-per-bar 3
```

### Output formats

```bash
# Akai MPC .xpm (default): XML program + individual WAV files
./sampleslice --input loop.wav --format mpc

# Kurzweil .krz
./sampleslice --input loop.wav --format krz

# Both formats in one pass
./sampleslice --input loop.wav --format both
```

### Slice boundaries

```bash
# Tighter slices: 5ms pre-pad, 100ms post-pad
./sampleslice --input loop.wav --pre 5 --post 100
```

### Note mapping

```bash
# Start slices at C4 (MIDI 60)
./sampleslice --input loop.wav --root 60

# General MIDI drum layout (kick=36, snare=38, hi-hat=42, …)
./sampleslice --input loop.wav --gm-map

# Custom assignment: slice 0→36, slice 1→42, slice 2→46
./sampleslice --input loop.wav --note-map "0=36,1=42,2=46"
```

### Per-slice processing

```bash
# Normalize every slice to 0 dBFS
./sampleslice --input loop.wav --normalize

# Strip inaudible leading/trailing silence
./sampleslice --input loop.wav --auto-trim

# Adjust the silence threshold (default 0.001 ≈ −60 dBFS)
./sampleslice --input loop.wav --auto-trim --trim-floor 0.005

# Remove near-duplicate slices (0.95 = very similar; higher = stricter)
./sampleslice --input loop.wav --dedupe 0.95
```

### Transient report

```bash
# JSON report (timing, MIDI note, duration, energy per slice)
./sampleslice --input loop.wav --report json

# CSV (useful for importing slice markers into a DAW)
./sampleslice --input loop.wav --report csv
```

The report is written to `<output>/<name>_report.json` (or `.csv`) alongside
the program file.

### KRZ-specific options

```bash
# Built-in envelope presets: drum, perc, pad, key
./sampleslice --input loop.wav --format krz --envelope-preset perc

# Raw envelope (13 values 0-255): att1level,att1time,att2level,att2time,
#   att3level,att3time,dec1level,dec1time,rel1level,rel1time,rel2level,
#   rel2time,rel3time
./sampleslice --input loop.wav --format krz --envelope "100,0,0,0,0,0,0,10,0,0,0,0,5"

# Borrow a real patch's VAST tone (algorithm + filter/pan/amp settings)
# from an existing KRZ file and apply it to your sliced samples
./sampleslice --input loop.wav --format krz \
  --vast-from "MyLibrary.krz" --vast-program "Kick 909"
```

`--voice-mode`, `--priority`, `--stereo`, and `--krz-compress` are accepted
for forward/API compatibility but currently have **no effect** on the
generated KRZ bytes (the CLI warns if you set `--voice-mode poly`,
`--stereo`, or `--krz-compress`). Only `--envelope`/`--envelope-preset` and
`--vast-from`/`--vast-program` actually change KRZ output today.

## Configuration

### Config file

SampleSlice looks for `~/.sampleslice.yaml` or `.sampleslice.yaml` in the
current directory. Override the path with `--config`:

```bash
./sampleslice --config /path/to/config.yaml --input loop.wav
```

Example `~/.sampleslice.yaml`:

```yaml
sensitivity: 0.3
min-interval: 60
pre: 10
post: 150
normalize: true
auto-trim: true
format: mpc
```

### Environment variables

Any flag can be set via environment variable using the `SAMPLESLICE_` prefix:

```bash
export SAMPLESLICE_SENSITIVITY=0.3
export SAMPLESLICE_FORMAT=krz
./sampleslice --input loop.wav
```

Precedence (highest → lowest): CLI flags → environment variables →
config file → built-in defaults.

### All flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--input` | `-i` | *(required)* | Path to input WAV file |
| `--output` | `-o` | `<input>_sliced/` | Output directory |
| `--name` | `-n` | `SampleSlice` | Program name |
| `--format` | `-f` | `mpc` | Output format: `mpc`, `xpm`, `krz`, `both` |
| `--root` | `-r` | `48` (C3) | Root MIDI note number |
| `--sensitivity` | `-s` | `0.5` | Detection sensitivity (0.0–1.0) |
| `--min-interval` | `-m` | `50` | Minimum ms between transients |
| `--window-size` | | `10` | Energy window size in ms |
| `--pre` | `-p` | `20` | Pre-transient padding in ms |
| `--post` | `-P` | `200` | Post-transient padding in ms |
| `--bpm` | | `0` (off) | BPM for beat-grid slicing; 0 uses transient detection |
| `--loop-bars` | | `1` | Bars per slice in beat-grid mode |
| `--beats-per-bar` | | `4` | Time-signature numerator for beat-grid mode |
| `--normalize` | | `false` | Peak-normalize each slice to 0 dBFS |
| `--auto-trim` | | `false` | Strip leading/trailing silence |
| `--trim-floor` | | `0.001` | Noise floor for auto-trim (linear amplitude) |
| `--dedupe` | | `0` (off) | Similarity threshold for deduplication (0–1) |
| `--gm-map` | | `false` | Use General MIDI drum note layout |
| `--note-map` | | | Custom note assignments, e.g. `0=36,1=42` |
| `--report` | | *(disabled)* | Report format: `json` or `csv` |
| `--krz-version` | | `2000` | KRZ file format version |
| `--krz-compress` | | `false` | Reserved; no effect on KRZ output yet |
| `--voice-mode` | | `drum` | Reserved; no effect on KRZ output yet |
| `--priority` | | `7`/`3` | Reserved; no effect on KRZ output yet |
| `--stereo` | | `false` | Reserved; no effect on KRZ output yet |
| `--envelope` | | | Raw envelope: 13 comma-separated values (0–255) |
| `--envelope-preset` | | | Envelope preset: `drum`, `perc`, `pad`, `key` |
| `--vast-from` | | | Path to an existing KRZ file to borrow a VAST tone from |
| `--vast-program` | | | Program name within `--vast-from` to borrow (used together) |
| `--config` | | `~/.sampleslice.yaml` | Config file path |

## Shell completion

```bash
# Bash (Linux)
sampleslice completion bash > /etc/bash_completion.d/sampleslice

# Bash (macOS with Homebrew bash-completion@2)
sampleslice completion bash > /usr/local/etc/bash_completion.d/sampleslice

# Zsh
sampleslice completion zsh > "${fpath[1]}/_sampleslice"

# Fish
sampleslice completion fish > ~/.config/fish/completions/sampleslice.fish

# PowerShell
sampleslice completion powershell | Out-String | Invoke-Expression
```

## Testing

```bash
go test ./...
```

All packages include unit tests. The `test/` package runs a full end-to-end
pipeline test: WAV creation → transient detection → slicing → WAV export.

## About the author

SampleSlice is written by [xyplex3](https://github.com/xyplex3), a developer and musician based in Seattle, WA.

The musical project is **Xyplex2** — industrial, experimental, and distorted-beats music released on the Detroit Industrial label. The debut album *Second Shift* came out in May 2022 and is available on Bandcamp as a digital download or limited-edition USB + cassette.

Xyplex2 is based in the Seattle area and is available for underground techno and industrial shows. If you like this project please book Xyplex2 for shows!

- [Xyplex2 — *Second Shift* on Bandcamp](https://xyplex2.bandcamp.com/album/xyplex2-second-shift)
- [Supervisory Control (YouTube)](https://www.youtube.com/watch?v=Bz2r5YaahPc)
- [Second Shift (YouTube)](https://www.youtube.com/watch?v=wHfb6Ot2kn8)
- [Extreme Directions (YouTube)](https://www.youtube.com/watch?v=cxTlZX4JkG4)
- [Direct Object Reference (YouTube)](https://www.youtube.com/watch?v=dvZuhBXuwZU)
- [Xyplex2 on Instagram](https://www.instagram.com/xyplex2official/)

## License

MIT
