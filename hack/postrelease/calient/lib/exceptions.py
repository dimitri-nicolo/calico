#!/usr/bin/env python3

"""
Custom exceptions for the test suite.
"""


class UndefinedVariableError(Exception):
    """
    For when a variable (in lib.utils.EnvironmentDefaults is requested
    but unset.
    """

    def __str__(self):
        return f"The variable {self.args[0]} is not defined."


class S3ObjectNotFoundError(Exception):
    """
    An S3 object's metadata was requested but the key was not found.
    """

    def __str__(self):
        return f"Object {self.args[0]}/{self.args[1]} does not exist"


class QuayNotAuthorizedError(Exception):
    """
    We tried to get information from Quay.io (either from the API or
    via the Docker CLI) but our authentication wasn't accepted.
    """
