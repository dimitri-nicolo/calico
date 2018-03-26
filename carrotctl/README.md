# carrotctl

Tigera licensing and entitlement commandline tool

![carrotctl](./carrabbit.jpg) <!-- .element height="20%" width="20%" -->


carrotctl can generate and retrieve licenses.

For v2.1 we basically have 2 major features:

1. Generate license (`carrotctl generate`)
2. Retrieve license (`carrotctl list` and `carrotctl retrieve`)

## Generate a new license

### Usage

Spec for `carrotctl generate` is:

```
Usage:
  carrotctl generate [flags]

Aliases:
  generate, gen, gen-lic, generate-license, make-me-a-license

Flags:
      --certificate string   Licensing intermediate certificate path (default "./tigera.io_certificate.pem")
  -c, --customer string      Customer name
      --debug                Print debug logs while generating this license
  -e, --expiry string        License expiration date in MM/DD/YYYY format. Expires at the end of the day cluster local timezone.
  -g, --graceperiod int      Number of days the cluster will keep working after the license expires (default 90)
  -h, --help                 help for generate
  -n, --nodes int            Number of nodes customer is licensed for. If not specified, it'll be an unlimited nodes license.
      --signing-key string   Private key path to sign the license content (default "./tigera.io_private_key.pem")
```

If none of the flags are passed then it will interactively ask the user to enter the data.

### Examples

#### Default fields:

```
carrotctl generate --customer happy-carrot-inc --expiry 3/14/2022
Confirm the license information:
_________________________________________________________________________
Customer name:                  happy-carrot-inc
Number of nodes:                Unlimited (site license)
License term expiration date:   2022-03-14 23:59:59 -0700 PDT
Features:                       [cnx all]
Checkin interval:               Offline license
Grace period (days):            90
License ID (auto-generated):    b2e8c974-a987-4004-b1bc-a739e6ad6272
________________________________________________________________________

Is the license information correct? [y/N]
y

Created license file 'happy-carrot-inc-license.yaml'
```


## Retrieve a license from database

`carrotctl list --name=boxy-box-inc` will list all key license fields for all the licenses issued for a customer name matching `boxy-box-inc*`

It will list `CustomerID` for each license issued for that customer, which can be used to retrieve the
license with `carrotctl retrieve --cid=<customer-id>` command.

Each license has a unique customer ID (UUID), even if it is for the same customer.

### Example

- List all the licenses issued for customer `team-rocket-inc`

```
carrotctl list --name="team-rocket-inc"

NAME                CUSTOMERID          TERM        NODES
team-rocket-inc     ash212453efsdf      3/14/2030   100
team-rocket-inc-ish meow15dsd3424f      1/1/2019    10
```

- Re-generate the license.yaml for the second license from database:

```
carrotctl retrieve --cid=meow15dsd3424f

Created license file 'team-rocket-inc-ish-license.yaml'
```

# Building

## DB setup

To develop the tool, you'll need to set up a suitable license database to test against.
Do NOT run on the official AWS instance: it will interact with the real license database.

```
# Install mariadb; you may need to consult your distribution's instructions.
pacman -Syu mysql

# Create the tables and user
```
mysql -u root -p < datastore/db.sql
```

## Building

With dep installed (`go get -u github.com/golang/dep/cmd/dep`), run the following.

```
dep ensure
go build -o dist/carrotctl ./carrotctl
```

## Testing

You can run generate like the following.  It'll pick up the certificates in the repo.
```
dist/carrotctl generate -c tigera -e 01/01/2019 -n 10
```
