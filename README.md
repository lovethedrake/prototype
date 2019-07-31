# The Drake

This repository houses a _proof-of-concept_ that demonstrates how CI/CD
pipelines based on an _open specification_ can be portable across multiple,
spec-compliant developer tools and CI/CD platforms.

To demonstrate this, two separate pieces of software were created that operate
on a common underlying pipeline model.

## The Drake Pipeline Specification

No _formal_ specification for the Drake pipeline model exists at this time, but
one is implied. A formal specification may be extracted from this early effort
if the model is proven tenable. The implied specification is refered to as the
__Drake Pipeline Specification__.

Drake __pipelines__ are _instantiations_ of the Drake Pipeline Specification and
are composed of one or more __stages__ which are executed _in sequence_. Stages,
in turn, are composed of __jobs__, which _may_ be executed concurrently.
During pipeline execution, all jobs in a given stage must complete before
that stage can be considered complete and the subsequent stage can begin.

Jobs are the fundamental building blocks of a Drake pipeline. A simple job
pairs OCI container configuration with a command to execute inside that
container. Commands may be of arbitrary complexity-- often, they may invoke a
shell script. Complex jobs composed of multiple containers that are networked
together are also possible.

For examples of Drake jobs and pipelines, refer to this repository's own
`Drakefile.yaml`.

### `drake`

The unqualified term "Drake," _especially_ if downcased and monospaced (i.e.
`drake`) refers to a specific, spec-compliant, developer-facing tool that
orchestrates execution of Drake pipelines using nothing but a (local) Docker
daemon on a developer's system-- something very few established CI/CD platforms
enable a developer to do.

Perhaps more importantly, the `drake` tool also permits local execution of
individual jobs. This makes it practical for developers to encapsulate tasks
that comprise their development workflow within jobs and enables frictionless
_reuse_ of those same jobs in defining CI/CD pipelines. For instance,
hypothetical jobs such as `test` or `lint` could be useful to a developer
locally, but are also sensible tasks to incorporate into a CI/CD pipeline.

#### `drake` Installation

First, be certain that `docker` is installed and functioning normally.

It's critically important to understand that `drake` will only work with a
_local_ Docker server (`dockerd`). A remote Docker server will not work because
`drake` mounts your project's source code into job containers and that is not
possible (or at least not practical) with a remote Docker server.

A Docker server running in a local VM via Docker for Mac or Docker for Windows
_will_ work as long as your project is located somewhere within your home
directory since Docker for Mac and Docker for Windows typically mount your
home directory into the VM.

With Docker functioning correctly, grab the latest pre-built `drake` binary from
[here](https://github.com/lovethedrake/prototype/releases/latest), rename it as
`drake` and place it on your system's path.

### Custom Brigade Worker for Drake

The concept of portable pipelines based on an open specification cannot
adequately be demonstrated without at least two implementations. To that end,
the __Custome Brigade Worker for Drake__ adapts the event-driven,
Kubernetes-native scripting engine into a full-fledged and Drake-compliant CI/CD
system. It drives all Brigade job execution based on pipelines defined in your
project's `Drakefile.yaml` _instead of_ the usual `brigade.js`.

Note: Brigade was merely the _fastest_ path to developing a Drake-compatible
CI/CD system. The vision for portable CI/CD pipelines promotes the development
of _many_ distinct CI/CD systems that may provide their own differentiated
experiences so long as they remain Drake-compatible.

#### Using Drake Pipelines with Brigade

First, be certain Brigade is installed and functioning normally in your
Kubernetes cluster. Ensure the Brigade Github App is also installed and
functioning normally. (This is not currently an easy process, but this is
Brigade pain; not Drake pain.)

Additionally, your Kubernetes cluster _must_ have a `StorageClass` that can
provision `PersistentVolumes` that support access mode `ReadWriteMany`. This is
because the custom worker will clone your project source code to a volume which
it may, at times, mount to multiple Kubernetes pods concurrently.

Use `brig project create` and follow all prompts to create a new Brigade
project. When prompted, choose to carry out advanced configuration. Ensure
shared storage uses a `StorageClass` that can provision `PersistentVolumes` that
support access mode `ReadWriteMany` and ensure the following configuration
related to a custom worker image is set as follows:

* Docker registry or Docker Hub org: `lovethedrake`
* Image: `prototype-brigade-worker`
* Tag: `v0.0.1`
* Command: `/brigade-worker/bin/brigade-worker`

Note any secrets generated during this process that may need to be added to your
Github App configuration.
