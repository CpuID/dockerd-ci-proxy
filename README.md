*Docker Daemon UNIX Socket Proxy for CI Child Containers*

# Summary

Acts as a UNIX socket proxy (eg. for `/var/run/docker.sock`) for Docker client <-> Docker daemon communication, injecting extra `docker run` arguments (eg. label/s and `cgroup-parent`) to all resource creation calls.

# Background

Let's say you are running CI (Jenkins, etc) slaves in your Docker-based container scheduled server cluster. And you want to run Docker-in-Docker to allow your CI jobs to spawn containers, but you read @jpetazzo 's post [here](https://jpetazzo.github.io/2015/09/03/do-not-use-docker-in-docker-for-ci/) about instead mounting the host Docker socket into the CI slave container.

At this point, your container scheduler knows about all containers that are started, and takes care of garbage collection and resource allocation for you. You now have another Docker client that can spawn "sibling" containers, with no relationship to what started them. If your CI slave container is terminated, any containers or images left behind by your CI jobs (especially in an `Up`/`Running` state) may not be garbage collected. In addition, you may "reserve" CPU/memory resources for containers spawned by your scheduler, but these child containers would not be recognised and allocated resources by the scheduler, causing a non-visible resource oversubscription.

## Labels

By putting a UNIX socket proxy between the volume mapping, we can add extra Docker labels to all newly created images or containers, allowing for reaping/GC by existing methods (eg. `docker container prune --filter 'labelname=xyz'`, or alternate approaches for running containers). There is currently no native capability to force adding labels for all operations by a single Docker client.

The same rule goes for:

- containers
- images
- networks
- volumes

Another approach to this is for the native Docker client to support default labels in the client config files. Requested upstream [here](https://github.com/moby/moby/issues/33644) - this will only cover using the official Docker CLI client and not alternate clients that talk to the same UNIX socket, which this project would cover.

## Parent CGroup

You can also apply a custom `cgroup-parent` to all child containers so they are grouped, to avoid OOM collateral damage to your other workloads on your container scheduler managed cluster, and "reserve" system resources via your scheduler. Eg. you may allocate 25% of memory on a system as "not managed by the scheduler", which you would hand to the CGroup to utilise.

# Shortcomings

This won't reserve capacity on the container scheduler (eg. ECS)... which won't be amazing :s

One potential workaround is just use `ECS_RESERVED_MEMORY` on the ECS agent to reserve say 25% of all memory for non-ECS stuff, eg. spawned containers? and adjust the % based on usage patterns?

Maybe also consider using a parent cgroup as a "memory pool" for all these random containers, as a way to avoid collateral damage from OOMs? eg. the 25% reserved gets applied to the "parent cgroup", and all containers spawned get thrown in that.

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



# License

Licensed under the MIT License
