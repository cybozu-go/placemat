# Boot server example

This example constructs a cluster of one boot server and two worker nodes.
The worker nodes boot from the boot server.  The configuration of the example
is as follows:

```
          [ Host ]
             | 172.16.0.1
             |                     172.16.0.0/24
---+---------+------+---------------+------ net0
   |                |               |
   | 172.16.0.11    | 172.16.0.101  | 172.16.0.102
.------.       .----------.    .----------.
| boot |       | worker-1 |    | worker-2 |
'------'       '----------'    '----------'
```

To run the example, launch `placemat` by the following command:

```console
$ sudo placemat2 cluster.example.yml
```

The cluster configuration is described in [`cluster.example.yml`](cluster.example.yml).
The cluster contains a Network resource named `net0` and three Node resources
named `boot`, `worker-1`, and `worker-2`.  It also contains an Image resource
named `ubuntu-image`.

The network resource `net0` exposes an interface as a bridge to the host,
with an IP address of `172.16.0.1`.

The `boot` node boots Ubuntu from [Ubuntu Cloud Image][] specified by
`ubuntu-image`, and is initialized by cloud-init.  Its settings are described in
[`user-data.example.yml`](user-data.example.yml) and
[`network-config.example.yml`](network-config.example.yml).
It downloads boot images, and then starts [dnsmasq][] and [nginx][] on Ubuntu
to work as a network boot server.

The worker nodes run Qemu's default BIOS to boot from the network.  They load
[iPXE][] provided by the boot node, and then load [Flatcar Container Linux][].
They are configured with empty disks, which will be seen as `/dev/vda`.

You can log-in to the `boot` node by `ubuntu`/`ubuntu`.  As for the worker
nodes, they are configured to accept auto login from the console.

```console
$ sudo pmctl2 node enter boot        # login with ubuntu/ubuntu
$ sudo pmctl2 node enter worker-1    # autologin as "core" user

# type Ctrl-q and Ctrl-x to leave from the node console
```

[Ubuntu Cloud Image]: https://cloud-images.ubuntu.com/
[dnsmasq]: http://www.thekelleys.org.uk/dnsmasq/doc.html
[nginx]: https://nginx.org/
[UEFI HTTP Boot]: https://github.com/tianocore/tianocore.github.io/wiki/HTTP-Boot
[iPXE]: https://ipxe.org/
[Flatcar Container Linux]: https://kinvolk.io/docs/flatcar-container-linux/latest/
