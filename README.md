```
 █████╗ ██╗  ██╗ ██████╗ ███████╗██╗      ██████╗ ██╗    ██╗
██╔══██╗██║ ██╔╝██╔═══██╗██╔════╝██║     ██╔═══██╗██║    ██║
███████║█████╔╝ ██║   ██║█████╗  ██║     ██║   ██║██║ █╗ ██║
██╔══██║██╔═██╗ ██║   ██║██╔══╝  ██║     ██║   ██║██║███╗██║
██║  ██║██║  ██╗╚██████╔╝██║     ███████╗╚██████╔╝╚███╔███╔╝
╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝ ╚═╝     ╚══════╝ ╚═════╝  ╚══╝╚══╝
```

# AkôFlow - Open Source Engine for Containerized Scientific Workflows

## Database lifecycle

The SQLite database is installed from one canonical schema and is not
migrated between schema shapes. When an engine reports an incompatible
database, stop it, remove the configured SQLite database file, and start it
again to create a clean database. Export any data that must be retained before
removing the file.

## Docker build service

The Compose stack runs the API, a rootless BuildKit daemon, and a persistent
artifact-store volume. It does **not** mount `/var/run/docker.sock`.

```sh
cp .env.example .env
# edit both values in .env
docker compose up --build
```

`AKOFLOW_API_TOKEN` is mandatory when the container listens on `0.0.0.0`; send
it as `Authorization: Bearer <token>`. `GET /akoflow-api/instance/` is the
sole public bootstrap endpoint, exposing only the control-plane identity so an
environment can discover it before credential exchange. Set
`AKOFLOW_API_ALLOWED_ORIGINS` to the specific Admin UI origins allowed to make
browser requests. BuildKit produces OCI archives. The server image includes a native-architecture Apptainer build
and can convert OCI to SIF; Compose grants `/dev/fuse` and `SYS_ADMIN` solely
for that operation, not Docker host control.

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
