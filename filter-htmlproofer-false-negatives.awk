BEGIN {
    current_header = 0;
    filtered_issue_count = 0;
}

# Avoid immediately printing the header line for each file where
# htmlproofer found errors.  We only want to print this if any of the
# errors in that file are ones that we don't want to filter out.
#
# The initial grouped bit is to allow for an initial ANSI color code,
# which htmlproofer emits when it's outputting to an ANSI terminal.
/^(.\[31m)?- \/_site/ {
    current_header = $0;
    next;
}

# Filter out problems with 'latest' pages.  These are only generated
# when we publish the CNX docs, so it's expected that they are
# currently missing.
/\/latest/ {
    next;
}

# Ditto for 'v2.0'.
/\/v2.0/ {
    next;
}

# The " * " prefix indicates an error (that we haven't filtered out).
# Print out the header with the name of the relevant file, if we
# haven't already done that, and count the error.
#
# As above, the inital group is a possible ANSI color code.
/^(.\[31m)?  \*  / {
    if (current_header != 0) {
	print current_header;
	current_header = 0;
    }
    filtered_issue_count++;
}

{
    print;
}

END {
    print "Number of issues:", filtered_issue_count;
}
