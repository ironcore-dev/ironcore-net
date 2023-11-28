// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

// Additional required RBAC rules

// Rules required for kubeconfig-rotation
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch
//+cluster=apinet:kubebuilder:rbac:groups=certificates.k8s.io,resources=certificatesigningrequests,verbs=create;get;list;watch
//+cluster=apinet:kubebuilder:rbac:groups=certificates.k8s.io,resources=certificatesigningrequests/apinetletclient,verbs=create

// Rules required for delegated authentication
//+cluster=apinet:kubebuilder:rbac:groups=authentication.k8s.io,resources=tokenreviews,verbs=create
//+cluster=apinet:kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews,verbs=create
