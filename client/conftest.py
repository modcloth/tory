# vim:fileencoding=utf-8

import json
import os
import sys

import pytest


HERE = os.path.dirname(os.path.abspath(__file__))


@pytest.fixture
def top_dir():
    return HERE


@pytest.fixture
def sampledata(top_dir):
    return dict(
        joyent=dict(
            sdc_listmachines=_load_json(top_dir, 'joyent-sdc-listmachines'),
        ),
        ec2=dict(
            inventory=_load_json(top_dir, 'ec2-inventory'),
        )
    )


def _load_json(top_dir, name):
    filename = os.path.join(top_dir, 'sampledata', '{}.json'.format(name))
    with open(filename) as infile:
        return json.load(infile)
