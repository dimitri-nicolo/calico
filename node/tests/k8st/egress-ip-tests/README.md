
# Notes about the egress tests

The egress tests reconfigure the node daemon set.  It takes a fair chunk of time to configure the daemonset at both
the start of the test and the end of the test (to put it back), so we run these tests seprately.

We don't bother putting back the config at the end of the test because the start of the test will reconfigure
everything required.
