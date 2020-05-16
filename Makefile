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

API_CLIENT_OUTFILE=leakdb
API_SERVER_OUTFILE=leakdb-server
API_LAMDBA_OUTFILE=leakdb-lambda
CURATOR_OUTFILE=leakdb-curator

.bin:
	mkdir -p ./bin/

bin/leakdb-curator: PKG=github.com/moloch--/leakdb/internal/curator
bin/leakdb-curator: .bin
	GOOS=$(GOOS) $(GO) build $(LDFLAGS) -o ./bin/$(CURATOR_OUTFILE) ./cmd/curator

bin/leakdb: PKG=github.com/moloch--/leakdb/internal/api-client
bin/leakdb: .bin
	GOOS=$(GOOS) $(GO) build $(LDFLAGS) -o ./bin/$(API_CLIENT_OUTFILE) ./cmd/api-client

bin/leakdb-server: PKG=github.com/moloch--/leakdb/internal/api-server
bin/leakdb-server: .bin
	GOOS=$(GOOS) $(GO) build $(LDFLAGS) -o ./bin/$(API_SERVER_OUTFILE) ./cmd/api-server

bin/leakdb-lambda: .bin
	cd aws/lambda/leakdb && GOOS=linux go build -o ./$(API_LAMDBA_OUTFILE) .
	zip ./bin/$(API_LAMDBA_OUTFILE).zip ./aws/lambda/leakdb/$(API_LAMDBA_OUTFILE)

#
# Curator Builds
#
.PHONY: macos
macos: GOOS=darwin
macos: bin/leakdb-curator

.PHONY: linux
linux: GOOS=linux
linux: bin/leakdb-curator

.PHONY: windows
windows: GOOS=windows
windows: CURATOR_OUTFILE=leakdb-curator.exe
windows: bin/leakdb-curator

#
# API Client Builds
#
.PHONY: macos
macos: GOOS=darwin
macos: bin/leakdb

.PHONY: linux
linux: GOOS=linux
linux: bin/leakdb

.PHONY: windows
windows: GOOS=windows
windows: CURATOR_OUTFILE=leakdb.exe
windows: bin/leakdb

#
# API Server Builds
#
.PHONY: macos
macos: GOOS=darwin
macos: bin/leakdb-server

.PHONY: linux
linux: GOOS=linux
linux: bin/leakdb-server

.PHONY: windows
windows: GOOS=windows
windows: CURATOR_OUTFILE=leakdb-server.exe
windows: bin/leakdb-server

#
# API Lambda Builds
#
.PHONY: lambda
lambda: bin/leakdb-lambda

clean:
	rm -f ./bin/leakdb*
