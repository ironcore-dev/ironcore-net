# Network Policy
When a `networking.ironcore.dev/NetworkPolicy` is created, a corresponding `core.apinet.ironcore.dev/NetworkPolicy` is created in the `apinet` cluster. The name of the `NetworkPolicy` in the `apinet` cluster is the uid of the `NetworkPolicy` in the `ironcore` cluster. Implement `NetworkPolicy` translation logic within the `apinetlet` component to ensure seamless interoperability between the `networking.ironcore.dev` and `core.apinet.ironcore.dev` groups.  

sample yaml:
```yaml
apiVersion: core.apinet.ironcore.dev/v1alpha1
kind: NetworkPolicy
metadata:
  namespace: default
  # networkpolicy-uid is the uid of corresponding networkpolicy in ironcore cluster
  name: networpolicy-uid
spec:
  # This specifies the target network to limit the traffic in.
  networkRef:
  # network-uid is the ironcore-net cluster's network name which refers to corresponding network's uid in ironcore cluster
    name: network-uid
  # Only network interfaces in the specified network will be selected.
  networkInterfaceSelector:
    matchLabels:
      app: db
  # If the policy types are not specified, they are inferred on whether
  # any ingress / egress rule exists. If no ingress / egress rule exists,
  # the network policy is denied on admission.
  policyTypes:
  - Ingress
  - Egress
  # Multiple ingress / egress rules are possible.
  ingress:
  - from:
    # Traffic can be limited from a source IP block.
    - ipBlock:
        cidr: 172.17.0.0/16
    # Traffic can also be limited to objects of the networking api.
    # For instance, to limit traffic from network interfaces, one could
    # specify the following:
    - objectSelector:
        kind: NetworkInterface
        matchLabels:
          app: web
    # Analogous to network interfaces, it is also possible to limit
    # traffic coming from load balancers:
    - objectSelector:
        kind: LoadBalancer
        matchLabels:
          app: web
    # Ports always have to be specified. Only traffic matching the ports
    # will be allowed.
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
