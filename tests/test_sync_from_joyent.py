# vim:fileencoding=utf-8

import json
import os

from io import BytesIO

from _pytest.monkeypatch import monkeypatch
import pytest

import tory_sync_from_joyent


@pytest.fixture
def sdc_listmachines_json_stream(sampledata):
    return BytesIO(json.dumps(sampledata['joyent']['sdc_listmachines']))


def fake_put_host(server, auth_token, host_def):
    return 200


def get_fake_stdin():
    return BytesIO(json.dumps(SDC_LISTMACHINES))


def setup():
    os.environ.clear()
    mp = monkeypatch()
    mp.setattr(tory_sync_from_joyent, '_put_host', fake_put_host)


def test_main(monkeypatch, sdc_listmachines_json_stream):
    monkeypatch.setattr('sys.stdin', sdc_listmachines_json_stream)
    ret = tory_sync_from_joyent.main(['tory-sync-from-joyent'])
    assert ret == 0
