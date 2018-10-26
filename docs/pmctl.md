
```console
$ pmctl [command]
pmctl is a command-line tool to control nodes, pods, and networks on placemat

Usage:
  pmctl [command]

Available Commands:
  help        Help about any command
  net         net subcommand
  node        node subcommand
  pod         pod subcommand

Flags:
      --endpoint string    API endpoint of the target placemat (default "http://localhost:10808")
  -h, --help               help for pmctl
```

#### node subcommand

`pmctl node list`
`pmctl node enter <NODE>`
`pmctl node action start <NODE>`
`pmctl node action stop <NODE>`
`pmctl node action restart <NODE>`

#### pod subcommand

`pmctl pod list`
`pmctl pod enter [--app=<APP>] [COMMANDS...]`


#### net subcommand

`pmctl net up <DEVICE>`
`pmctl net down <DEVICE>`
`pmctl net delay [--delay=<DELAY>] <DEVICE>`
`pmctl net loss [--loss=<LOSS>] <DEVICE>`
`pmctl net clear <DEVICE>`

