# IP lifecycle

An `IP` is a namespaced handle to a claimed IP address.
All namespaced resources wanting an IP have to claim the corresponding
`IP` object in order to use it.

## IP management

When creating an `IP`, a vacant IP address has to be allocated.
This allocation is done via cluster-scoped `IPAddress` objects.
An `IPAddress`es name is the IP it represents. As such, detecting whether
an `IPAddress` is taken can be easily done by `Get`ting the `IPAddress`
with the IP to check and inspecting the result: If the `IPAddress` is
present, it means it's taken. Otherwise, at least during the time of
inspection, the `IPAddress` is vacant and ready to be claimed.

The `IPAddress` and `IP` are tied together in the
[`IP`'s store `BeforeCreate` hook](https://github.com/ironcore-dev/ironcore-net/blob/main/internal/registry/ip/storage.go) using
the [`ipaddressallocator`](https://github.com/ironcore-dev/ironcore-net/blob/main/internal/registry/ip/ipaddressallocator/ipaddressallocator.go).

The `Allocator` tries to create `IPAddress`es with the `claimRef` pointing
to the `IP` about to be created. It continues to do so until it either
finds a vacant `IPAddress` (creation succeeds) or it times out after too
many attempts fail (`AlreadyExists` errors).

The valid public `IPAddress` prefixes can be configured using the
`apiserver`s `public-prefix` flag.

When deleting an `IP`, the corresponding `IPAddress` is cleaned up
alongside the claiming `IP`.

## Claiming the IP

To claim an `IP`, claimer has to set the `spec.claimRef` of the `IP`.
Once set, if the claimer wants to release it while it is present, the
claimer has to actively delete the `spec.claimRef`.

If a claimer does not exist anymore, the `IPGarbageCollector` will take
care of releasing the `IP`.

### Claiming an IP for `apinet` objects

For `apinet` objects (`NetworkInterface`, `NATGateway`, `LoadBalancer`),
claiming `IP`s is simple: When creating the object, there usually are
two 'modes' for acquiring an `IP`: Either only the `IPFamily` (if applicable)
is specified, causing a dynamic `IP` object to be created and claimed, or
the desired `IP` is specified, causing the `IP`s in the namespace to be
searched for the `IP` in question and to be claimed if possible.

For the `apinet` objects, this whole process is done directly during
`Create` / `Update`. There is no eventual reconciliation of the IPs.
This also means that if an `Update` causes an `IP` not to be used anymore,
it will be released.

Example network interface claiming a dynamic public IP:

```yaml
apiVersion: core.apinet.ironcore.dev/v1alpha1
kind: NetworkInterface
metadata:
  namespace: default
  name: my-public-nic
spec:
  networkRef:
    name: my-networ
  nodeRef:
    name: my-node
  ips:
  - 192.168.178.3
  publicIPs:
    - name: public-ip-1
      ipFamily: IPv4
```
