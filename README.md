# ironcore-net

[![REUSE status](https://api.reuse.software/badge/github.com/ironcore-dev/ironcore-net)](https://api.reuse.software/info/github.com/ironcore-dev/ironcore-net)
[![Go Report Card](https://goreportcard.com/badge/github.com/ironcore-dev/ironcore-net)](https://goreportcard.com/report/github.com/ironcore-dev/ironcore-net)
[![GitHub License](https://img.shields.io/static/v1?label=License&message=Apache-2.0&color=blue)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://makeapullrequest.com)

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

## Licensing

Copyright 2025 SAP SE or an SAP affiliate company and IronCore contributors. Please see our [LICENSE](LICENSE) for
copyright and license information. Detailed information including third-party components and their licensing/copyright
information is available [via the REUSE tool](https://api.reuse.software/info/github.com/ironcore-dev/ironcore-net).

<p align="center"><img alt="Bundesministerium für Wirtschaft und Energie (BMWE)-EU funding logo" src="https://apeirora.eu/assets/img/BMWK-EU.png" width="400"/></p>
