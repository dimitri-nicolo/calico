#!/usr/bin/env python3
"""
Generic utility functions and classes for testing
"""

import os
import sys
import subprocess

from typing import Any

from rich.table import Table
from rich import print as print_rich
import requests

try:
    from log_config import log
    from exceptions import UndefinedVariableError
except ImportError:
    from .log_config import log
    from .exceptions import UndefinedVariableError


# Simple formatting of URLs
def release_url(project_slug, release_name):
    """
    Given a Github project slug and a release name, return the release's URL.
    """
    return f"https://github.com/{project_slug}/releases/tag/{release_name}"


def tag_url(project_slug, release_name):
    """
    Given a Github project slug and a tag name, return the tag's URL.
    """
    return f"https://github.com/{project_slug}/tree/{release_name}"


def http_get_is_200(url):
    """
    Given a URL, return True if we can fetch the URL successfully
    with no session information or authentication.
    """
    resp = requests.get(url, timeout=5)
    return resp.status_code == 200


def get_version_from_specfile(filepath):
    """
    Given a path to an RPM specfile, use `rpmspec` to parse the
    file and return the calculated PROVIDEVERSION field.
    """
    rpmspec_path = get_path_to_binary("rpmspec")
    cmd = (rpmspec_path, "-q", "--queryformat", "%{PROVIDEVERSION}", filepath)
    result = subprocess.check_output(cmd).decode().strip()
    return result


def get_path_to_binary(binary):
    """
    Use `which` (which is probably /usr/bin/which but could be
    the shell builtin `which`) to determine which binary to use
    (by calling `which <binary>`)
    """
    cmd = ("which", binary)
    result = subprocess.check_output(cmd).decode().strip()
    return result


def gh_repo_has_tag(repo, tag_name, all_tags=False):
    """
    Given a repository and a tag_name, check to see if the
    repository has a tag named `tag_name`.

    We can't just use `repo.get_git_tag()` like normal people
    because that only supports annotated tags, so we have to
    just fetch a list of tags and iterate like chumps.

    By default we only check the first page of results, which
    should be more than enough for any recent tag.. Pass in
    `all_tags=True` to check every tag, which will take a
    long time and possibly get you rate-limited.
    """
    if all_tags:
        search_space = repo.get_tags()
        log.warning(
            "Searching for tag %s in %s, this could take a while and get you rate-limited",
            tag_name,
            search_space.totalCount,
        )
    else:
        search_space = repo.get_tags().get_page(0)
    for tag in search_space:
        if tag.name == tag_name:
            return True
    return False


# Gets a variable from the environment, else
# from a predefined list of defaults, else
# raises an exception
class EnvironmentDefaults:
    """
    Fetches a variable from the variable store. Currently supports
    two tiers:

    1. Is the variable specified in the environment? return that.
    2. Is the variable specified in the defaults list that was passed
       in? return that.
    3. Otherwise, raise an exception.

    The 'defaults list' is meant to be a compiled-in set of example
    defaults. Down the road, we will support specifying a .env file
    which will override those, and then environment variables will
    override those again.
    """

    def __init__(self, defaults):
        self.defaults = defaults

    def show_defaults_print(self):
        print("Current variable defaults:")
        for key, val in sorted(self.defaults.items()):
            current_val = getattr(self, key)
            print(f"  {key}")
            print(f"    Default: {val}")
            if val != current_val:
                print(f"    Current: {current_val}")
        print("Variables can be overridden in the environment (export AWS_PROFILE=...)")

    def show_defaults_table(self):
        table = Table(
            "Variable",
            "Default value",
            "Current value",
            title="Current variable defaults",
            caption="Variables can be overridden in the environment (export AWS_PROFILE=...)")
        for key, val in sorted(self.defaults.items()):
            current_val = getattr(self, key)
            if val != current_val:
                val = f"[red]{val}[/red]"
                current_val = f"[green]{current_val}[/green]"
            table.add_row(key, val, current_val)
        print_rich(table)

    @property
    def variable_names(self):
        yield from sorted(self.defaults)

    def show_variables(self):
        if sys.stdout.isatty():
            self.show_defaults_table()
        else:
            self.show_defaults_print()

    def __getitem__(self, __name: str) -> Any:
        if __name in os.environ:
            return os.environ[__name]
        if __name in self.defaults:
            return self.defaults[__name]
        raise UndefinedVariableError(__name)

    def __getattr__(self, __name: str) -> Any:
        return self[__name]


# Simple auth handler for bearer tokens
class HTTPBearerAuth(requests.auth.AuthBase):  # pylint: disable=too-few-public-methods
    """
    A requests.auth class to handle HTTP authentication
    using HTTP Bearer tokens.
    """

    def __init__(self, token):
        self.token = token

    def __call__(self, r):
        r.headers["Authorization"] = "Bearer " + self.token
        return r
