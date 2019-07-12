
# Go Module Learnings

Previous Tigera/Calico projects used [glide](https://github.com/Masterminds/glide)
to import and manage dependencies. For Voltron, we decided to start with Go
modules and adjust as needed. The decision was taken with an "eye to the
future", considering that most Go projects would have to migrate sooner or
later.

While the dependency challenges were different from other Tigera projects that
attempted to use Go modules (as discussed [here](https://docs.google.com/document/d/14NL4Mnn0sKvZO9921ePPsyq9Ci_KfiAzM1o1jrCG0ic/edit#)
-- for example, having to import modules from private repos) we noticed a few _gotchas_ worth discussing.

A good and comprehensive reference is [the Go modules wiki page](https://github.com/golang/go/wiki/Modules)
where most of the following items are discussed (and much more!).

## Background

(_[reference](https://github.com/golang/go/wiki/Modules#modules)_)

> A module is a collection of related Go packages that are versioned together as a single unit.

> * A repository contains one or more Go modules.
> * Each module contains one or more Go packages.
> * Each package consists of one or more Go source files in a single directory.

## The Normal Flow

At the base of the project, run:

    $ go mod init [github.com/my/repo]

This will create a file called `go.mod` which contains, at this point:

    module github.com/my/repo

    go 1.12

Most go commands (`go build`, `go test`, ...) will pull the required modules
out of the `import` statements and update the `go.mod` file automatically. As
such, running other commands (such as `go get`) isn't necessary unless _specific versions_ are required.

## Various _gotchas_

### go.sum

Another file, the `go.sum` file, is updated when the `go.mod` is modified.
The `go.sum` [isn't a lock file](https://github.com/golang/go/wiki/Modules#is-gosum-a-lock-file-why-does-gosum-include-information-for-module-versions-i-am-no-longer-using) unlike other package management systems.

The `go.sum`, for example, will contain references to modules no longer used in
the project. This is normal.

Even though it is recommended to add `go.sum` to Git, the file might be modified
locally if `go get` commands are executed. The result might vary from computer to computer.

A target was added to the Makefile to regenerate the `go.sum` from scratch:

    $ make mod-regen-sum

### Outdated go.mod

It's possible for the `go.mod` to miss modules or contain extra ones. The recommended approach
is to run:

    $ go mod tidy
    $ go mod verify

There's a make target to run those:

    $ make mod-tidy

Note: the static check in this repo will check that `go mod tidy` does _not_ end up
modifying the `go.mod`.

### GO111MODULE

The behavior is documented [here](https://github.com/golang/go/wiki/Modules#when-do-i-get-old-behavior-vs-new-module-based-behavior). Here are some recommendations:

- leave the `GO111MODULE` variable unset
- put your repos outside `GOPATH`

At this point, you shouldn't have to set/unset/manipulate `GO111MODULE`. (if you do, let's document why)

### Global Tool Installs

Many tools will suggest to install using `go get ...`. If you run this inside a
Go module project, the `go.mod` will be modified. (!)

There are some workarounds ... here's the [doc](https://github.com/golang/go/wiki/Modules#why-does-installing-a-tool-via-go-get-fail-with-error-cannot-find-main-module) and the relevant
[Github issue](https://github.com/golang/go/issues/24250#issuecomment-377553022).

This is a bit messy...

### What are all those `indirect` in go.mod?

That happens when imported packages aren't using Go modules yet. To keep track of those _indirect_ dependencies, they need to be kept in `go.mod`. This is explained [here](https://github.com/golang/go/wiki/Modules#why-does-go-mod-tidy-record-indirect-and-test-dependencies-in-my-gomod).

### What are all those `incompatible` in go.mod?

That happens when imported packages aren't using Go modules yet _BUT_ were using a v2+ versioning. This is explained [here](https://github.com/golang/go/wiki/Modules#can-a-module-consume-a-v2-package-that-has-not-opted-into-modules-what-does-incompatible-mean).

### Why is that module in go.mod?!

A variety of commands can answer that question, with varying levels of clarity:

    $ go mod graph

will show what module requires what other module. This isn't list packages and doesn't seem like the complete picture.

    $ go mod why -m all

will dump "paragraphs" where each line is imported by the previous line.

There's a make target to turn this info into a graph:

    $ make mod-graph

