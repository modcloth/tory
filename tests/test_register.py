# vim:fileencoding=utf-8

import json
import os

from io import BytesIO

from _pytest.monkeypatch import monkeypatch
import pytest

import tory_register


def fake_put_host(server, auth_token, host_def):
    return 200


def setup():
    os.environ.clear()
    mp = monkeypatch()
    mp.setattr(tory_register, '_put_host', fake_put_host)


def test_main():
    ret = tory_register.main(['tory-register'])
    assert ret == 0
