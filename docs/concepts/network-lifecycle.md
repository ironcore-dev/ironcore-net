# Network lifecycle

## ID management

When creating a `Network`, a vacant network ID has to be allocated.
This allocation is done via cluster-scoped `NetworkID` objects.
A `NetworkID`s name is the ID it represents. As such, detecting whether
a `NetworkID` is taken can be easily done by `Get`ting the `NetworkID`
with the ID to check and inspecting the result: If the `NetworkID` is
present, it means it's taken. Otherwise, at least during the time of
inspection, the `NetworkID` is vacant and ready to be claimed.

The `NetworkID` and `Network` are tied together in the
[`Network`'s store `BeforeCreate` hook](../../internal/registry/network/storage.go) using
the [`networkidallocator`](../../internal/registry/network/networkidallocator/networkidallocator.go).

The `Allocator` tries to create `NetworkID`s with the `claimRef` pointing
to the `Network` about to be created. It continues to do so until it either
finds a vacant `NetworkID` (creation succeeds) or it times out after too
many attempts fail (`AlreadyExists` errors).

The valid `NetworkID` range can be configured using the `apiserver`s
`min-vni` / `max-vni` flags.

When deleting a `Network`, the corresponding `NetworkID` is cleaned up
alongside the claiming `Network`.

## Network Peering

When creating network peering both `ironcore` `Network`s has to specify matching, 
`spec.peerings` referencing each other respectively. A `ironcore` `Network` can be peered with 
multiple `network`s in any namespcae.

The `apinetlet` `NetworkController` checks if there are any `peeringClaimRefs` 
present in `ironcore` `Network`. If yes then get `ironcore-net` `Network` using `UID` of `peeringClaimRef` 
and add `spec.ID` of that `ironcore-net` `Network` into `spec.peeredIDs` of current `ironcore-net` `Network`.

The `apinetlet` `NetworkController` also checks for `status.peerings` in `ironcore` `Network`.
If yes then update `ironcore-net` `Network`s `status.peerings` with `ironcore` `Network`s `status.peerings`

Once `ironcore-net` `Network` is updated with `spec.peeredIDs`, `metalnetlet` `NetworkController` 
updates `metalnet` `Network` `spec.peeredIDs` with corresponding `ironcore-net` `Network` `spec.peeredIDs`