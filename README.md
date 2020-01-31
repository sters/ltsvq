# ltsvq

ltsvq is LTSV Queryer that written in Go.

## Installation

```bash
go get github.com/sters/ltsvq
```

## Usage

```bash
cat example.ltsv | ltsvq -q "select * from ltsv where host like '192%'"
```

In real case, for example: Find suspicious remote_addr.

```bash
zcat /var/log/nginx/*.gz | ./ltsvq -q 'select count(uri), uri, remote_addr from ltsv where status IN ("400", "401", "403", "404") group by uri order by count(uri) desc' | less
```
