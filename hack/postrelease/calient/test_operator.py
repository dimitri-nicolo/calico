#!/usr/bin/env python3

"""
Tests for Operator, including testing images, tags, releases.
"""


import pytest

import github

from variables import defaults

from lib.utils import gh_repo_has_tag
from lib.operator import Operator
from lib.quay import Quay

from lib.exceptions import UndefinedVariableError

@pytest.fixture(name="github_obj")
def create_github_obj():
    """
    Create and return a Github object with the standard
    authorization token.
    """
    try:
        return github.Github(auth=github.Auth.Token(defaults.GITHUB_TOKEN))
    except UndefinedVariableError:
        pytest.fail("Required variable GITHUB_TOKEN was not specified")


@pytest.fixture(name="operator_obj")
def create_operator_obj():
    """
    Create and return an Operator object for the
    current operator version.
    """
    return Operator(defaults.OPERATOR_VERSION, project=defaults.OPERATOR_IMAGE)


@pytest.fixture(name="github_operator_repo")
def get_github_operator_repo(github_obj, operator_obj):
    """
    Create and return a Repo object from Github
    based on the current Operator object's settings.
    """
    return github_obj.get_repo(operator_obj.project)


@pytest.fixture(name="operator_arches")
def fetch_operator_arches(operator_obj):
    """
    Return the architectures that the Operator
    manifest was build with.
    """
    return operator_obj.arches

@pytest.mark.github
@pytest.mark.operator_release
def test_operator_tag_exists(operator_obj, github_operator_repo):
    """
    Test that the Github repository has a tag for the Operator's
    current version.
    """
    if not gh_repo_has_tag(github_operator_repo, operator_obj.version):
        pytest.fail(
            f"Operator tag {operator_obj.obj_version} is not published on Github"
        )


@pytest.mark.github
@pytest.mark.operator_release
def test_operator_release_exists(operator_obj, github_operator_repo):
    """
    Test that the Github repository has a release for the Operator's
    current version.
    """
    try:
        github_operator_repo.get_release(operator_obj.version)
    except github.UnknownObjectException:
        pytest.fail(
            f"Operator release {operator_obj.version} is not published on Github"
        )

@pytest.mark.quay
@pytest.mark.operator_release
def test_operator_manifest_exists(operator_obj):
    """
    Validate that the operator manifest exists. the `docker manifest create`
    command will validate that component images (e.g. -amd64, -s390x) exist
    so we don't need to validate them individually.
    """
    try:
        Quay(token=defaults.QUAY_TOKEN).get_tag(operator_obj.image_name, operator_obj.version)
    except UndefinedVariableError as exc:
        pytest.fail(f"Required variable {exc.args[0]} not defined")
