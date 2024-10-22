```mermaid
graph TD
    Manager[Manager] -->|Assigns Task| Worker1[Worker Node 1]
    Manager -->|Assigns Task| Worker2[Worker Node 2]
    Manager -->|Assigns Task| Worker3[Worker Node 3]

    subgraph Worker1[Worker Node 1]
        Docker1[Docker Container Runtime]
    end

    subgraph Worker2[Worker Node 2]
        Docker2[Docker Container Runtime]
    end

    subgraph Worker3[Worker Node 3]
        Docker3[Docker Container Runtime]
    end
```

## Usage:
```bash
Cube is a CLI tool for orchestrating tasks across a distributed system.
It assigns tasks to worker nodes, which execute them in Docker containers.
With Cube, you can efficiently manage, monitor, and scale tasks across multiple nodes.

Usage:
  cube [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  manager     Manager command to operate a Cube manager
  node        Node command to list nodes.
  run         Run a new task.
  status      Status command to list tasks.
  stop        Stop a running task.
  worker      Worker command to operate a Cube worker node.

Flags:
  -h, --help     help for cube
  -t, --toggle   Help message for toggle

Use "cube [command] --help" for more information about a command.
```
