# Objects

`ironcore-net` provides multiple objects to interact with.
As `ironcore-net` is a Kubernetes-API, all objects are written
in a declarative fashion, meaning that they represent the desired
state and will be reconciled to eventually manifest that state in
the real world.

## IP

An IP can be used to get a static hold of an IP. Currently,
only public IPs can be obtained this way.

Upon its creation, a public IP object gets assigned an available
IP. When deleting the IP, the corresponding public IP is released
again.

```yaml
apiVersion: core.apinet.ironcore.dev/v1alpha1
kind: IP
metadata:
  namespace: default
  name: my-public-ip
spec:
  type: Public
  ip: 10.0.0.1 # This is allocated automatically.
  # claimRef: # claimRef is set as soon as the IP is claimed.
  #   name: my-nic
```

## `Network`

To set up a networking infrastructure, the primary object to create
is a `Network`. A `Network` is an isolated networking domain.
Communication within a `Network` happens on Layer 3 (no ethernet) via
the IP protocol. Peers inside a `Network` can reach each other unless
configured otherwise.

Example manifest:

```yaml
apiVersion: core.apinet.ironcore.dev/v1alpha1
kind: Network
metadata:
  namespace: default
  name: my-network
# spec:
#   id: "301"
```

When creating a `Network`, its `spec.id` is automatically allocated.

## `NetworkInterface`

A `NetworkInterface` is the 'default' peer inside a `Network`. To
create a `NetworkInterface`, the target `Node` and primary internal
IPs have to be known in advance. Once created, a `NetworkInterface`
can also dynamically claim and release `publicIPs` and be target of
a `NATGateway` (via `spec.natGateways`).

There are two ways to use `publicIPs`: Either the public IP literal
is specified upon creation, which causes the namespace of the
`NetworkInterface` to be searched for the corresponding `IP` object
to be claimed. If no literal is specified, a dynamic `IP` object
will be created that will have a controller reference set to the
`NetworkInterface`, causing it to be deleted when the `NetworkInterface`
is deleted.

Once picked up by the target `Node`, the network interface is created
and reports its PCI address and state as part of its `status`.

Example manifest:

```yaml
apiVersion: core.apinet.ironcore.dev/v1alpha1
kind: NetworkInterface
metadata:
  namespace: default
  name: my-nic
spec:
  networkRef:
    name: my-network
  nodeRef:
    name: my-node
  ips:
  - 192.168.178.1
  publicIPs:
  - name: ip-1
    ip: 10.0.0.1
status:
  pciAddress:
    bus: "06"
    domain: "0000"
    function: "3"
    slot: "00"
  state: Ready
```

## `Instance`

An `Instance` allows deploying dynamic network functions onto `Node`s
inside the cluster. Currently, only `Instance`s of `type: LoadBalancer`
are available.

To create a load balancer `Instance`, the load balancer type, the
IPs and the network has to be specified.

If the `nodeRef` field is empty, the `scheduler` automatically
determines a suitable `Node` for the `Instance` to run on. Scheduling
of `Instance`s can be influenced by using `spec.affinity`, allowing
for node-affinity and instance anti-affinity. This is especially
useful when deploying load balancer instances when there should only
be a single instance per topology domain.

Example manifest:

```yaml
apiVersion: core.apinet.ironcore.dev/v1alpha1
kind: Instance
metadata:
  namespace: default
  name: my-instance
spec:
  type: LoadBalancer
  loadBalancerType: Public
  networkRef:
    name: my-network
  ips:
  - 10.0.0.2
```

## `LoadBalancer`

A `LoadBalancer` manages its `Instance`s and declares its routing
by its corresponding `LoadBalancerRouting`. Under the hood, a
`LoadBalancer` creates a `DaemonSet` managing its `Instance`s.
Currently, `Instance`s can contain multiple IPs, but the desired
architecture is to have an `Instance` containing only a single IP.
This will eventually make the management of multiple `DaemonSet`s
per `LoadBalancer` a requirement.

For now, everytime the IPs of a `LoadBalancer` are updated,
all its `Instance`s are updated (done by the `DaemonSet` controller).

```yaml
apiVersion: core.apinet.ironcore.dev/v1alpha1
kind: LoadBalancer
metadata:
  namespace: default
  name: my-instance
spec:
  type: Public
  networkRef:
    name: my-network
  ips:
  - name: ip-1
    ip: 10.0.0.2
  template: {}
```

### `NetworkPolicy`

A `NetworkPolicy` limits traffic to and from various objects like `NetworkInterfaces`, `LoadBalancers` etc. for the target objects within a specific network. 

When a `NetworkPolicy` is applied, a `NetworkPolicyRule` object is created to contain the policy rules specified in the `NetworkPolicy`. 

Then, `metalnetlet` translates these policy rules from the `NetworkPolicyRule` object and applies them to the `NetworkInterface`s.

Example manifest
```yaml
apiVersion: core.apinet.ironcore.dev/v1alpha1
kind: NetworkPolicy
metadata:
  namespace: default
  name: my-networkpolicy
spec:
  networkRef:
    name: my-network
  networkInterfaceSelector:
    matchLabels:
      app: db
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - ipBlock:
        cidr: 172.17.0.0/16
    - objectSelector:
        kind: NetworkInterface
        matchLabels:
    - objectSelector:
        kind: LoadBalancer
        matchLabels:
          app: web
    ports:
    - protocol: TCP
      port: 5432
  egress:
  - to:
    - ipBlock:
        cidr: 10.0.0.0/24
    ports:
    - protocol: TCP
      port: 8080
```

## `NATGateway`

A `NATGateway` allows NAT-ing external IPs to multiple target
`NetworkInterface`s inside a network. The NATed IPs are managed
using a `NATTable` the `NATGateway` controller updates depending
on the amount of target `NetworkInterface`s.

A `NATGateway` always tries to claim all `NetworkInterface`s inside
its network that don't have a public IP of the IP family the `NATGateway`
has. The claim is depicted by the `NetworkInterface`'s `spec.natGateways`.

Example manifest:

```yaml
apiVersion: core.apinet.ironcore.dev/v1alpha1
kind: NATGateway
metadata:
  namespace: default
  name: my-nat-gateway
spec:
  networkRef:
    name: my-network
  ipFamily: IPv4
  ips:
  - name: ip-1
    ip: 10.0.0.3
  portsPerNetworkInterface: 1024
```
