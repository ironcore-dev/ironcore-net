// Copyright 2023 IronCore authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

const (
	ControllerRevisionHashLabel = "apinet.ironcore.dev/controller-revision-hash"

	IPFamilyLabel = "apinet.ironcore.dev/ip-family"
	IPIPLabel     = "apinet.ironcore.dev/ip"

	TopologyLabelPrefix    = "topology.core.apinet.ironcore.dev/"
	TopologyPartitionLabel = TopologyLabelPrefix + "partition"
	TopologyZoneLabel      = TopologyLabelPrefix + "zone"
)
