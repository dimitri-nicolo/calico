# Chart layout

The chart uses lexicographic ordering to organize the manifests. All resources get their own file
with a name of the following form:

	XX-<kind>-<name>

`XX` is a numeric value to help group resources together.

- `00`: Namespaces
- `01`: Custom resource definitions
- `02`: Tigera operator resources
- `03`: Elastic operator resources
- `04`: Prometheus operator resources
