# Cluster layout

Assign every node a zone and a capacity, and apply the layout in one step.

```terraform
--8<-- "examples/guides/cluster-layout/main.tf"
```

- `garage_cluster_status` lists the nodes the cluster knows about, with their full IDs and host names.
- `roles` is keyed by node ID. Here, the zone comes from the host name, and every node stores up to 2 TB.
- A node without `capacity` is a gateway: it answers requests but stores no data.
- `zone_redundancy` is unset here, so Garage replicates each partition in as many zones as possible. Set it to a number to require a minimum instead.

On apply, the provider stages every change and applies them as one new layout version. Garage then moves the data in the background; the `garage_cluster_health` data source shows when every partition is available again.

## Rules

- **The resource is authoritative.** A node in the layout but missing from `roles` is removed from it, so its data moves to the other nodes.
- **Changes staged by hand are reverted** before the provider applies its own.
- **Declare it once per cluster.** Two configurations managing the layout would undo each other.
- **`tofu destroy` leaves the layout as it is**, because removing every role would make the cluster unusable.

New nodes still have to be connected to the cluster before they appear in `garage_cluster_status`, with `garage node connect` or the bootstrap peers of `garage.toml`.
