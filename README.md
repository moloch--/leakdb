# LeakDB

LeakDB is a tool set designed to allow organizations to build and deploy their own plaintext "Have I Been Pwned" service. The LeakDB tool set can normalize, deduplicate, index, sort, and search leaked data sets on the multiple terabyte scale without the need to distribute large files to individual users. Once curated, LeakDB can search terabytes of data in less than a tenth of a second, and the LeakDB server exposes a simple JSON API that can be queried using the command line client or any http client.

LeakDB uses a configurable [bloom filter](https://en.wikipedia.org/wiki/Bloom_filter) to remove duplicate entires, sorts indexes using [external quicksort](https://en.wikipedia.org/wiki/External_sorting) (i.e., memory constrained) with a [k-way binary tree merge](https://en.wikipedia.org/wiki/K-way_merge_algorithm), and [binary tree search](https://en.wikipedia.org/wiki/Binary_tree) to find entries in the index.

![Go](https://github.com/moloch--/leakdb/workflows/Go/badge.svg?branch=master) [![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

### Install

Download the latest [release](https://github.com/moloch--/leakdb/releases) or use `go get -u https://github.com/moloch--/leakdb`

### Usage

See the [wiki](https://github.com/moloch--/leakdb/wiki) for detailed usage.

### Compile From Source

Just run `make` files will be put in `./bin`

