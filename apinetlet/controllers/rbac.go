// Copyright 2023 OnMetal authors
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

package controllers

// Additional required RBAC rules

// Rules required for kubeconfig-rotation
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch
//+cluster=apinet:kubebuilder:rbac:groups=certificates.k8s.io,resources=certificatesigningrequests,verbs=create;get;list;watch
//+cluster=apinet:kubebuilder:rbac:groups=certificates.k8s.io,resources=certificatesigningrequests/apinetletclient,verbs=create

// Rules required for delegated authentication
//+cluster=apinet:kubebuilder:rbac:groups=authentication.k8s.io,resources=tokenreviews,verbs=create
//+cluster=apinet:kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews,verbs=create
