<p>Packages:</p>
<ul>
<li>
<a href="#core.apinet.ironcore.dev%2fv1alpha1">core.apinet.ironcore.dev/v1alpha1</a>
</li>
</ul>
<h2 id="core.apinet.ironcore.dev/v1alpha1">core.apinet.ironcore.dev/v1alpha1</h2>
<div>
<p>Package v1alpha1 is the v1alpha1 version of the API.</p>
</div>
Resource Types:
<ul><li>
<a href="#core.apinet.ironcore.dev/v1alpha1.DaemonSet">DaemonSet</a>
</li><li>
<a href="#core.apinet.ironcore.dev/v1alpha1.IP">IP</a>
</li><li>
<a href="#core.apinet.ironcore.dev/v1alpha1.IPAddress">IPAddress</a>
</li><li>
<a href="#core.apinet.ironcore.dev/v1alpha1.Instance">Instance</a>
</li><li>
<a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancer">LoadBalancer</a>
</li><li>
<a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancerRouting">LoadBalancerRouting</a>
</li><li>
<a href="#core.apinet.ironcore.dev/v1alpha1.NATGateway">NATGateway</a>
</li><li>
<a href="#core.apinet.ironcore.dev/v1alpha1.NATGatewayAutoscaler">NATGatewayAutoscaler</a>
</li><li>
<a href="#core.apinet.ironcore.dev/v1alpha1.NATTable">NATTable</a>
</li><li>
<a href="#core.apinet.ironcore.dev/v1alpha1.Network">Network</a>
</li><li>
<a href="#core.apinet.ironcore.dev/v1alpha1.NetworkID">NetworkID</a>
</li><li>
<a href="#core.apinet.ironcore.dev/v1alpha1.NetworkInterface">NetworkInterface</a>
</li><li>
<a href="#core.apinet.ironcore.dev/v1alpha1.Node">Node</a>
</li></ul>
<h3 id="core.apinet.ironcore.dev/v1alpha1.DaemonSet">DaemonSet
</h3>
<div>
<p>DaemonSet is the schema for the daemonsets API.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
core.apinet.ironcore.dev/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>DaemonSet</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.DaemonSetSpec">
DaemonSetSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>nodeSelector</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
<p>Selector selects all Instance that are managed by this daemon set.</p>
</td>
</tr>
<tr>
<td>
<code>template</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.InstanceTemplate">
InstanceTemplate
</a>
</em>
</td>
<td>
<p>Template is the instance template.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.DaemonSetStatus">
DaemonSetStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.IP">IP
</h3>
<div>
<p>IP is the schema for the ips API.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
core.apinet.ironcore.dev/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>IP</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.IPSpec">
IPSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>type</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.IPType">
IPType
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>ipFamily</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#ipfamily-v1-core">
Kubernetes core/v1.IPFamily
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>ip</code><br/>
<em>
<a href="../api/#api.ironcore.dev/net.IP">
github.com/ironcore-dev/ironcore-net/apimachinery/api/net.IP
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>claimRef</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.IPClaimRef">
IPClaimRef
</a>
</em>
</td>
<td>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.IPStatus">
IPStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.IPAddress">IPAddress
</h3>
<div>
<p>IPAddress is the schema for the ipaddresses API.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
core.apinet.ironcore.dev/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>IPAddress</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.IPAddressSpec">
IPAddressSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>ip</code><br/>
<em>
<a href="../api/#api.ironcore.dev/net.IP">
github.com/ironcore-dev/ironcore-net/apimachinery/api/net.IP
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>claimRef</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.IPAddressClaimRef">
IPAddressClaimRef
</a>
</em>
</td>
<td>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.Instance">Instance
</h3>
<div>
<p>Instance is the schema for the instances API.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
core.apinet.ironcore.dev/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>Instance</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.InstanceSpec">
InstanceSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>type</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.InstanceType">
InstanceType
</a>
</em>
</td>
<td>
<p>Type specifies the InstanceType to deploy.</p>
</td>
</tr>
<tr>
<td>
<code>loadBalancerType</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancerType">
LoadBalancerType
</a>
</em>
</td>
<td>
<p>LoadBalancerType is the load balancer type this instance is for.</p>
</td>
</tr>
<tr>
<td>
<code>networkRef</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#localobjectreference-v1-core">
Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<p>NetworkRef references the network the instance is on.</p>
</td>
</tr>
<tr>
<td>
<code>ips</code><br/>
<em>
<a href="../api/#api.ironcore.dev/net.IP">
[]github.com/ironcore-dev/ironcore-net/apimachinery/api/net.IP
</a>
</em>
</td>
<td>
<p>IPs are the IPs of the instance.</p>
</td>
</tr>
<tr>
<td>
<code>loadBalancerPorts</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancerPort">
[]LoadBalancerPort
</a>
</em>
</td>
<td>
<p>LoadBalancerPorts are the load balancer ports of this instance.</p>
</td>
</tr>
<tr>
<td>
<code>affinity</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.Affinity">
Affinity
</a>
</em>
</td>
<td>
<p>Affinity are affinity constraints.</p>
</td>
</tr>
<tr>
<td>
<code>topologySpreadConstraints</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.TopologySpreadConstraint">
[]TopologySpreadConstraint
</a>
</em>
</td>
<td>
<p>TopologySpreadConstraints describes how a group of instances ought to spread across topology
domains. Scheduler will schedule instances in a way which abides by the constraints.
All topologySpreadConstraints are ANDed.</p>
</td>
</tr>
<tr>
<td>
<code>nodeRef</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#localobjectreference-v1-core">
Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<p>NodeRef references the node hosting the load balancer instance.
Will be set by the scheduler if empty.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.InstanceStatus">
InstanceStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.LoadBalancer">LoadBalancer
</h3>
<div>
<p>LoadBalancer is the schema for the loadbalancers API.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
core.apinet.ironcore.dev/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>LoadBalancer</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancerSpec">
LoadBalancerSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>type</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancerType">
LoadBalancerType
</a>
</em>
</td>
<td>
<p>Type specifies the type of load balancer.</p>
</td>
</tr>
<tr>
<td>
<code>networkRef</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#localobjectreference-v1-core">
Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<p>NetworkRef references the network the load balancer is part of.</p>
</td>
</tr>
<tr>
<td>
<code>ips</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancerIP">
[]LoadBalancerIP
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>IPs specifies the IPs of the load balancer.</p>
</td>
</tr>
<tr>
<td>
<code>ports</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancerPort">
[]LoadBalancerPort
</a>
</em>
</td>
<td>
<p>Ports are the ports the load balancer should allow.
If empty, the load balancer allows all ports.</p>
</td>
</tr>
<tr>
<td>
<code>selector</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
<p>Selector selects all Instance that are managed by this daemon set.</p>
</td>
</tr>
<tr>
<td>
<code>template</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.InstanceTemplate">
InstanceTemplate
</a>
</em>
</td>
<td>
<p>Template is the instance template.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancerStatus">
LoadBalancerStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.LoadBalancerRouting">LoadBalancerRouting
</h3>
<div>
<p>LoadBalancerRouting is the schema for the loadbalancerroutings API.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
core.apinet.ironcore.dev/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>LoadBalancerRouting</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>destinations</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancerDestination">
[]LoadBalancerDestination
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NATGateway">NATGateway
</h3>
<div>
<p>NATGateway is the schema for the natgateways API.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
core.apinet.ironcore.dev/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>NATGateway</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NATGatewaySpec">
NATGatewaySpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>ipFamily</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#ipfamily-v1-core">
Kubernetes core/v1.IPFamily
</a>
</em>
</td>
<td>
<p>IPFamily is the IP family of the NAT gateway.</p>
</td>
</tr>
<tr>
<td>
<code>networkRef</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#localobjectreference-v1-core">
Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<p>NetworkRef references the network the NAT gateway is part of.</p>
</td>
</tr>
<tr>
<td>
<code>ips</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NATGatewayIP">
[]NATGatewayIP
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>IPs specifies the IPs of the NAT gateway.</p>
</td>
</tr>
<tr>
<td>
<code>portsPerNetworkInterface</code><br/>
<em>
int32
</em>
</td>
<td>
<p>PortsPerNetworkInterface specifies how many ports to allocate per network interface.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NATGatewayStatus">
NATGatewayStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NATGatewayAutoscaler">NATGatewayAutoscaler
</h3>
<div>
<p>NATGatewayAutoscaler is the schema for the natgatewayautoscalers API.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
core.apinet.ironcore.dev/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>NATGatewayAutoscaler</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NATGatewayAutoscalerSpec">
NATGatewayAutoscalerSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>natGatewayRef</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#localobjectreference-v1-core">
Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<p>NATGatewayRef points to the target NATGateway to scale.</p>
</td>
</tr>
<tr>
<td>
<code>minPublicIPs</code><br/>
<em>
int32
</em>
</td>
<td>
<p>MinPublicIPs is the minimum number of public IPs to allocate for a NAT Gateway.</p>
</td>
</tr>
<tr>
<td>
<code>maxPublicIPs</code><br/>
<em>
int32
</em>
</td>
<td>
<p>MaxPublicIPs is the maximum number of public IPs to allocate for a NAT Gateway.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NATGatewayAutoscalerStatus">
NATGatewayAutoscalerStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NATTable">NATTable
</h3>
<div>
<p>NATTable is the schema for the nattables API.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
core.apinet.ironcore.dev/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>NATTable</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>ips</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NATIP">
[]NATIP
</a>
</em>
</td>
<td>
<p>IPs specifies how to NAT the IPs for the NAT gateway.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.Network">Network
</h3>
<div>
<p>Network is the schema for the networks API.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
core.apinet.ironcore.dev/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>Network</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NetworkSpec">
NetworkSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>id</code><br/>
<em>
string
</em>
</td>
<td>
<p>ID is the ID of the network.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NetworkStatus">
NetworkStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NetworkID">NetworkID
</h3>
<div>
<p>NetworkID is the schema for the networkids API.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
core.apinet.ironcore.dev/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>NetworkID</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NetworkIDSpec">
NetworkIDSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>claimRef</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NetworkIDClaimRef">
NetworkIDClaimRef
</a>
</em>
</td>
<td>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NetworkInterface">NetworkInterface
</h3>
<div>
<p>NetworkInterface is the schema for the networkinterfaces API.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
core.apinet.ironcore.dev/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>NetworkInterface</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NetworkInterfaceSpec">
NetworkInterfaceSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>nodeRef</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#localobjectreference-v1-core">
Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<p>NodeRef is the node the network interface is hosted on.</p>
</td>
</tr>
<tr>
<td>
<code>networkRef</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#localobjectreference-v1-core">
Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<p>NetworkRef references the network that the network interface is in.</p>
</td>
</tr>
<tr>
<td>
<code>ips</code><br/>
<em>
<a href="../api/#api.ironcore.dev/net.IP">
[]github.com/ironcore-dev/ironcore-net/apimachinery/api/net.IP
</a>
</em>
</td>
<td>
<p>IPs are the internal IPs of the network interface.</p>
</td>
</tr>
<tr>
<td>
<code>prefixes</code><br/>
<em>
<a href="../api/#api.ironcore.dev/net.IPPrefix">
[]github.com/ironcore-dev/ironcore-net/apimachinery/api/net.IPPrefix
</a>
</em>
</td>
<td>
<p>Prefixes are additional prefixes to route to the network interface.</p>
</td>
</tr>
<tr>
<td>
<code>natGateways</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NetworkInterfaceNAT">
[]NetworkInterfaceNAT
</a>
</em>
</td>
<td>
<p>NATs specify the NAT of the network interface IP family.
Can only be set if there is no matching IP family in PublicIPs.</p>
</td>
</tr>
<tr>
<td>
<code>publicIPs</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NetworkInterfacePublicIP">
[]NetworkInterfacePublicIP
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PublicIPs are the public IPs the network interface should have.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NetworkInterfaceStatus">
NetworkInterfaceStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.Node">Node
</h3>
<div>
<p>Node is the schema for the nodes API.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code><br/>
string</td>
<td>
<code>
core.apinet.ironcore.dev/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code><br/>
string
</td>
<td><code>Node</code></td>
</tr>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NodeSpec">
NodeSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NodeStatus">
NodeStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.Affinity">Affinity
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.InstanceSpec">InstanceSpec</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>nodeAffinity</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NodeAffinity">
NodeAffinity
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>instanceAntiAffinity</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.InstanceAntiAffinity">
InstanceAntiAffinity
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.DaemonSetSpec">DaemonSetSpec
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.DaemonSet">DaemonSet</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>nodeSelector</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
<p>Selector selects all Instance that are managed by this daemon set.</p>
</td>
</tr>
<tr>
<td>
<code>template</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.InstanceTemplate">
InstanceTemplate
</a>
</em>
</td>
<td>
<p>Template is the instance template.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.DaemonSetStatus">DaemonSetStatus
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.DaemonSet">DaemonSet</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>collisionCount</code><br/>
<em>
int32
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.IPAddressClaimRef">IPAddressClaimRef
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.IPAddressSpec">IPAddressSpec</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>group</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>resource</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>namespace</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>uid</code><br/>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/types#UID">
k8s.io/apimachinery/pkg/types.UID
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.IPAddressSpec">IPAddressSpec
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.IPAddress">IPAddress</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ip</code><br/>
<em>
<a href="../api/#api.ironcore.dev/net.IP">
github.com/ironcore-dev/ironcore-net/apimachinery/api/net.IP
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>claimRef</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.IPAddressClaimRef">
IPAddressClaimRef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.IPClaimRef">IPClaimRef
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.IPSpec">IPSpec</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>group</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>resource</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>uid</code><br/>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/types#UID">
k8s.io/apimachinery/pkg/types.UID
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.IPSpec">IPSpec
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.IP">IP</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.IPType">
IPType
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>ipFamily</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#ipfamily-v1-core">
Kubernetes core/v1.IPFamily
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>ip</code><br/>
<em>
<a href="../api/#api.ironcore.dev/net.IP">
github.com/ironcore-dev/ironcore-net/apimachinery/api/net.IP
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>claimRef</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.IPClaimRef">
IPClaimRef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.IPStatus">IPStatus
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.IP">IP</a>)
</p>
<div>
</div>
<h3 id="core.apinet.ironcore.dev/v1alpha1.IPType">IPType
(<code>string</code> alias)</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.IPSpec">IPSpec</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Public&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.InstanceAffinityTerm">InstanceAffinityTerm
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.InstanceAntiAffinity">InstanceAntiAffinity</a>)
</p>
<div>
<p>InstanceAffinityTerm defines a set of instances (namely those matching the labelSelector that this instance should be
co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose
value of the label with key <topologyKey> matches that of any node on which a instance of the set of instances is running.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>labelSelector</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
<p>LabelSelector over a set of resources, in this case instances.</p>
</td>
</tr>
<tr>
<td>
<code>topologyKey</code><br/>
<em>
string
</em>
</td>
<td>
<p>TopologyKey indicates that this instance should be co-located (affinity) or not co-located (anti-affinity)
with the instances matching the labelSelector, where co-located is defined as running on a
node whose value of the label with key topologyKey matches that of any node on which any of the
selected instances is running.
Empty topologyKey is not allowed.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.InstanceAntiAffinity">InstanceAntiAffinity
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.Affinity">Affinity</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>requiredDuringSchedulingIgnoredDuringExecution</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.InstanceAffinityTerm">
[]InstanceAffinityTerm
</a>
</em>
</td>
<td>
<p>RequiredDuringSchedulingIgnoredDuringExecution specifies anti-affinity requirements at
scheduling time, that, if not met, will cause the instance not be scheduled onto the node.
When there are multiple elements, the lists of nodes corresponding to each
instanceAffinityTerm are intersected, i.e. all terms must be satisfied.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.InstanceSpec">InstanceSpec
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.Instance">Instance</a>, <a href="#core.apinet.ironcore.dev/v1alpha1.InstanceTemplate">InstanceTemplate</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.InstanceType">
InstanceType
</a>
</em>
</td>
<td>
<p>Type specifies the InstanceType to deploy.</p>
</td>
</tr>
<tr>
<td>
<code>loadBalancerType</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancerType">
LoadBalancerType
</a>
</em>
</td>
<td>
<p>LoadBalancerType is the load balancer type this instance is for.</p>
</td>
</tr>
<tr>
<td>
<code>networkRef</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#localobjectreference-v1-core">
Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<p>NetworkRef references the network the instance is on.</p>
</td>
</tr>
<tr>
<td>
<code>ips</code><br/>
<em>
<a href="../api/#api.ironcore.dev/net.IP">
[]github.com/ironcore-dev/ironcore-net/apimachinery/api/net.IP
</a>
</em>
</td>
<td>
<p>IPs are the IPs of the instance.</p>
</td>
</tr>
<tr>
<td>
<code>loadBalancerPorts</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancerPort">
[]LoadBalancerPort
</a>
</em>
</td>
<td>
<p>LoadBalancerPorts are the load balancer ports of this instance.</p>
</td>
</tr>
<tr>
<td>
<code>affinity</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.Affinity">
Affinity
</a>
</em>
</td>
<td>
<p>Affinity are affinity constraints.</p>
</td>
</tr>
<tr>
<td>
<code>topologySpreadConstraints</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.TopologySpreadConstraint">
[]TopologySpreadConstraint
</a>
</em>
</td>
<td>
<p>TopologySpreadConstraints describes how a group of instances ought to spread across topology
domains. Scheduler will schedule instances in a way which abides by the constraints.
All topologySpreadConstraints are ANDed.</p>
</td>
</tr>
<tr>
<td>
<code>nodeRef</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#localobjectreference-v1-core">
Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<p>NodeRef references the node hosting the load balancer instance.
Will be set by the scheduler if empty.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.InstanceStatus">InstanceStatus
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.Instance">Instance</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ips</code><br/>
<em>
<a href="../api/#api.ironcore.dev/net.IP">
[]github.com/ironcore-dev/ironcore-net/apimachinery/api/net.IP
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>collisionCount</code><br/>
<em>
int32
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.InstanceTemplate">InstanceTemplate
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.DaemonSetSpec">DaemonSetSpec</a>, <a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancerSpec">LoadBalancerSpec</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>metadata</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.InstanceSpec">
InstanceSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>type</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.InstanceType">
InstanceType
</a>
</em>
</td>
<td>
<p>Type specifies the InstanceType to deploy.</p>
</td>
</tr>
<tr>
<td>
<code>loadBalancerType</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancerType">
LoadBalancerType
</a>
</em>
</td>
<td>
<p>LoadBalancerType is the load balancer type this instance is for.</p>
</td>
</tr>
<tr>
<td>
<code>networkRef</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#localobjectreference-v1-core">
Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<p>NetworkRef references the network the instance is on.</p>
</td>
</tr>
<tr>
<td>
<code>ips</code><br/>
<em>
<a href="../api/#api.ironcore.dev/net.IP">
[]github.com/ironcore-dev/ironcore-net/apimachinery/api/net.IP
</a>
</em>
</td>
<td>
<p>IPs are the IPs of the instance.</p>
</td>
</tr>
<tr>
<td>
<code>loadBalancerPorts</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancerPort">
[]LoadBalancerPort
</a>
</em>
</td>
<td>
<p>LoadBalancerPorts are the load balancer ports of this instance.</p>
</td>
</tr>
<tr>
<td>
<code>affinity</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.Affinity">
Affinity
</a>
</em>
</td>
<td>
<p>Affinity are affinity constraints.</p>
</td>
</tr>
<tr>
<td>
<code>topologySpreadConstraints</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.TopologySpreadConstraint">
[]TopologySpreadConstraint
</a>
</em>
</td>
<td>
<p>TopologySpreadConstraints describes how a group of instances ought to spread across topology
domains. Scheduler will schedule instances in a way which abides by the constraints.
All topologySpreadConstraints are ANDed.</p>
</td>
</tr>
<tr>
<td>
<code>nodeRef</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#localobjectreference-v1-core">
Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<p>NodeRef references the node hosting the load balancer instance.
Will be set by the scheduler if empty.</p>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.InstanceType">InstanceType
(<code>string</code> alias)</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.InstanceSpec">InstanceSpec</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;LoadBalancer&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.LoadBalancerDestination">LoadBalancerDestination
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancerRouting">LoadBalancerRouting</a>)
</p>
<div>
<p>LoadBalancerDestination is the destination of the load balancer.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ip</code><br/>
<em>
<a href="../api/#api.ironcore.dev/net.IP">
github.com/ironcore-dev/ironcore-net/apimachinery/api/net.IP
</a>
</em>
</td>
<td>
<p>IP is the target IP.</p>
</td>
</tr>
<tr>
<td>
<code>targetRef</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancerTargetRef">
LoadBalancerTargetRef
</a>
</em>
</td>
<td>
<p>TargetRef is the target providing the destination.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.LoadBalancerIP">LoadBalancerIP
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancerSpec">LoadBalancerSpec</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the load balancer IP.</p>
</td>
</tr>
<tr>
<td>
<code>ipFamily</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#ipfamily-v1-core">
Kubernetes core/v1.IPFamily
</a>
</em>
</td>
<td>
<p>IPFamily is the IP family of the IP. Has to match IP if specified. If unspecified and IP is specified,
will be defaulted by using the IP family of IP.
If only IPFamily is specified, a random IP of that family will be allocated if possible.</p>
</td>
</tr>
<tr>
<td>
<code>ip</code><br/>
<em>
<a href="../api/#api.ironcore.dev/net.IP">
github.com/ironcore-dev/ironcore-net/apimachinery/api/net.IP
</a>
</em>
</td>
<td>
<p>IP specifies a specific IP to allocate. If empty, a random IP will be allocated if possible.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.LoadBalancerPort">LoadBalancerPort
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.InstanceSpec">InstanceSpec</a>, <a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancerSpec">LoadBalancerSpec</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>protocol</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#protocol-v1-core">
Kubernetes core/v1.Protocol
</a>
</em>
</td>
<td>
<p>Protocol is the protocol the load balancer should allow.
If not specified, defaults to TCP.</p>
</td>
</tr>
<tr>
<td>
<code>port</code><br/>
<em>
int32
</em>
</td>
<td>
<p>Port is the port to allow.</p>
</td>
</tr>
<tr>
<td>
<code>endPort</code><br/>
<em>
int32
</em>
</td>
<td>
<p>EndPort marks the end of the port range to allow.
If unspecified, only a single port, Port, will be allowed.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.LoadBalancerSpec">LoadBalancerSpec
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancer">LoadBalancer</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancerType">
LoadBalancerType
</a>
</em>
</td>
<td>
<p>Type specifies the type of load balancer.</p>
</td>
</tr>
<tr>
<td>
<code>networkRef</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#localobjectreference-v1-core">
Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<p>NetworkRef references the network the load balancer is part of.</p>
</td>
</tr>
<tr>
<td>
<code>ips</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancerIP">
[]LoadBalancerIP
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>IPs specifies the IPs of the load balancer.</p>
</td>
</tr>
<tr>
<td>
<code>ports</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancerPort">
[]LoadBalancerPort
</a>
</em>
</td>
<td>
<p>Ports are the ports the load balancer should allow.
If empty, the load balancer allows all ports.</p>
</td>
</tr>
<tr>
<td>
<code>selector</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
<p>Selector selects all Instance that are managed by this daemon set.</p>
</td>
</tr>
<tr>
<td>
<code>template</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.InstanceTemplate">
InstanceTemplate
</a>
</em>
</td>
<td>
<p>Template is the instance template.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.LoadBalancerStatus">LoadBalancerStatus
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancer">LoadBalancer</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>collisionCount</code><br/>
<em>
int32
</em>
</td>
<td>
<p>CollisionCount is used to construct names for IP addresses for the load balancer.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.LoadBalancerTargetRef">LoadBalancerTargetRef
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancerDestination">LoadBalancerDestination</a>)
</p>
<div>
<p>LoadBalancerTargetRef is a load balancer target.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>uid</code><br/>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/types#UID">
k8s.io/apimachinery/pkg/types.UID
</a>
</em>
</td>
<td>
<p>UID is the UID of the target.</p>
</td>
</tr>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the target.</p>
</td>
</tr>
<tr>
<td>
<code>nodeRef</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#localobjectreference-v1-core">
Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<p>NodeRef references the node the destination network interface is on.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.LoadBalancerType">LoadBalancerType
(<code>string</code> alias)</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.InstanceSpec">InstanceSpec</a>, <a href="#core.apinet.ironcore.dev/v1alpha1.LoadBalancerSpec">LoadBalancerSpec</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Internal&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Public&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NATGatewayAutoscalerSpec">NATGatewayAutoscalerSpec
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.NATGatewayAutoscaler">NATGatewayAutoscaler</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>natGatewayRef</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#localobjectreference-v1-core">
Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<p>NATGatewayRef points to the target NATGateway to scale.</p>
</td>
</tr>
<tr>
<td>
<code>minPublicIPs</code><br/>
<em>
int32
</em>
</td>
<td>
<p>MinPublicIPs is the minimum number of public IPs to allocate for a NAT Gateway.</p>
</td>
</tr>
<tr>
<td>
<code>maxPublicIPs</code><br/>
<em>
int32
</em>
</td>
<td>
<p>MaxPublicIPs is the maximum number of public IPs to allocate for a NAT Gateway.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NATGatewayAutoscalerStatus">NATGatewayAutoscalerStatus
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.NATGatewayAutoscaler">NATGatewayAutoscaler</a>)
</p>
<div>
</div>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NATGatewayIP">NATGatewayIP
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.NATGatewaySpec">NATGatewaySpec</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name is the semantic name of the NAT gateway IP.</p>
</td>
</tr>
<tr>
<td>
<code>ip</code><br/>
<em>
<a href="../api/#api.ironcore.dev/net.IP">
github.com/ironcore-dev/ironcore-net/apimachinery/api/net.IP
</a>
</em>
</td>
<td>
<p>IP specifies a specific IP to allocate. If empty, a random IP will be allocated if possible.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NATGatewaySpec">NATGatewaySpec
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.NATGateway">NATGateway</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ipFamily</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#ipfamily-v1-core">
Kubernetes core/v1.IPFamily
</a>
</em>
</td>
<td>
<p>IPFamily is the IP family of the NAT gateway.</p>
</td>
</tr>
<tr>
<td>
<code>networkRef</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#localobjectreference-v1-core">
Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<p>NetworkRef references the network the NAT gateway is part of.</p>
</td>
</tr>
<tr>
<td>
<code>ips</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NATGatewayIP">
[]NATGatewayIP
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>IPs specifies the IPs of the NAT gateway.</p>
</td>
</tr>
<tr>
<td>
<code>portsPerNetworkInterface</code><br/>
<em>
int32
</em>
</td>
<td>
<p>PortsPerNetworkInterface specifies how many ports to allocate per network interface.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NATGatewayStatus">NATGatewayStatus
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.NATGateway">NATGateway</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>usedNATIPs</code><br/>
<em>
int64
</em>
</td>
<td>
<p>UsedNATIPs is the number of NAT IPs in-use.</p>
</td>
</tr>
<tr>
<td>
<code>requestedNATIPs</code><br/>
<em>
int64
</em>
</td>
<td>
<p>RequestedNATIPs is the number of requested NAT IPs.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NATIP">NATIP
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.NATTable">NATTable</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ip</code><br/>
<em>
<a href="../api/#api.ironcore.dev/net.IP">
github.com/ironcore-dev/ironcore-net/apimachinery/api/net.IP
</a>
</em>
</td>
<td>
<p>IP is the IP to NAT.</p>
</td>
</tr>
<tr>
<td>
<code>sections</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NATIPSection">
[]NATIPSection
</a>
</em>
</td>
<td>
<p>Sections are the sections of the NATIP.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NATIPSection">NATIPSection
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.NATIP">NATIP</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ip</code><br/>
<em>
<a href="../api/#api.ironcore.dev/net.IP">
github.com/ironcore-dev/ironcore-net/apimachinery/api/net.IP
</a>
</em>
</td>
<td>
<p>IP is the source IP.</p>
</td>
</tr>
<tr>
<td>
<code>port</code><br/>
<em>
int32
</em>
</td>
<td>
<p>Port is the start port of the section.</p>
</td>
</tr>
<tr>
<td>
<code>endPort</code><br/>
<em>
int32
</em>
</td>
<td>
<p>EndPort is the end port of the section</p>
</td>
</tr>
<tr>
<td>
<code>targetRef</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NATTableIPTargetRef">
NATTableIPTargetRef
</a>
</em>
</td>
<td>
<p>TargetRef references the entity having the IP.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NATTableIPTargetRef">NATTableIPTargetRef
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.NATIPSection">NATIPSection</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>uid</code><br/>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/types#UID">
k8s.io/apimachinery/pkg/types.UID
</a>
</em>
</td>
<td>
<p>UID is the UID of the target.</p>
</td>
</tr>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the target.</p>
</td>
</tr>
<tr>
<td>
<code>nodeRef</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#localobjectreference-v1-core">
Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<p>NodeRef references the node the destination network interface is on.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NetworkIDClaimRef">NetworkIDClaimRef
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.NetworkIDSpec">NetworkIDSpec</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>group</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>resource</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>namespace</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>uid</code><br/>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/types#UID">
k8s.io/apimachinery/pkg/types.UID
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NetworkIDSpec">NetworkIDSpec
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.NetworkID">NetworkID</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>claimRef</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NetworkIDClaimRef">
NetworkIDClaimRef
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NetworkInterfaceNAT">NetworkInterfaceNAT
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.NetworkInterfaceSpec">NetworkInterfaceSpec</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ipFamily</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#ipfamily-v1-core">
Kubernetes core/v1.IPFamily
</a>
</em>
</td>
<td>
<p>IPFamily is the IP family of the handling NAT gateway.</p>
</td>
</tr>
<tr>
<td>
<code>claimRef</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NetworkInterfaceNATClaimRef">
NetworkInterfaceNATClaimRef
</a>
</em>
</td>
<td>
<p>ClaimRef references the NAT claim handling the network interface&rsquo;s NAT.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NetworkInterfaceNATClaimRef">NetworkInterfaceNATClaimRef
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.NetworkInterfaceNAT">NetworkInterfaceNAT</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the claiming NAT gateway.</p>
</td>
</tr>
<tr>
<td>
<code>uid</code><br/>
<em>
<a href="https://pkg.go.dev/k8s.io/apimachinery/pkg/types#UID">
k8s.io/apimachinery/pkg/types.UID
</a>
</em>
</td>
<td>
<p>UID is the uid of the claiming NAT gateway.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NetworkInterfacePublicIP">NetworkInterfacePublicIP
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.NetworkInterfaceSpec">NetworkInterfaceSpec</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code><br/>
<em>
string
</em>
</td>
<td>
<p>Name is the semantic name of the network interface public IP.</p>
</td>
</tr>
<tr>
<td>
<code>ipFamily</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#ipfamily-v1-core">
Kubernetes core/v1.IPFamily
</a>
</em>
</td>
<td>
<p>IPFamily is the IP family of the IP. Has to match IP if specified. If unspecified and IP is specified,
will be defaulted by using the IP family of IP.
If only IPFamily is specified, a random IP of that family will be allocated if possible.</p>
</td>
</tr>
<tr>
<td>
<code>ip</code><br/>
<em>
<a href="../api/#api.ironcore.dev/net.IP">
github.com/ironcore-dev/ironcore-net/apimachinery/api/net.IP
</a>
</em>
</td>
<td>
<p>IP specifies a specific IP to allocate. If empty, a random ephemeral IP will be allocated.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NetworkInterfaceSpec">NetworkInterfaceSpec
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.NetworkInterface">NetworkInterface</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>nodeRef</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#localobjectreference-v1-core">
Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<p>NodeRef is the node the network interface is hosted on.</p>
</td>
</tr>
<tr>
<td>
<code>networkRef</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#localobjectreference-v1-core">
Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<p>NetworkRef references the network that the network interface is in.</p>
</td>
</tr>
<tr>
<td>
<code>ips</code><br/>
<em>
<a href="../api/#api.ironcore.dev/net.IP">
[]github.com/ironcore-dev/ironcore-net/apimachinery/api/net.IP
</a>
</em>
</td>
<td>
<p>IPs are the internal IPs of the network interface.</p>
</td>
</tr>
<tr>
<td>
<code>prefixes</code><br/>
<em>
<a href="../api/#api.ironcore.dev/net.IPPrefix">
[]github.com/ironcore-dev/ironcore-net/apimachinery/api/net.IPPrefix
</a>
</em>
</td>
<td>
<p>Prefixes are additional prefixes to route to the network interface.</p>
</td>
</tr>
<tr>
<td>
<code>natGateways</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NetworkInterfaceNAT">
[]NetworkInterfaceNAT
</a>
</em>
</td>
<td>
<p>NATs specify the NAT of the network interface IP family.
Can only be set if there is no matching IP family in PublicIPs.</p>
</td>
</tr>
<tr>
<td>
<code>publicIPs</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NetworkInterfacePublicIP">
[]NetworkInterfacePublicIP
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PublicIPs are the public IPs the network interface should have.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NetworkInterfaceState">NetworkInterfaceState
(<code>string</code> alias)</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.NetworkInterfaceStatus">NetworkInterfaceStatus</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;Error&#34;</p></td>
<td><p>NetworkInterfaceStateError is used for any NetworkInterface that is some error occurred.</p>
</td>
</tr><tr><td><p>&#34;Pending&#34;</p></td>
<td><p>NetworkInterfaceStatePending is used for any NetworkInterface that is in an intermediate state.</p>
</td>
</tr><tr><td><p>&#34;Ready&#34;</p></td>
<td><p>NetworkInterfaceStateReady is used for any NetworkInterface that is ready.</p>
</td>
</tr></tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NetworkInterfaceStatus">NetworkInterfaceStatus
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.NetworkInterface">NetworkInterface</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>state</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NetworkInterfaceState">
NetworkInterfaceState
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>pciAddress</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.PCIAddress">
PCIAddress
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>prefixes</code><br/>
<em>
<a href="../api/#api.ironcore.dev/net.IPPrefix">
[]github.com/ironcore-dev/ironcore-net/apimachinery/api/net.IPPrefix
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>publicIPs</code><br/>
<em>
<a href="../api/#api.ironcore.dev/net.IP">
[]github.com/ironcore-dev/ironcore-net/apimachinery/api/net.IP
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>natIPs</code><br/>
<em>
<a href="../api/#api.ironcore.dev/net.IP">
[]github.com/ironcore-dev/ironcore-net/apimachinery/api/net.IP
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NetworkSpec">NetworkSpec
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.Network">Network</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>id</code><br/>
<em>
string
</em>
</td>
<td>
<p>ID is the ID of the network.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NetworkStatus">NetworkStatus
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.Network">Network</a>)
</p>
<div>
</div>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NodeAffinity">NodeAffinity
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.Affinity">Affinity</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>requiredDuringSchedulingIgnoredDuringExecution</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NodeSelector">
NodeSelector
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NodeSelector">NodeSelector
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.NodeAffinity">NodeAffinity</a>)
</p>
<div>
<p>NodeSelector represents the union of the results of one or more queries
over a set of nodes; that is, it represents the OR of the selectors represented
by the node selector terms.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>nodeSelectorTerms</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NodeSelectorTerm">
[]NodeSelectorTerm
</a>
</em>
</td>
<td>
<p>Required. A list of node selector terms. The terms are ORed.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NodeSelectorOperator">NodeSelectorOperator
(<code>string</code> alias)</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.NodeSelectorRequirement">NodeSelectorRequirement</a>)
</p>
<div>
<p>NodeSelectorOperator is the set of operators that can be used in
a node selector requirement.</p>
</div>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;DoesNotExist&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Exists&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Gt&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;In&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;Lt&#34;</p></td>
<td></td>
</tr><tr><td><p>&#34;NotIn&#34;</p></td>
<td></td>
</tr></tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NodeSelectorRequirement">NodeSelectorRequirement
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.NodeSelectorTerm">NodeSelectorTerm</a>)
</p>
<div>
<p>NodeSelectorRequirement is a requirement for a selector. It&rsquo;s a combination of the key to match, the operator
to match with, and zero to n values, depending on the operator.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>key</code><br/>
<em>
string
</em>
</td>
<td>
<p>Key is the key the selector applies to.</p>
</td>
</tr>
<tr>
<td>
<code>operator</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NodeSelectorOperator">
NodeSelectorOperator
</a>
</em>
</td>
<td>
<p>Operator represents the key&rsquo;s relationship to the values.
Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</p>
</td>
</tr>
<tr>
<td>
<code>values</code><br/>
<em>
[]string
</em>
</td>
<td>
<p>Values are the values to relate the key to via the operator.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NodeSelectorTerm">NodeSelectorTerm
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.NodeSelector">NodeSelector</a>)
</p>
<div>
<p>NodeSelectorTerm matches no objects if it&rsquo;s empty. The requirements of the selector are ANDed.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>matchExpressions</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NodeSelectorRequirement">
[]NodeSelectorRequirement
</a>
</em>
</td>
<td>
<p>MatchExpressions matches nodes by the label selector requirements.</p>
</td>
</tr>
<tr>
<td>
<code>matchFields</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.NodeSelectorRequirement">
[]NodeSelectorRequirement
</a>
</em>
</td>
<td>
<p>MatchFields matches the nodes by their fields.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NodeSpec">NodeSpec
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.Node">Node</a>)
</p>
<div>
</div>
<h3 id="core.apinet.ironcore.dev/v1alpha1.NodeStatus">NodeStatus
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.Node">Node</a>)
</p>
<div>
</div>
<h3 id="core.apinet.ironcore.dev/v1alpha1.PCIAddress">PCIAddress
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.NetworkInterfaceStatus">NetworkInterfaceStatus</a>)
</p>
<div>
<p>PCIAddress is a PCI address.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>domain</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>bus</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>slot</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>function</code><br/>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.TopologySpreadConstraint">TopologySpreadConstraint
</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.InstanceSpec">InstanceSpec</a>)
</p>
<div>
<p>TopologySpreadConstraint specifies how to spread matching instances among the given topology.</p>
</div>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>maxSkew</code><br/>
<em>
int32
</em>
</td>
<td>
<p>MaxSkew describes the degree to which instances may be unevenly distributed.
When <code>whenUnsatisfiable=DoNotSchedule</code>, it is the maximum permitted difference
between the number of matching instances in the target topology and the global minimum.
The global minimum is the minimum number of matching instances in an eligible domain
or zero if the number of eligible domains is less than MinDomains.</p>
</td>
</tr>
<tr>
<td>
<code>topologyKey</code><br/>
<em>
string
</em>
</td>
<td>
<p>TopologyKey is the key of node labels. Nodes that have a label with this key
and identical values are considered to be in the same topology.
We consider each <key, value> as a &ldquo;bucket&rdquo;, and try to put balanced number
of instances into each bucket.
We define a domain as a particular instance of a topology.
Also, we define an eligible domain as a domain whose nodes meet the requirements of
nodeAffinityPolicy and nodeTaintsPolicy.</p>
</td>
</tr>
<tr>
<td>
<code>whenUnsatisfiable</code><br/>
<em>
<a href="#core.apinet.ironcore.dev/v1alpha1.UnsatisfiableConstraintAction">
UnsatisfiableConstraintAction
</a>
</em>
</td>
<td>
<p>WhenUnsatisfiable indicates how to deal with a instance if it doesn&rsquo;t satisfy
the spread constraint.
- DoNotSchedule (default) tells the scheduler not to schedule it.
- ScheduleAnyway tells the scheduler to schedule the instance in any location,
but giving higher precedence to topologies that would help reduce the
skew.</p>
</td>
</tr>
<tr>
<td>
<code>labelSelector</code><br/>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
<p>LabelSelector is used to find matching instances.
Instances that match this label selector are counted to determine the number of instances
in their corresponding topology domain.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="core.apinet.ironcore.dev/v1alpha1.UnsatisfiableConstraintAction">UnsatisfiableConstraintAction
(<code>string</code> alias)</h3>
<p>
(<em>Appears on:</em><a href="#core.apinet.ironcore.dev/v1alpha1.TopologySpreadConstraint">TopologySpreadConstraint</a>)
</p>
<div>
</div>
<table>
<thead>
<tr>
<th>Value</th>
<th>Description</th>
</tr>
</thead>
<tbody><tr><td><p>&#34;DoNotSchedule&#34;</p></td>
<td><p>DoNotSchedule instructs the scheduler not to schedule the instance
when constraints are not satisfied.</p>
</td>
</tr></tbody>
</table>
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>
</em></p>
