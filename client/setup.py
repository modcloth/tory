import sys

from setuptools import setup


def main():
    setup(
        name='tory-client',
        version=_get_version(),
        py_modules=[
            'tory_inventory',
            'tory_register',
            'tory_sync_from_joyent',
        ],
        entry_points={
            'console_scripts': [
                'tory-inventory = tory_inventory:main',
                'tory-register = tory_register:main',
                'tory-sync-from-joyent = tory_sync_from_joyent:main',
            ]
        }
    )

    return 0


def _get_version():
    with open('VERSION') as version:
        return version.read().strip()


if __name__ == '__main__':
    sys.exit(main())
