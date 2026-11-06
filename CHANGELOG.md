# Changelog

All notable changes to this project. Format based on [Keep a Changelog](https://keepachangelog.com/).

## [Unreleased]

## [0.2.0] - 2026-11-06

### Added

- Homebrew cask on `infomosquitos/homebrew-tap` (`brew install --cask infomosquitos/tap/ticketmosquito`), with release automation that bumps version/sha256.
- Chrome/Firefox 3-tab extension (Issue / Projects / Settings) with session capture, crawl, and cron controls.
- Extension Build CI workflow; release packages Chrome and Firefox extension zips.

### Changed

- Extension toolchain: esbuild + TypeScript 7 (`tsc --noEmit`).
- Popup accent color to blue.

### Fixed

- Jira session capture/validate (CHIPS cookies, identity dedupe).
- Extension project detection and null-safe popup text.
- Crawl project URL / key parsing and progress reporting.
- Server start crash.

## [0.1.0] - 2026-10-18

### Added

- Initial version copied from spacemosquito
