# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [v0.1.0] - 2026-04-03

### Added

- `json` package: case-sensitive JSON unmarshaling via `Unmarshal` and `Decoder`
- `xml` package: case-sensitive XML unmarshaling via `Unmarshal` and `Decoder`
- Case-sensitive field matching at all nesting levels (nested structs, slices of structs)
- Support for embedded structs (including via pointer)
- `DisallowUnknownFields` for strict validation on both packages
- Trailing data protection on JSON `Decoder` (rejects by default, configurable via `AllowTrailingData`)
- XML attribute support with case-sensitive matching (`xml:",attr"`)
- Field map caching via `sync.Map` for repeated unmarshal calls
- CI with GitHub Actions (build, test, lint)
