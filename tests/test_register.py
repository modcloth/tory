# vim:fileencoding=utf-8

import json
import os

from io import BytesIO

from _pytest.monkeypatch import monkeypatch
import pytest

import tory_register


def fake_put_host(server, auth_token, host_def):
    return 200


def fake_get_local_ipv4():
    return '127.0.0.1'


def setup():
    os.environ.clear()
    mp = monkeypatch()
    mp.setattr(tory_register, '_put_host', fake_put_host)
    mp.setattr(tory_register, '_get_local_ipv4', fake_get_local_ipv4)


def test_main():
    ret = tory_register.main(['tory-register'])
    assert ret == 0
