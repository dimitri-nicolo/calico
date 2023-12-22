# Calico Enterprise post-release tests

This suite of tests validates the release of Calico Enterprise. The test
suite is divided into multiple stages which can be validated independently
(as release phases are completed) or which can be run all at once.

## I just want to run the tests!

TL;DR:

1. Check the current default variables: `make docker-show-variables`
2. Determine any variables you need to change (e.g. versions) and
   export them (or provide them on the `make` command-line)
4. Run `make docker-test_all`

### But I don't want to run them in Docker!

*Run them in Docker.* Running them on your local system can introduce errors due to
library versions, Python versions, paths, etc. which can affect the reliability
of the tests. Unless you're writing new tests you shouldn't run the tests outside
of Docker, and even then you should consider it.

## Test structure

### Layout

Tests are located in this directory, named `test_*.py`. Supplemental functionality
is located in the `lib/` directory, including utility functions, classes, and so on.

### Division of Responsibilities

Currently, the goal is to keep logic and most work in the `lib` functions, and to restrict
the tests to validating whether or not things are as expected. For example, the functionality
and logic for fetching a release archive is stored in `lib.aws.py:S3.get_release_archive_metadata()`,
including what the paths should be and how to fetch it. This functionality is then called by
the tests themselves to receive (essentially) a yes/no on whether that file exists/could be
accessed. This keeps the tests themselves simple, and makes it easier/clearer what the tests
are trying to accomplish. It also makes it easier to reuse functionality by keeping it in the
same place.

### Markers

Pytest allows developers to "mark" tests with certain flags or functionalities through Python's
decorators functionality. Current markers available can be listed by pytest via `pytest --markers`,
but the custom markers we use for our tests (as of this writing) are as follows:

```
@pytest.mark.quay: the test interacts with quay.io (likely via the API)

@pytest.mark.docker: the test interacts with docker containers or the docker cli

@pytest.mark.aws: the test interacts with AWS resources

@pytest.mark.github: the test makes API or HTTP requests to Github

@pytest.mark.calico_release_cut: The test validates images re-tagged or created as part of the release cut stage

@pytest.mark.operator_release: The test validates images re-tagged or created as part of the operator release stage

@pytest.mark.postrelease_artifacts: the test validates artifacts built and uploaded at the end of a release cut
```

Tests during a run can be filtered down to specific markers with `pytest -m <marker>`; for example, `pytest -m github`
or `pytest -m 'not aws'`. In order to run a specific marker using this test suite, use the `test_marker` rule via Docker
(see "Testing custom markers" below).

## Running the tests

The tests can be most easily run via the included `Makefile` and its rules.

### Basic `Makefile` rules

Currently, there are a few major top-level make targets which are most likely to be useful for testing:

* `test_calico_release_cut` - Execute tests which should pass after a release cut is completed

* `test_operator_release` - Execute tests which should pass after an operator version is released

* `test_postrelease_artifacts` - Execute tests which should pass once postrelease artifacts are published

* `test_all` - Run all tests

### Testing custom markers

To run tests for a specific marker, you can use the `test_marker-%` Makefile rule. For example,
`make docker-test_marker-github` will run (effectively) `pytest -m github` inside of a clean
Docker container.

### Running in Docker

By running the `docker-%` rule, you can execute any existing makefile rule, including the custom
markers rule, inside of a Docker container. For example, `make docker-test_calico_release_cut`
will run the `test_calico_release_cut` make target inside of a docker container.

In all cases, the docker container will first install the test requirements from `requirements.txt`
automatically. To override versions, see the instructions in "I just want to run the tests!" above.
