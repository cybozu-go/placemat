# Boot server example

This example constructs a boot server and workers (client) nodes.  The worker
nodes boots from the boot server.  The configuration of the example is shown in
the following:

```
          [ Host ]
             | 172.16.0.1
             |                  172.16.0.0/24
---+---------+------+------------+------ net0
   |                |            |
   | 172.16.0.3     |            |
.------.       .----------. .----------.
| boot |       | worker-1 | | worker-2 |
'------'       '----------' '----------'
```

To run the example, launch placemat by following:

```console
$ sudo placemat cluster.yml
```

The cluster configuration is described in [`cluster.example.yml`](cluster.example.yml).
The cluster contains a Network resource named `net0`, a Node resource named
`boot`, and a NodeSet resource named `worker`.

Network resource `net0` expose an interface as bridge to host, with IP address
`172.16.0.1`.  The `boot` node boots Ubuntu from [Ubuntu Cloud Image][] and it
initialized by cloud-init.  Its settings are described in
[`user-data.example.yml`](user-data.example.yml) and
[`network-config.example.yml`](network-config.example.yml).

You can log-in to `boot` node by `ubuntu`/`ubuntu`.  For `worker-N` nodes
provisioned by the boot server, they will boot [iPXE][] provided from boot
server.

```console
$ sudo pmctl node enter boot        # login with ubuntu/ubuntu
$ sudo pmctl node enter worker-1
```

[Ubuntu Cloud Image]: https://cloud-images.ubuntu.com/
[UEFI HTTP Boot]: https://github.com/tianocore/tianocore.github.io/wiki/HTTP-Boot
[iPXE]: http://ipxe.org/
