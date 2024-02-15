# Changelog

## [v0.17.0](https://github.com/k1LoW/grpcstub/compare/v0.16.0...v0.17.0) - 2024-02-15
### Other Changes
- Update pkgs by @k1LoW in https://github.com/k1LoW/grpcstub/pull/68

## [v0.16.0](https://github.com/k1LoW/grpcstub/compare/v0.15.3...v0.16.0) - 2024-01-12
### Breaking Changes 🛠
- Fix to accept *testing.B by @k1LoW in https://github.com/k1LoW/grpcstub/pull/67

## [v0.15.3](https://github.com/k1LoW/grpcstub/compare/v0.15.2...v0.15.3) - 2023-10-30
### New Features 🎉
- Add methods for clearing by @k1LoW in https://github.com/k1LoW/grpcstub/pull/64

## [v0.15.2](https://github.com/k1LoW/grpcstub/compare/v0.15.1...v0.15.2) - 2023-10-26
### Other Changes
- Bump google.golang.org/grpc from 1.57.0 to 1.57.1 by @dependabot in https://github.com/k1LoW/grpcstub/pull/62

## [v0.15.1](https://github.com/k1LoW/grpcstub/compare/v0.15.0...v0.15.1) - 2023-10-12
### Other Changes
- Bump golang.org/x/net from 0.9.0 to 0.17.0 by @dependabot in https://github.com/k1LoW/grpcstub/pull/61

## [v0.15.0](https://github.com/k1LoW/grpcstub/compare/v0.14.0...v0.15.0) - 2023-09-25
### New Features 🎉
- Response() supports accepting any value by @k1LoW in https://github.com/k1LoW/grpcstub/pull/58

## [v0.14.0](https://github.com/k1LoW/grpcstub/compare/v0.13.0...v0.14.0) - 2023-09-22
### Breaking Changes 🛠
- Requests() returns matched requests only and UnmatchedRequests() returns unmatched requests by @k1LoW in https://github.com/k1LoW/grpcstub/pull/56
### New Features 🎉
- Add Stringer for Request type by @k1LoW in https://github.com/k1LoW/grpcstub/pull/57

## [v0.13.0](https://github.com/k1LoW/grpcstub/compare/v0.12.0...v0.13.0) - 2023-09-21
### Breaking Changes 🛠
- Fix resolvePaths() to handle relative paths correctly by @k1LoW in https://github.com/k1LoW/grpcstub/pull/53

## [v0.12.0](https://github.com/k1LoW/grpcstub/compare/v0.11.1...v0.12.0) - 2023-09-15
### Breaking Changes 🛠
- Use github.com/bufbuild/protocompile instead of github.com/jhump/protoreflect by @k1LoW in https://github.com/k1LoW/grpcstub/pull/51

## [v0.11.1](https://github.com/k1LoW/grpcstub/compare/v0.11.0...v0.11.1) - 2023-06-08
### Other Changes
- Add DisableReflection option by @k1LoW in https://github.com/k1LoW/grpcstub/pull/49

## [v0.11.0](https://github.com/k1LoW/grpcstub/compare/v0.10.2...v0.11.0) - 2023-06-07
### Breaking Changes 🛠
- Use google.golang.org/protobuf/reflect/protoreflect instead of github.com/jhump/protoreflect by @k1LoW in https://github.com/k1LoW/grpcstub/pull/48

## [v0.10.2](https://github.com/k1LoW/grpcstub/compare/v0.10.1...v0.10.2) - 2023-05-17
- Change health check service name by @k1LoW in https://github.com/k1LoW/grpcstub/pull/45

## [v0.10.1](https://github.com/k1LoW/grpcstub/compare/v0.10.0...v0.10.1) - 2023-05-17
- Add `flipflop` service for grpc.health.v1.Health.Watch stubbing by @k1LoW in https://github.com/k1LoW/grpcstub/pull/43

## [v0.10.0](https://github.com/k1LoW/grpcstub/compare/v0.9.0...v0.10.0) - 2023-05-17
- Revert https://github.com/k1LoW/grpcstub/pull/32 by @k1LoW in https://github.com/k1LoW/grpcstub/pull/42

## [v0.9.0](https://github.com/k1LoW/grpcstub/compare/v0.8.1...v0.9.0) - 2023-05-17
- Add EnableHealthCheck option by @k1LoW in https://github.com/k1LoW/grpcstub/pull/39

## [v0.8.1](https://github.com/k1LoW/grpcstub/compare/v0.8.0...v0.8.1) - 2023-03-19
- Allow direct use of ResponseDynamic by @k1LoW in https://github.com/k1LoW/grpcstub/pull/37

## [v0.8.0](https://github.com/k1LoW/grpcstub/compare/v0.7.0...v0.8.0) - 2023-03-19
- Add utility methods that embedds fmt.Sprintf by @k1LoW in https://github.com/k1LoW/grpcstub/pull/28
- Add ImportPath option by @k1LoW in https://github.com/k1LoW/grpcstub/pull/30
- Support dynamic response by @k1LoW in https://github.com/k1LoW/grpcstub/pull/31
- Cast time.Time to timestamppb.Timestamp by @k1LoW in https://github.com/k1LoW/grpcstub/pull/32
- Support ResponseDynamic to `repeated` by @k1LoW in https://github.com/k1LoW/grpcstub/pull/33
- Support ResponseDynamic to `optional` by @k1LoW in https://github.com/k1LoW/grpcstub/pull/34
- Remove methods of Message by @k1LoW in https://github.com/k1LoW/grpcstub/pull/35
- Support custom generator for ResponseDynamic by @k1LoW in https://github.com/k1LoW/grpcstub/pull/36

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
