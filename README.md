# Lame DNS

Lame delegation checking at scale!

## Building

Requires go1.18+.

```shell
$ go build
```

## Running

```
Usage of ./lame-dns:
  -expected-ns string
        comma-separated list of domains which we expect nameservers to be under, findings are logged otherwise
  -list string
        comma-separated list of domain lists
  -parallel uint
        number of worker threads to use (default 10)
  -verbose
        show verbose messages
```

## Examples

```shell
$ ./lame-dns -list domain_list.txt -expected-ns googledomains.com,google.com,markmonitor.com,google
```

## Findings

Results are printed to stdout, and any logs, errors, or debug messages are printed to stderr.
You can pipe these to different files to save each independently. ex: `./lame-dns $ARGS >results.txt 2>results.log`.

Findings: 

* `ERROR: server:` an unexpected error occurred while sending the DNS query to a specific nameserver on every retry attempt
* `varying responses:` one (or more) of the nameservers did not return all of the records the other nameservers for the name returned.
* `ERROR querying authoritative:` an unexpected error occurred while sending parallel requests to all authoritative nameservers. (this error will likely also include a more specific `ERROR: server:` as well)
* `unexpected difference in nameservers:` authoritative nameservers returned different results from parent non-authoritative nameservers
  * `> extra nameservers returned by authoritative NS:` if any of the authoritative nameservers returned any new or unexpected nameservers, they will be printed here
* `lame delegation:` a lame delegation was found, meaning a domain's NS records to not point to authoritative servers
* `unexpected nameserver:` only displayed with `-expected-ns` and one of the input domains nameservers are not subdomains of `-expected-ns`


## Performance

The speed will largely depend on the argument to `-parallel`. The only real bottleneck is network latency, so this program can be extremely fast if given enough workers. However, if there are a lot of network errors, especially for any of the apex/parent/tld nameservers, then it will slow down considerably as these requests are retried.


## Verifying Findings

You can use the following dig command to roughly verify the results of this program. The `+trace` flag is similar, but not quite as through as the tests this program performs. To check for Authoritative responses, look for the `aa` flag in the dig response output.

```shell
dig +trace +question +qr +comments +nodnssec -t NS -q $DOMAIN
```
