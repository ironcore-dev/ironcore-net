# onmetal-api-net
[![Pull Request Code test](https://github.com/onmetal/onmetal-api-net/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/onmetal/onmetal-api-net/actions/workflows/test.yml)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](https://makeapullrequest.com)
[![GitHub License](https://img.shields.io/static/v1?label=License&message=Apache-2.0&color=blue&style=flat-square)](LICENSE)

## Overview

`onmetal-api-net` provides networking functions across multiple
peers.

`onmetal-api-net` conceptually consists of a control-plane and
`Node`s. The API of `onmetal-api-net` is realized by an aggregated API
server. The `controller-manager` reconciles state of these objects.
The `scheduler` (currently built into the `controller-manager`)
assigns functions to `Node`s.

A `Node` is currently implemented via `metalnetlet`, an agent
using a `metalnet` cluster run the payload functions on. A
`metalnetlet` creates `Node` objects corresponding to all
`Node`s inside the `metalnet` custer.

The integration to `onmetal-api` is realized via the `apinetlet`,
an agent using an `onmetal-api-net` cluster to realize `onmetal-api`
objects like `LoadBalancer`s, `VirtualIP`s and more.

## Contributing

We'd love to get feedback from you. Please report bugs, suggestions or post questions by opening a GitHub issue.

## License

[Apache-2.0](LICENSE)
