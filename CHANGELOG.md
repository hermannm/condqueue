# Changelog

## [v0.2.1] - 2023-11-17

- Change `AwaitMatchingItem` to call `context.Cause` on canceled context

## [v0.2.0] - 2023-11-01

- Improve performance by significantly reducing lock contention and cross-goroutine communication in
  `Add` and `AwaitMatchingItem`
- Remove redundant `Context` arguments from `Add` and `Clear`

## [v0.1.0] - 2023-09-29

- Initial release

[Unreleased]: https://github.com/hermannm/condqueue/compare/v0.2.1...HEAD

[v0.2.1]: https://github.com/hermannm/condqueue/compare/v0.2.0...v0.2.1

[v0.2.0]: https://github.com/hermannm/condqueue/compare/v0.1.0...v0.2.0

[v0.1.0]: https://github.com/hermannm/condqueue/compare/0fbef38...v0.1.0
