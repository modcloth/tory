PYTEST ?= py.test
PYTEST_FLAGS ?= \
	--cov-report term-missing \
	--cov tory_sync_from_joyent \
	--cov tory_register \
	--cov tory_inventory \
	--pep8 -rs --pdb

all:
	$(PYTEST) $(PYTEST_FLAGS) tests/
