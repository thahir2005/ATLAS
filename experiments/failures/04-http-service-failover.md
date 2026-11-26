# Experiment 04 — HTTP Service Failover

## Objective

Test real HTTP application connectivity between Mumbai and Singapore
during a direct regional network-link failure.

## Environment

- Three Linux network namespaces
- Mumbai: 10.10.1.1
- Singapore: 10.10.2.1
- Frankfurt: 10.10.3.1
- Regional API port: TCP 8080
- Dynamic routing: FRRouting eBGP

## Baseline

HTTP requests from Mumbai to all three regional APIs returned valid
JSON responses with HTTP 200.

## Experiment A — Automatic source address

The direct Mumbai–Singapore link was disabled while HTTP requests
were continuously sent to Singapore.

Configuration:
- Request interval: approximately 100 ms
- Connection timeout: 1 second
- Link failure duration: approximately 5 seconds

Results:

| Metric | Result |
| --- | ---: |
| Total requests | 51 |
| Successful requests | 45 |
| Failed requests | 6 |
| Failure rate | 11.76% |

HTTP requests failed during the transition and resumed after
the direct link was restored.

## Experiment B — Explicit regional loopback source

The test was repeated with the Mumbai regional loopback address
explicitly selected as the source:

    curl --interface 10.10.1.1 http://10.10.2.1:8080/region

After the direct link was disabled, both routes selected Frankfurt:

Mumbai forward route:

    10.10.2.1 via 172.20.31.1 dev fm0

Singapore return route:

    10.10.1.1 via 172.20.23.2 dev sf0

Results:
- Alternate routes detected successfully.
- Three HTTP requests succeeded.
- All three returned the Singapore API response.
- Final health check succeeded after link restoration.

## Analysis

The first test used an automatically selected source address.
The second test explicitly used the Mumbai regional loopback.

The results demonstrate that source-address selection and return-path
routing are important when validating application connectivity.

The initial HTTP failure rate is specific to that test configuration.
The corrected test confirms that HTTP traffic can traverse the
Frankfurt alternate path.

## Limitations

The corrected experiment validates three requests after route
convergence. It does not establish zero packet loss or uninterrupted
application availability throughout the entire failure transition.

All regions run inside one Colima VM on one physical Mac. This is
a network and application resilience simulation, not independent
geographic infrastructure.
