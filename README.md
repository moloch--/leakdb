# LeakDB

LeakDB is a cost effective bring-your-own-data tool set to normalize, deduplicate, index, sort, and search leaked databases. Once curated, LeakDB can search terabytes of data in less than a tenth of a second. The LeakDB server exposes a simple JSON API, which can be queried using the command line client or any http client.

LeakDB normalizes data sets, uses a configurable [bloom filter](https://en.wikipedia.org/wiki/Bloom_filter) to remove duplicate entires, sorts indexes using [external quicksort](https://en.wikipedia.org/wiki/External_sorting) with a [k-way binary tree merge](https://en.wikipedia.org/wiki/K-way_merge_algorithm), and [binary tree search](https://en.wikipedia.org/wiki/Binary_tree) to find entries in the index.

![Go](https://github.com/moloch--/leakdb/workflows/Go/badge.svg?branch=master)

### Install

Download the latest [release](https://github.com/moloch--/leakdb/releases) or use `go get -u https://github.com/moloch--/leakdb`

### Usage

See the [wiki](https://github.com/moloch--/leakdb/wiki) for detailed usage.

### Compile From Source

Just run `make` files will be put in `./bin`

