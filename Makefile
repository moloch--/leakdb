# ---------------------------------------------------------------------
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.

# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.

# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.
# ---------------------------------------------------------------------

GO ?= go

#
# Check that commands exist
#
EXECUTABLES = git go date
K := $(foreach exec,$(EXECUTABLES),\
        $(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH")))

#
# Version info
#
VERSION = 1.0.0
COMPILED_AT = $(shell date +%s)
GIT_DIRTY = $(shell git diff --quiet|| echo 'Dirty')
GIT_COMMIT = $(shell git rev-parse HEAD)
LDFLAGS = -ldflags "-s -w \
	-X $(PKG).Version=$(VERSION) \
	-X $(PKG).CompiledAt=$(COMPILED_AT) \
	-X $(PKG).GitCommit=$(GIT_COMMIT) \
	-X $(PKG).GitDirty=$(GIT_DIRTY)"

bin:
	mkdir -p ./bin/

leakdb-curator: bin
	GOOS=$(GOOS) $(GO) build $(LDFLAGS) -o ./bin/leakdb-curator ./cmd/curator

.PHONY: macos
macos:
	$(eval GOOS := darwin)
	$(MAKE) bin/leakdb-curator

clean:
	rm -f ./bin/leakdb
	rm -f ./bin/leakdb-curator
