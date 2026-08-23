```
 █████╗ ██╗  ██╗ ██████╗ ███████╗██╗      ██████╗ ██╗    ██╗
██╔══██╗██║ ██╔╝██╔═══██╗██╔════╝██║     ██╔═══██╗██║    ██║
███████║█████╔╝ ██║   ██║█████╗  ██║     ██║   ██║██║ █╗ ██║
██╔══██║██╔═██╗ ██║   ██║██╔══╝  ██║     ██║   ██║██║███╗██║
██║  ██║██║  ██╗╚██████╔╝██║     ███████╗╚██████╔╝╚███╔███╔╝
╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝ ╚═╝     ╚══════╝ ╚═════╝  ╚══╝╚══╝
```

# AkôFlow - Open Source Engine for Containerized Scientific Workflows

## Executable and workspace delivery

An activity declares how its executable is obtained. The planner resolves it
to a verified local path before submitting to an offline runtime such as
Slurm/Apptainer.

```yaml
command:
  executable:
    source:
      type: oci
      reference: docker.io/library/alpine:3.20
    delivery:
      strategy: gateway-transfer
      targetFormat: sif
```

Other source types are `catalog`, `local-container-image`, `build`,
`local-file`, `remote-file`, `object-storage`, and `http`. A SIF already
visible to compute nodes can be used in place:

```yaml
command:
  executable:
    source:
      type: remote-file
      resourceRef: plafrim
      path: /apps/containers/tool.sif
      format: sif
    delivery:
      strategy: use-in-place
```

`destination-pull` is explicit. It is the only mode in which a Slurm script
may receive `docker://…`; a plain `alpine:3.20` is rejected until an artifact
materialization resolves it to a local SIF/path.

Workspace data is content addressed. A `WorkspaceRevision` lists paths and
blob digests, the destination reports a `WorkspaceInventory`, and
reconciliation transfers only missing blobs. A workspace materialization is
committed only after every missing digest is verified. Executors use the same
barrier for a committed executable artifact and workspace before submission.

## Workflow Engine
