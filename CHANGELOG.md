# Change Log

All notable changes to this project will be documented in this file.
This project adheres to [Semantic Versioning](http://semver.org/).

## [Unreleased]

### Changed

- Update dependencies (#167)
    - Update Golang to 1.18

## [2.3.0] - 2022-04-22

### Added

- Add simple NUMA support (#164)

## [2.2.0] - 2022-04-18

### Added

- Flexible SMP configuration for nodes (#160)

### Changed

- Update dependencies (#159)

## [2.1.0] - 2022-03-15

### Added

- Support multiple device backends for placemat VMs (#156)

## [2.0.6] - 2022-03-08

### Changed

- Change LICENSE from MIT to Apache 2.0 (#150)
- Support Multiqueue virtio-net (#152)

### Fixed

- Disable rp_filter inside a netns (#153)

## [2.0.5] - 2021-04-02

### Fixed

- Fix Internal Server Error when vm's power status cannot be confirmed (#145)

## [2.0.4] - 2021-03-22

### Fixed

- Fix example (#141)
  - The PR #141 also fixed the bug that doesn't expand SUDO_USER environment variable.

## [2.0.3] - 2021-03-17

### Fixed

- Fix BMC server startup failure (#139)
- Stabilize small test by extending timeout period (#138)
- Import cybozu-go/netutil and replace DetectMTU (#137)

## [2.0.2] - 2021-02-19

### Fixed

- dcnet: detect MTU by "ip route get" (#135)

## [2.0.1] - 2021-02-12

### Fixed

- Fix goroutine captures loop variable. (#133)

## [2.0.0] - 2021-02-08

See [upgrade_v2.md](docs/upgrade_v2.md) for more information.

## [1.5.3] - 2020-09-29

### Fixed

- Activate vhost-net. (#119)

## [1.5.2] - 2020-09-29

### Fixed

- Randomize MAC address for KVM NICs (#117)

## [1.5.1] - 2020-07-21

### Fixed

- Fix aio parameter for node volume devices when cache is specified (#115)

## [1.5.0] - 2020-07-20

### Added

- Add cache mode parameter for node volume devices (#113).
- Support creating node volume devices using raw format files (#113).
- Support creating node volume devices using LVs on host machine (#113).

## [1.4.0] - 2019-12-09

### Added

- Add stub HTTPS server for virtual BMC (#101).

## [1.3.9] - 2019-10-11

### Changed

- Add `iptables` rules for internal networking (#98).

## [1.3.8] - 2019-10-01

### Changed

- Use host CPU flags with `qemu -cpu host` for stability (#96).
- Replace yaml library (#94).

## [1.3.7] - 2019-07-26

### Added

- Add qemu option to use para-virtualized RNG for fast boot (#92).

## [1.3.6] - 2019-07-22

### Added

- Software TPM support (#91).

## [1.3.5] - 2019-03-15

### Added

- [`pmctl`](docs/pmctl.md) Add forward subcommand (#85).

## [1.3.4] - 2019-03-11

### Changed

- Wait resuming VMs after saving/loading snapshots (#83).

## [1.3.3] - 2019-03-04

### Changed

- Use formal import path for k8s.io/apimachinery (#82).

## [1.3.2] - 2019-02-18

### Changed

- [`pmctl`](docs/pmctl.md) Exit abnormally when failed to connect to server (#81).

## [1.3.1] - 2019-01-22

### Added

- [`pmctl`](docs/pmctl.md) Add snapshot list command. (#80)

## [1.3.0] - 2019-01-18

### Added

- [`pmctl`](docs/pmctl.md) Add snapshot subcommand. (#79)

## [1.2.0] - 2018-12-07

### Added

- [`pmctl`](docs/pmctl.md) Add completion subcommand. (#73)
- Release Debian Package. (#74)

### Changed

- Use fixed Debian image. (#72)

## [1.1.0] - 2018-11-06

### Added

- [`pmctl`](docs/pmctl.md) is a command-line client to control placemat.

### Removed

- `placemat-connect` as it is replaced by `pmctl`.

## [1.0.1] - 2018-10-23

### Changed

- Use cybozu-go/well instead of cybozu-go/cmd

## [1.0.0] - 2018-10-21

### Added

- Many things.  See git log.

[Unreleased]: https://github.com/cybozu-go/placemat/compare/v2.3.0...HEAD
[2.3.0]: https://github.com/cybozu-go/placemat/compare/v2.2.0...v2.3.0
[2.2.0]: https://github.com/cybozu-go/placemat/compare/v2.1.0...v2.2.0
[2.1.0]: https://github.com/cybozu-go/placemat/compare/v2.0.6...v2.1.0
[2.0.6]: https://github.com/cybozu-go/placemat/compare/v2.0.5...v2.0.6
[2.0.5]: https://github.com/cybozu-go/placemat/compare/v2.0.4...v2.0.5
[2.0.4]: https://github.com/cybozu-go/placemat/compare/v2.0.3...v2.0.4
[2.0.3]: https://github.com/cybozu-go/placemat/compare/v2.0.2...v2.0.3
[2.0.2]: https://github.com/cybozu-go/placemat/compare/v2.0.1...v2.0.2
[2.0.1]: https://github.com/cybozu-go/placemat/compare/v2.0.0...v2.0.1
[2.0.0]: https://github.com/cybozu-go/placemat/compare/v1.5.3...v2.0.0
[1.5.3]: https://github.com/cybozu-go/placemat/compare/v1.5.2...v1.5.3
[1.5.2]: https://github.com/cybozu-go/placemat/compare/v1.5.1...v1.5.2
[1.5.1]: https://github.com/cybozu-go/placemat/compare/v1.5.0...v1.5.1
[1.5.0]: https://github.com/cybozu-go/placemat/compare/v1.4.0...v1.5.0
[1.4.0]: https://github.com/cybozu-go/placemat/compare/v1.3.9...v1.4.0
[1.3.9]: https://github.com/cybozu-go/placemat/compare/v1.3.8...v1.3.9
[1.3.8]: https://github.com/cybozu-go/placemat/compare/v1.3.7...v1.3.8
[1.3.7]: https://github.com/cybozu-go/placemat/compare/v1.3.6...v1.3.7
[1.3.6]: https://github.com/cybozu-go/placemat/compare/v1.3.5...v1.3.6
[1.3.5]: https://github.com/cybozu-go/placemat/compare/v1.3.4...v1.3.5
[1.3.4]: https://github.com/cybozu-go/placemat/compare/v1.3.3...v1.3.4
[1.3.3]: https://github.com/cybozu-go/placemat/compare/v1.3.2...v1.3.3
[1.3.2]: https://github.com/cybozu-go/placemat/compare/v1.3.1...v1.3.2
[1.3.1]: https://github.com/cybozu-go/placemat/compare/v1.3.0...v1.3.1
[1.3.0]: https://github.com/cybozu-go/placemat/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/cybozu-go/placemat/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/cybozu-go/placemat/compare/v1.0.1...v1.1.0
[1.0.1]: https://github.com/cybozu-go/placemat/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/cybozu-go/placemat/compare/v0.1...v1.0.0
