# Changelog

## [v0.7.0](https://github.com/k1LoW/grpcstub/compare/v0.6.2...v0.7.0) - 2023-03-17
- [BREAKING] Support dir path for `Proto` by @k1LoW in https://github.com/k1LoW/grpcstub/pull/25
- [BREAKING] Skip registration of conflicted descriptors by @k1LoW in https://github.com/k1LoW/grpcstub/pull/27

## [v0.6.2](https://github.com/k1LoW/grpcstub/compare/v0.6.1...v0.6.2) - 2023-03-07
- Bump golang.org/x/text from 0.3.3 to 0.3.8 by @dependabot in https://github.com/k1LoW/grpcstub/pull/22
- Bump golang.org/x/net from 0.0.0-20201021035429-f5854403a974 to 0.7.0 by @dependabot in https://github.com/k1LoW/grpcstub/pull/24

## [v0.6.1](https://github.com/k1LoW/grpcstub/compare/v0.6.0...v0.6.1) - 2022-10-09
- Always keep file paths unique by @k1LoW in https://github.com/k1LoW/grpcstub/pull/20

## [v0.6.0](https://github.com/k1LoW/grpcstub/compare/v0.5.1...v0.6.0) - 2022-10-09
- [BREAKING] Add Option and Change function signature of NewServer() by @k1LoW in https://github.com/k1LoW/grpcstub/pull/17

## [v0.5.1](https://github.com/k1LoW/grpcstub/compare/v0.5.0...v0.5.1) - 2022-10-09
- Use tagpr by @k1LoW in https://github.com/k1LoW/grpcstub/pull/15

## [v0.5.0](https://github.com/k1LoW/grpcstub/compare/v0.4.0...v0.5.0) (2022-07-15)

* Support TLS [#14](https://github.com/k1LoW/grpcstub/pull/14) ([k1LoW](https://github.com/k1LoW))
* Add Server.ClientConn as alias [#13](https://github.com/k1LoW/grpcstub/pull/13) ([k1LoW](https://github.com/k1LoW))

## [v0.4.0](https://github.com/k1LoW/grpcstub/compare/v0.3.0...v0.4.0) (2022-07-10)

* gRPC conn close before server close [#12](https://github.com/k1LoW/grpcstub/pull/12) ([k1LoW](https://github.com/k1LoW))
* Fix grpc.Dial option [#11](https://github.com/k1LoW/grpcstub/pull/11) ([k1LoW](https://github.com/k1LoW))

## [v0.3.0](https://github.com/k1LoW/grpcstub/compare/v0.2.4...v0.3.0) (2022-07-06)

* Only the first response sends Header in bidirectional streaming [#10](https://github.com/k1LoW/grpcstub/pull/10) ([k1LoW](https://github.com/k1LoW))

## [v0.2.4](https://github.com/k1LoW/grpcstub/compare/v0.2.3...v0.2.4) (2022-07-05)

* Fix handle client streaming [#9](https://github.com/k1LoW/grpcstub/pull/9) ([k1LoW](https://github.com/k1LoW))

## [v0.2.3](https://github.com/k1LoW/grpcstub/compare/v0.2.2...v0.2.3) (2022-07-04)

* Fix keys convert: use OrigName option [#8](https://github.com/k1LoW/grpcstub/pull/8) ([k1LoW](https://github.com/k1LoW))
* The keys of the parameters of the recorded request message should be the same as in the proto file. [#7](https://github.com/k1LoW/grpcstub/pull/7) ([k1LoW](https://github.com/k1LoW))
* Use encoding/json [#6](https://github.com/k1LoW/grpcstub/pull/6) ([k1LoW](https://github.com/k1LoW))

## [v0.2.2](https://github.com/k1LoW/grpcstub/compare/v0.2.1...v0.2.2) (2022-07-03)

* Fix register desc [#5](https://github.com/k1LoW/grpcstub/pull/5) ([k1LoW](https://github.com/k1LoW))

## [v0.2.1](https://github.com/k1LoW/grpcstub/compare/v0.2.0...v0.2.1) (2022-07-03)

* Resolve relative proto paths for reflection [#4](https://github.com/k1LoW/grpcstub/pull/4) ([k1LoW](https://github.com/k1LoW))

## [v0.2.0](https://github.com/k1LoW/grpcstub/compare/v0.1.1...v0.2.0) (2022-07-03)

* Add Server.Addr() [#3](https://github.com/k1LoW/grpcstub/pull/3) ([k1LoW](https://github.com/k1LoW))

## [v0.1.1](https://github.com/k1LoW/grpcstub/compare/v0.1.0...v0.1.1) (2022-07-02)

* Add LICENSE [#2](https://github.com/k1LoW/grpcstub/pull/2) ([k1LoW](https://github.com/k1LoW))

## [v0.1.0](https://github.com/k1LoW/grpcstub/compare/3408f46825de...v0.1.0) (2022-07-02)

* Add response status handling [#1](https://github.com/k1LoW/grpcstub/pull/1) ([k1LoW](https://github.com/k1LoW))
