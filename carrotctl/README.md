# carrotctl

Tigera licensing and entitlement commandline tool

![carrotctl](./carrabbit.jpg)


carrotctl can generate and retrieve licenses.

For v2.1 we basically have 2 major commands:

1. `carrotctl generate license`
2. `carrotctl retrieve license`

### Generate a new license

Spec for `carrotctl generate license` is:

```
Usage:
  carrotctl generate license [flags]

Flags:
      --debug             print debug information about the license fields
      --graceperiod int   number of nodes customer is licensed for (default 90)
  -h, --help              help for generate
      --name string       customer name
      --nodes int         number of nodes customer is licensed for
      --term int          license term
```

If none of the flags are passed then it will interactively ask the user to enter the data.

example:

With flags:
```
carrotctl generate license --name happy-carrot-inc --nodes 555 --term 365 --graceperiod 999
Confirm the license information:
Customer name:        happy-carrot-inc
Number of nodes:      555
License term (days):  365
Grace period (days):  999
Is the license information correct? [y/N]
y

Created license file 'happy-carrot-inc-license.yaml'
```

Interactive prompt:

```
carrotctl.go generate license
Enter the customer name:
lame-banana-inc
Enter number of nodes the customer is licensed for:
22
Enter the license term (in days):
69
Confirm the license information:
Customer name:        lame-banana-inc
Number of nodes:      22
License term (days):  69
Grace period (days):  90
Is the license information correct? [y/N]
y

Created license file 'lame-banana-inc-license.yaml'
```

### Retrieve a license from database

`carrotctl list license --name=boxy-box-inc` will list all key license fields for all the licenses issued for a customer name matching `boxy-box-inc*`

It will list `CustomerID` for each license issued for that customer, which can be used to retrieve the 
license with `carrotctl retrieve license --cid=<customer-id>` command.

Each license has a unique customer ID (UUID), even if it is for the same customer.

Example:

List all the licenses issued for customer `team-rocket-inc`
```
carrotctl list license --name="team-rocket-inc"

NAME                CUSTOMERID          TERM        NODES
team-rocket-inc     ash212453efsdf      365         100
team-rocket-inc-ish meow15dsd3424f      10          10
```

Re-generate the license.yaml for the second license from database:

```
carrotctl retrieve license --cid=meow15dsd3424f

Created license file 'team-rocket-inc-ish-license.yaml'
```