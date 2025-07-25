# Step-by-step private NeoFS deployment

This is a short guide on how to deploy a private NeoFS storage network on bare
metal without docker images. This guide does not cover details on how to start
consensus, Alphabet, or Storage nodes. This guide covers only `neofs-adm`
related configuration details.

## Prerequisites

To follow this guide you need:
- latest released version of [neo-go](https://github.com/nspcc-dev/neo-go/releases) (v0.97.2 at the moment),
- latest released version of [neofs-adm](https://github.com/nspcc-dev/neofs-node/releases) utility (v0.25.1 at the moment),
- latest released version of compiled [neofs-contract](https://github.com/nspcc-dev/neofs-contract/releases) (v0.11.0 at the moment).

## Step 1: Prepare network configuration

To start a network, you need a set of consensus nodes, the same number of
Alphabet nodes and any number of Storage nodes. While the number of Storage
nodes can be scaled almost infinitely, the number of consensus and Alphabet
nodes can't be changed so easily right now. Consider this before going any further.

It is easier to use`neofs-adm` with a predefined configuration. First, create
a network configuration file. In this example, there is going to be only one
consensus / Alphabet node in the network.

```
$ neofs-adm config init --path foo.network.yml
Initial config file saved to foo.network.yml

$ cat foo.network.yml
rpc-endpoint: https://neo.rpc.node:30333
alphabet-wallets: /home/user/deploy/alphabet-wallets
network:
  max_object_size: 67108864
  epoch_duration: 240
  basic_income_rate: 0
  fee:
    candidate: 0
    container: 0
    withdraw: 0
credentials:
  az: hunter2
```

For private installation, it is recommended to set all **fees** and **basic
income rate** to 0.

As for **epoch duration**, consider consensus node block generation frequency.
With default 15 seconds per block, 240 blocks are going to be a 1-hour epoch.

For **max object size**, 67108864 (64 MiB) or 134217728 (128 MiB) should provide
good chunk distribution in most cases.

With this config, generate wallets (private keys) of consensus nodes. The same
wallets will be used for Alphabet nodes. Make sure, that dir for alphabet
wallets already exists.

```
$ neofs-adm -c foo.network.yml fschain generate-alphabet --size 1
size: 1
alphabet-wallets: /home/user/deploy/alphabet-wallets
wallet[0]: hunter2
```

Do not lose wallet files and network config. Store it in an encrypted backed up
storage.

## Step 2: Launch consensus nodes

Configure blockchain nodes with the generated wallets from the previous step.
Config examples can be found in
[neo-go repository](https://github.com/nspcc-dev/neo-go/tree/master/config).

Gather public keys from **all** generated wallets. We are interested in the first
`simple signature contract` public key.

```
$ neo-go wallet dump-keys -w alphabet-wallets/az.json
NitdS4k4f1Hh5mbLJhAswBK3WC2gQgPN1o (simple signature contract):
02c1cc85f9c856dbe2d02017349bcb7b4e5defa78b8056a09b3240ba2a8c078869

NiMKabp3ddi3xShmLAXhTfbnuWb4cSJT6E (1 out of 1 multisig contract):
02c1cc85f9c856dbe2d02017349bcb7b4e5defa78b8056a09b3240ba2a8c078869

NiMKabp3ddi3xShmLAXhTfbnuWb4cSJT6E (1 out of 1 multisig contract):
02c1cc85f9c856dbe2d02017349bcb7b4e5defa78b8056a09b3240ba2a8c078869
```

Put the list of public keys into `ProtocolConfiguration.StandbyCommittee`
section. Specify the wallet path and the password in `ApplicationConfiguration.P2PNotary`
and `ApplicationConfiguration.UnlockWallet` sections.

```yaml
ProtocolConfiguration:
  StandbyCommittee:
    - 02c1cc85f9c856dbe2d02017349bcb7b4e5defa78b8056a09b3240ba2a8c078869
ApplicationConfiguration:
  P2PNotary:
    Enabled: true
    UnlockWallet:
      Path: "/home/user/deploy/alphabet-wallets/az.json"
      Password: "hunter2"
  UnlockWallet:
    Path: "/home/user/deploy/alphabet-wallets/az.json"
    Password: "hunter2"
```

Then, launch consensus nodes. They should connect to each other and start
producing blocks in consensus. You might want to deploy additional RPC
nodes at this stage because Storage nodes should be connected to the chain too.
It is not recommended to use a consensus node as an RPC node due to security policies
and possible overload issues.

## Step 3: Launch Alphabet nodes

Configure Alphabet nodes with the wallets generated in step 1. For
`fschain.validators` use a list of public keys from
`ProtocolConfiguration.StandbyCommittee`.

```yaml
wallet:
  path: "/home/user/deploy/alphabet-wallets/az.json"
  password: "hunter2"
  account: "NitdS4k4f1Hh5mbLJhAswBK3WC2gQgPN1o"

fschain:
  validators:
    - 02c1cc85f9c856dbe2d02017349bcb7b4e5defa78b8056a09b3240ba2a8c078869
```

## Step 4: Launch Storage node

Generate a new wallet for a Storage node.

```
$ neofs-adm -c foo.network.yml fschain generate-storage-wallet --storage-wallet ./sn01.json --initial-gas 10.0
New password >
Waiting for transactions to persist...

$ neo-go wallet dump-keys -w sn01.json
Ngr7p8Z9S22XDH6VkUG9oXobv8zZRAWwwv (simple signature contract):
0355eccb72cd46f09a3e5237eaa0f4949cceb5ecfa5a225bd3bb9fd021c4d75b85
```

Configure the Storage node to use this wallet.

```
node:
  wallet:
    path: "/home/user/deploy/sn01.json"
    address: "Ngr7p8Z9S22XDH6VkUG9oXobv8zZRAWwwv"
    password: "foobar"
```

The storage node will be included in the network map in the next NeoFS epoch. To
speed up this process, you can increment epoch counter immediately.

```
$ neofs-adm -c foo.network.yml fschain force-new-epoch
Current epoch: 8, increase to 9.
Waiting for transactions to persist...
```

---

After that, NeoFS Storage is ready to work. You can access it directly or
with protocol gates.
