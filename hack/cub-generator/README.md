# cub-generator
Generator go projects (tiger cubs)

Generated go projects using best practices and guidelines in Tigera

You can generate a simple skeleton for a project that contains a CI pipeline in Semaphore, a Makefile, a Docker Image.

You can also generate an HTTP server or a reverse proxy that has basic unit level test written.


## How to run

Generate an HTTP server:

```
cub-generator-amd64 -n <NAME> -l <PATH_FOR_THE_NEW_PROJECT> --type http-server
```

Generate a reverse proxy:

```
cub-generator-amd64 -n <NAME> -l <PATH_FOR_THE_NEW_PROJECT> --type proxy
```


Generate a project skeleton:

```
cub-generator-amd64 -n <NAME> -l <PATH_FOR_THE_NEW_PROJECT> --type skeleton
```
