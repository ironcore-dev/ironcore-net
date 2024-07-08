# `ironcore` integration

`ironcore-net` controls networking over multiple peers
and intelligently manages functions. It can be operated and
used independently of `ironcore`. The binding to `ironcore`
is only realized via `apinetlet`.

## Mapped objects / interaction

The `apinetlet` is a controller that has access to an `ironcore`-enabled
cluster and an `ironcore-net`-enabled cluster. It maps objects of
`ironcore`'s `networking` group to corresponding entities in
`ironcore-net`, if possible.

### `Network`

When an `networking.ironcore.dev/Network` is created, a corresponding
`core.apinet.ironcore.dev/Network` is created in the `apinet` cluster.
The name of the `Network` in the `apinet` cluster is the `uid` of the
`Network` in the `ironcore` cluster.

Once created and with an allocated `ID`, the `ironcore` `Network` will
be patched with the corresponding provider ID of the `apinet` `Network` and
set to `state: Available`.
The provider ID format & parsing can be found in [`provider.go`](../../apinetlet/provider/provider.go).

If `ironcore` `Network` has peerings, then they will be translated and 
patched into `apinet` `Network`. More details can be found here 
[`Network Peering`](./network-lifecycle.md#network-peering)

### `LoadBalancer`

For a `networking.ironcore.dev/LoadBalancer` a corresponding
`core.apinet.ironcore.dev/LoadBalancer` is created, also having the
`uid` of the source object as its name.

The `apinet` `LoadBalancer` is configured to have an IP per IP family
if it's a `Public` load balancer. Otherwise, it simply uses the IPs
specified via the `ironcore` `LoadBalancer`.

For the routing targets, the `ironcore` `LoadBalancerRouting` is
inspected and transformed into an `apinet` `LoadBalancerRouting`.

For its instances, the `apinet` `LoadBalancer` is created with a `template`
that specifies instance anti-affinity to ensure instances are distributed
cross-zone.

### `NetworkPolicy`

For a `networking.ironcore.dev/NetworkPolicy` a corresponding
`core.apinet.ironcore.dev/NetworkPolicy` is created in the `apinet` cluster.
The name of the `NetworkPolicy` in the `apinet` cluster is the `uid` of the
`NetworkPolicy` in the `ironcore` cluster.
The `NetworkPolicy` applies to `NetworkInterfaces` within a specific `Network`, filtered by the label specified in the `NetworkPolicy` spec.

Based on the `PolicyTypes` (`egress` and/or `ingress`), rules can be specified to limit the traffic on the target object to and from various objects like `LoadBalancer`, `NetworkInterface` or `IPBlock` on certain `ports`.

When a `NetworkPolicy` is applied, a `NetworkPolicyRule` object is created with the specified policy rules. `Metalnetlet` then reads the `NetworkPolicyRule` and enforces these policy (firewall) rules on the target `NetworkInterface`s.

for example refer to [NetworkPolicy object](./objects.md#networkpolicy)


### `NATGateway`

For a `networking.ironcore.dev/NATGateway` a corresponding
`core.apinet.ironcore.dev/NATGateway` is created, also having the
`uid` of the source object as its name. Additionally, a
`NATGatewayAutoscaler` is created, ensuring there are enough public
IPs available.

The `apinet` `NATGateway` will try to target all `NetworkInterface`s
in its `Network` that share an `IPFamily` but don't have a public
IP for that family and no other `NATGateway` claiming it.

### `NetworkInterface`

Since the location of an `apinet` `NetworkInterface` depends on an
`apinet` `Node`, `apinetlet` can *not* create a mapping `apinet`
`NetworkInterface` for an `ironcore` `NetworkInterface`. Instead,
the `MachinePool` implementing entity is responsible of doing so.

The desired flow here is for the `MachinePool` implementor to create
an `apinet` `NetworkInterface` for each desired `ironcore`
`NetworkInterface` a `Machine` specifies. Then, upon successful creation,
the `MachinePool` implementor has to patch the `ironcore`'s
`NetworkInterface` `spec.providerID` to the provider ID of the
`apinet` `NetworkInterface` (again, see
[`provider.go`](../../apinetlet/provider/provider.go) on how to obtain
/ format the provider ID correctly).

Once the `providerID` of the `ironcore` `NetworkInterface` is set,
`apinetlet` takes care reporting the status of the `ironcore`
`NetworkInterface` by observing the matching `apinet` `NetworkInterface`.
`apinetlet` then also applies requested `VirtualIP`s and `LoadBalancer`
targets to the `apinet` `NetworkInterface`.
