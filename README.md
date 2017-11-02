*Docker Daemon UNIX Socket Label Proxy*

# Summary

Acts as a UNIX socket proxy (eg. for `/var/run/docker.sock`) for Docker client <-> Docker daemon communication, injecting extra Docker label/s to all resource creation calls.

# Background

Let's say you are running CI (Jenkins, etc) slaves in your Docker-based container scheduled server cluster. And you want to run Docker-in-Docker to allow your CI jobs to spawn containers, but you read @jpetazzo 's post [here](https://jpetazzo.github.io/2015/09/03/do-not-use-docker-in-docker-for-ci/) about instead mounting the host Docker socket into the CI slave container.

At this point, your container scheduler knows about all containers that are started, and takes care of garbage collection for you. You now have another Docker client that can spawn "sibling" containers, with no relationship to what started them. If your CI slave container is terminated, any containers or images left behind by your CI jobs (especially in an `Up`/`Running` state) may not be garbage collected.

By putting a UNIX socket proxy between the volume mapping, we can add extra Docker labels to all newly created images or containers, allowing for reaping/GC by existing methods (eg. `docker container prune --filter 'labelname=xyz'`, or alternate approaches for running containers). There is currently no native capability to force adding labels for all operations by a single Docker client.

The same rule goes for:

- containers
- images
- networks
- volumes

Another approach to this is for the native Docker client to support default labels in the client config files. Requested upstream [here](https://github.com/moby/moby/issues/33644) - this will only cover using the official Docker CLI client and not alternate clients that talk to the same UNIX socket, which this project would cover.

# Shortcomings

This won't reserve capacity on the container scheduler (eg. ECS)... which won't be amazing :s

# Usage



# License

Licensed under the MIT License
