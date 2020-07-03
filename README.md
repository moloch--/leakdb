# LeakDB

LeakDB is a tool set designed to allow organizations to build and deploy their own internal _plaintext_ "Have I Been Pwned"-like service. The LeakDB tool set can normalize, deduplicate, index, sort, and search leaked data sets on the multi-terabyte-scale, without the need to distribute large files to individual users. Once curated, LeakDB can search terabytes of data in less than a tenth of a second, and the LeakDB server exposes a simple JSON API that can be queried using the command line client or any http client. It can be deployed in a serverless configuration with a BigQuery backend (no indexes), or as an offline/traditional server with indexes.

LeakDB uses a configurable [bloom filter](https://en.wikipedia.org/wiki/Bloom_filter) to remove duplicate entires, sorts indexes using [external parallel quicksort](https://en.wikipedia.org/wiki/External_sorting) (i.e., memory constrained) with a [k-way binary tree merge](https://en.wikipedia.org/wiki/K-way_merge_algorithm), and [binary tree search](https://en.wikipedia.org/wiki/Binary_tree) to find entries in the index.

![Go](https://github.com/moloch--/leakdb/workflows/Go/badge.svg?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/moloch--/leakdb)](https://goreportcard.com/report/github.com/moloch--/leakdb) [![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

### Bring Your Own Data

__⚠️ Important:__ You must provide your own data, these tools are provided as-is and not distributed with any data sets public or otherwise.

### Download

Download the latest [release](https://github.com/moloch--/leakdb/releases)

### Usage

See the [wiki](https://github.com/moloch--/leakdb/wiki) for detailed setup and usage.

### Compile From Source

Just run `make <platform>`, files will be put in `./bin`. The client, curator, and server are pure Go and should support any valid Go compiler target. The serverless binary is Linux only, since AWS Lambda only supports Linux. The easiest way to compile the Windows binaries is to cross-compile them from a better operating system like Linux or MacOS.

For example:
* `make macos`
* `make linux`
* `make windows`
