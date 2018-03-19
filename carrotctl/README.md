# carrotctl

Tigera licensing and entitlement commandline tool

![carrotctl](./carrabbit.jpg) <!-- .element height="20%" width="20%" -->


carrotctl can generate and retrieve licenses.

For v2.1 we basically have 2 major features:

1. Generate license (`carrotctl generate license`)
2. Retrieve license (`carrotctl list license` and `carrotctl retrieve license`)

## Generate a new license

### Usage

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
      --term string       license expiration date in MM/DD/YYYY format. Expires on that day at 23:59:59:999999999 (nanoseconds) customer cluster local timezone.
```

If none of the flags are passed then it will interactively ask the user to enter the data.

### Examples

#### With flags

```
carrotctl generate license --name happy-carrot-inc --nodes 555 --term 3/14/2029 --graceperiod 999
Confirm the license information:
Customer name:        happy-carrot-inc
Number of nodes:      555
License term expiration date:  2029-03-14 23:59:59 -0700 PDT
Grace period (days):  999
Is the license information correct? [y/N]
y

Created license file 'happy-carrot-inc-license.yaml'
```

#### Interactive prompt

```
carrotctl.go generate license
Enter the customer name:
lame-banana-inc
Enter number of nodes the customer is licensed for:
22
Enter the license expiration date (MM/DD/YYYY):
3/14/2212
Confirm the license information:
Customer name:        lame-banana-inc
Number of nodes:      22
License term expiration date:  2212-03-14 23:59:59 -0800 PST
Grace period (days):  90
Is the license information correct? [y/N]
y

Created license file 'lame-banana-inc-license.yaml'
```

## Retrieve a license from database

`carrotctl list license --name=boxy-box-inc` will list all key license fields for all the licenses issued for a customer name matching `boxy-box-inc*`

It will list `CustomerID` for each license issued for that customer, which can be used to retrieve the 
license with `carrotctl retrieve license --cid=<customer-id>` command.

Each license has a unique customer ID (UUID), even if it is for the same customer.

### Example

- List all the licenses issued for customer `team-rocket-inc`

```
carrotctl list license --name="team-rocket-inc"

NAME                CUSTOMERID          TERM        NODES
team-rocket-inc     ash212453efsdf      3/14/2030   100
team-rocket-inc-ish meow15dsd3424f      1/1/2019    10
```

- Re-generate the license.yaml for the second license from database:

```
carrotctl retrieve license --cid=meow15dsd3424f

Created license file 'team-rocket-inc-ish-license.yaml'
```