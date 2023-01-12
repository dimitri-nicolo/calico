# Copyright (c) 2022-2023 Tigera, Inc. All rights reserved.

import logging
import re

from tests.k8st.test_base import Container, Pod, TestBase
from tests.k8st.utils.utils import DiagsCollector, calicoctl, kubectl, run, retry_until_success

_log = logging.getLogger(__name__)

class TestExternalNetwork(TestBase):
    def setUp(self):
        super(TestExternalNetwork, self).setUp()

    def tearDown(self):
        super(TestExternalNetwork, self).tearDown()
