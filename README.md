# deep-packet-inspection

It starts a typha client to get updates on resources required by tigera-dpi daemonset.

## Building and testing

To build this locally, use one of the following commands:

```
make image
```

or

```
make ci
```

## Adding and running test

To run all tests

```
make fv ut
```

FV test runs against real k8s, they should include "[FV]" in the test name.