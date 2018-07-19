*Docker Daemon UNIX Socket Proxy for CI Child Containers*

# Summary

Acts as a UNIX socket proxy (eg. for `/var/run/docker.sock`) for Docker client <-> Docker daemon communication, injecting extra `docker run` arguments (eg. label/s and `cgroup-parent`) to all resource creation calls.

# Background

Let's say you are running CI (Jenkins, etc) slaves in your Docker-based container scheduled server cluster. And you want to run Docker-in-Docker to allow your CI jobs to spawn containers, but you read @jpetazzo 's post [here](https://jpetazzo.github.io/2015/09/03/do-not-use-docker-in-docker-for-ci/) about instead mounting the host Docker socket into the CI slave container.

At this point, your container scheduler knows about all containers that are started, and takes care of garbage collection and resource allocation for you. You now have another Docker client that can spawn "sibling" containers, with no relationship to what started them. If your CI slave container is terminated, any containers or images left behind by your CI jobs (especially in an `Up`/`Running` state) may not be garbage collected. In addition, you may "reserve" CPU/memory resources for containers spawned by your scheduler, but these child containers would not be recognised and allocated resources by the scheduler, causing a non-visible resource oversubscription.

## Labels

By putting a UNIX socket proxy between the volume mapping, we can add extra Docker labels to all newly created images or containers, allowing for reaping/GC by existing methods (eg. `docker container prune --filter 'labelname=xyz'`, or alternate approaches for running containers). There is currently no native capability to force adding labels for all operations by a single Docker API client.

The same rule goes for (specific API calls):

- containers `/containers/create`
- images (builds, not pulls) `/build``
- networks `/networks/create`
- volumes `/volumes/create`
- services `/services/create`
- secrets `/secrets/create`
- configs `/configs/create`


Another approach to this is for the native Docker client to support default labels in the client config files. Requested upstream [here](https://github.com/moby/moby/issues/33644) - this will only cover using the official Docker CLI client and not alternate clients that talk to the same UNIX socket, which this project would cover.

Note: `docker import` operations will not apply a label currently. This feature could be added if worthwhile for completeness (the API seems to have what's required).

## Parent CGroup

You can also apply a custom `cgroup-parent` to all child containers so they are grouped, to avoid OOM collateral damage to your other workloads on your container scheduler managed cluster, and "reserve" system resources via your scheduler. Eg. you may need 256MB for a Jenkins agent, but you might allocate 2048MB and the child containers will use the surplus when spawned within the same parent cgroup.

This will be applied for `/containers/create` API calls only (`docker run` effectively), when the `--cgroup-parent` (`-cg`) flag is set.

There is currently no way to define the CGroup name to be used, it is detected automatically. As this process is designed to be run with access to the Docker daemon socket, a `/containers/${id-self}/json` API call is performed and the `CgroupParent` value is used (shared parent for this container + any children). If `CgroupParent` is empty and `--cgroup-parent` is enabled, the `/containers/create` API call will fail with an error.

## Parent CGroup Container Startup Example

Concept seems to work:

```
cd /sys/fs/cgroup/memory
mkdir testcontainergroup
cd testcontainergroup
echo 134217728 > memory.limit_in_bytes
docker run -it --rm --test-cgroup=/testcontainergroup/ alpine:3.6 sh
(run something that consumes memory, and it should max out about 128MB)
```

Note: `docker stats` will still show the unconstrained memory threshold, not the parent cgroup limit.

# Usage

```
NAME:
   dockerd-ci-proxy - Docker Daemon UNIX Socket Proxy for CI Child Containers

USAGE:
   dockerd-ci-proxy [global options] command [command options] [arguments...]

VERSION:
   0.0.1

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --debug, -d                       Debug Mode [$DCP_DEBUG]
   --dockersocket value, --ds value  The Docker daemon API UNIX socket to connect to (default: "/var/run/docker.sock") [$DCP_DOCKER_SOCKET]
   --listensocket value, --ls value  The UNIX listen socket for this process, Docker API clients will point at this path (default: "/var/run/docker-ci-proxy.sock") [$DCP_LISTEN_SOCKET]
   --cgroupparent, --cp              If enabled, overrides the CgroupParent of create operations to match this container [$DCP_CGROUP_PARENT]
   --labelname value, --ln value     The Docker label name to apply to resources (default: "Created-Via") [$DCP_LABEL_NAME]
   --labelvalue value, --lv value    The Docker label value to apply to resources (default: "dockerd-ci-proxy") [$DCP_LABEL_VALUE]
   --help, -h                        show help
   --version, -v                     print the version
```

# License

Licensed under the MIT License
