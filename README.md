# ironcore-net

[![REUSE status](https://api.reuse.software/badge/github.com/ironcore-dev/ironcore-net)](https://api.reuse.software/info/github.com/ironcore-dev/ironcore-net)
[![Pull Request Code test](https://github.com/ironcore-dev/ironcore-net/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/ironcore-dev/ironcore-net/actions/workflows/test.yml)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://makeapullrequest.com)
[![GitHub License](https://img.shields.io/static/v1?label=License&message=Apache-2.0&color=blue)](LICENSE)

## Overview

`ironcore-net` provides networking functions across multiple
peers.

`ironcore-net` conceptually consists of a control-plane and
`Node`s. The API of `ironcore-net` is realized by an aggregated API
server. The `controller-manager` reconciles state of these objects.
The `scheduler` (currently built into the `controller-manager`)
assigns functions to `Node`s.

A `Node` is currently implemented via `metalnetlet`, an agent
using a `metalnet` cluster run the payload functions on. A
`metalnetlet` creates `Node` objects corresponding to all
`Node`s inside the `metalnet` custer.

The integration to `ironcore` is realized via the `apinetlet`,
an agent using an `ironcore-net` cluster to realize `ironcore`
objects like `LoadBalancer`s, `VirtualIP`s and more.

Documentation about the concepts of `ironcore-net` can be found
in the [`concepts` directory](docs/concepts).

## Contributing

We'd love to get feedback from you. Please report bugs, suggestions or post questions by opening a GitHub issue.

## License

[Apache-2.0](LICENSE)
