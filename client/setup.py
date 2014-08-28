import sys

from setuptools import setup


def main():
    desc = 'client tools for the tory ansible inventory'
    setup(
        name='tory-client',
        url='https://github.com/modcloth/tory',
        author='ModCloth, Inc.',
        author_email='platformsphere+pypi@modcloth.com',
        description=desc,
        long_description=desc,
        version=_get_version(),
        classifiers=[
            'Development Status :: 4 - Beta',
            'Environment :: Console',
            'Intended Audience :: Developers',
            'Intended Audience :: System Administrators',
            'License :: OSI Approved :: MIT License',
            'Natural Language :: English',
            'Operating System :: OS Independent',
            'Programming Language :: Python :: 2.7',
            'Topic :: System :: Systems Administration',
            'Topic :: Utilities',
        ],
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
