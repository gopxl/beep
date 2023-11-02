# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Added a changelog file ([#7](https://github.com/gopxl/beep/pull/7))
- Support for single channel ogg/vorbis ([#10](https://github.com/gopxl/beep/pull/10))

### Fixed
- Fix `FileSize` for saving .wav ([#6](https://github.com/gopxl/beep/pull/6))
- Fix `flac.Decode` handling of `io.EOF` ([#127](https://github.com/gopxl/beep/pull/127))

### Changed
- Upgrade Go version to 1.21 ([#2](https://github.com/gopxl/beep/pull/2))
- Upgrade Oto version to 3.1 ([#3](https://github.com/gopxl/beep/pull/3))
- Upgrade Tcell version to 2.6.0 ([#122](https://github.com/gopxl/beep/pull/122))
- Upgrade go-mp3 version to 0.3.4 ([#122](https://github.com/gopxl/beep/pull/122))
- Upgrade jfreymuth/oggvorbis version to 1.0.5 ([#122](https://github.com/gopxl/beep/pull/122))
- Upgrade mewkiz/flac version to 1.0.8 ([#122](https://github.com/gopxl/beep/pull/122))
- Panic when `Resampler` is given a ratio of `Inf` or `NaN`. ([#120](https://github.com/gopxl/beep/pull/120))

## [v1.0.0] 2023-10-07
- Forked [faiface/beep](https://github.com/faiface/beep) to [gopxl/beep](https://github.com/gopxl/beep).
