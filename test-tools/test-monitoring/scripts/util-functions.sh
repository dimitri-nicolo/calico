#!/bin/sh
# Utility functions for apply-all.sh script.

# Returns the CPU requests for a generic prometheus instance based
# on the provided CLUSTER_SIZE.
get_prom_cpu() {
    cpu="500m"

    # If given an explicit value, then use that.
    if [ "$PROM_CPU" != "" ]; then 
	    echo $PROM_CPU
	    return 0
    fi

    if [ $CLUSTER_SIZE -gt 10 ]; then cpu="2000m"; fi
    if [ $CLUSTER_SIZE -gt 100 ]; then cpu="4000m"; fi
    if [ $CLUSTER_SIZE -gt 200 ]; then cpu="8000m"; fi
    if [ $CLUSTER_SIZE -gt 300 ]; then cpu="15000m"; fi
    if [ $CLUSTER_SIZE -gt 500 ]; then cpu="25000m"; fi

    echo $cpu
}

# Returns the memory requests for a generic prometheus instance based
# on the provided CLUSTER_SIZE.
get_prom_mem() {
    mem="2Gi"

    # If given an explicit value, then use that.
    if [ "$PROM_MEM" != "" ]; then 
	    echo $PROM_MEM
	    return 0
    fi

    if [ $CLUSTER_SIZE -gt 10 ]; then mem="8Gi"; fi
    if [ $CLUSTER_SIZE -gt 100 ]; then mem="16Gi"; fi
    if [ $CLUSTER_SIZE -gt 200 ]; then mem="32Gi"; fi
    if [ $CLUSTER_SIZE -gt 300 ]; then mem="50Gi"; fi
    if [ $CLUSTER_SIZE -gt 500 ]; then mem="100Gi"; fi

    echo $mem
}


# The some prometheus instances require a lot more memory in order 
# to function properly, so they get their own values. Expects CLUSTER_SIZE
# to be defined.
get_prom_high_mem() {
    mem="2Gi"

    # If given an explicit value, then use that.
    if [ "$PROM_HIGH_MEM" != "" ]; then 
	    echo $PROM_HIGH_MEM
	    return 0
    fi

    if [ $CLUSTER_SIZE -gt 10 ]; then mem="16Gi"; fi
    if [ $CLUSTER_SIZE -gt 100 ]; then mem="32Gi"; fi
    if [ $CLUSTER_SIZE -gt 200 ]; then mem="64Gi"; fi
    if [ $CLUSTER_SIZE -gt 300 ]; then mem="200Gi"; fi
    if [ $CLUSTER_SIZE -gt 500 ]; then mem="400Gi"; fi

    echo $mem
}

# The some prometheus instances require a lot more cpu in order 
# to function properly, so they get their own values. Expects CLUSTER_SIZE
# to be defined.
get_prom_high_cpu() {
    cpu="500m"

    # If given an explicit value, then use that.
    if [ "$PROM_HIGH_CPU" != "" ]; then 
	    echo $PROM_HIGH_CPU
	    return 0
    fi

    if [ $CLUSTER_SIZE -gt 10 ]; then cpu="4000m"; fi
    if [ $CLUSTER_SIZE -gt 100 ]; then cpu="8000m"; fi
    if [ $CLUSTER_SIZE -gt 200 ]; then cpu="15000m"; fi
    if [ $CLUSTER_SIZE -gt 300 ]; then cpu="25000m"; fi
    if [ $CLUSTER_SIZE -gt 500 ]; then cpu="60000m"; fi

    echo $cpu
}
