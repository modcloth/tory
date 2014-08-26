# vim:fileencoding=utf-8

import os

from _pytest.monkeypatch import monkeypatch

import tory_inventory


def fake_fetch_inventory(url):
    return '{"_meta":{"hostvars":{}}}'


def setup():
    os.environ.clear()
    mp = monkeypatch()
    mp.setattr(tory_inventory, '_fetch_inventory', fake_fetch_inventory)


def test_main():
    ret = tory_inventory.main(['tory-inventory'])
    assert ret == 0
