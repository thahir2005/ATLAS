package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"atlas.local/control-plane/controller/internal/consensus"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ATLAS_LEASE_ERROR: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	region := flag.String("region", "", "controller region name")
	endpointsFlag := flag.String(
		"endpoints",
		"http://10.10.1.1:2379,http://10.10.2.1:2379,http://10.10.3.1:2379",
		"comma-separated etcd client endpoints",
	)
	ttl := flag.Int("ttl", 10, "leadership lease TTL in seconds")
	resourceKey := flag.String(
		"key",
		"/atlas/resources/active-region",
		"protected resource key",
	)
	firstValue := flag.String("value", "", "value written after acquiring leadership")
	delay := flag.Duration(
		"delay-before-second-write",
		0,
		"delay before attempting a second protected write",
	)
	secondValue := flag.String(
		"second-value",
		"stale-write-attempt",
		"value used by the delayed write",
	)
	hold := flag.Duration(
		"hold-after-first-write",
		0,
		"how long to retain leadership after the first write",
	)
	flag.Parse()

	if strings.TrimSpace(*region) == "" {
		return errors.New("--region is required")
	}
	if *ttl < 3 {
		return errors.New("--ttl must be at least 3 seconds")
	}
	if *firstValue == "" {
		*firstValue = *region + "-leader"
	}

	endpoints := strings.Split(*endpointsFlag, ",")
	for i := range endpoints {
		endpoints[i] = strings.TrimSpace(endpoints[i])
		if endpoints[i] == "" {
			return errors.New("etcd endpoint list contains an empty endpoint")
		}
	}

	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("create etcd client: %w", err)
	}
	defer client.Close()

	manager, err := consensus.NewManager(client, consensus.Config{
		TTL: *ttl,
	})
	if err != nil {
		return fmt.Errorf("create leadership manager: %w", err)
	}

	acquireCtx, cancelAcquire := context.WithTimeout(
		context.Background(),
		45*time.Second,
	)
	defer cancelAcquire()

	fmt.Printf("%s region=%s event=ACQUIRING_LEADERSHIP\n",
		time.Now().Format(time.RFC3339Nano), *region)

	leadership, err := manager.Acquire(acquireCtx)
	if err != nil {
		return err
	}

	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := leadership.Release(releaseCtx); err != nil {
			fmt.Fprintf(os.Stderr, "RELEASE_WARNING: %v\n", err)
		}
	}()

	token := leadership.Token()

	fmt.Printf("%s region=%s event=LEADERSHIP_ACQUIRED fencing_token=%d lease_id=%d\n",
		time.Now().Format(time.RFC3339Nano), *region, token, leadership.LeaseID())

	checkCtx, cancelCheck := context.WithTimeout(context.Background(), 3*time.Second)
	isLeader, err := leadership.IsLeader(checkCtx)
	cancelCheck()
	if err != nil {
		return err
	}
	if !isLeader {
		return consensus.ErrNotLeader
	}

	writeCtx, cancelWrite := context.WithTimeout(context.Background(), 3*time.Second)
	err = leadership.FencedPut(writeCtx, *resourceKey, *firstValue)
	cancelWrite()
	if err != nil {
		return fmt.Errorf("first protected write: %w", err)
	}

	fmt.Printf("%s region=%s event=FIRST_WRITE_ACCEPTED fencing_token=%d value=%s\n",
		time.Now().Format(time.RFC3339Nano), *region, token, *firstValue)

	if *hold > 0 {
		fmt.Printf("%s region=%s event=HOLDING_LEADERSHIP duration=%s\n",
			time.Now().Format(time.RFC3339Nano), *region, *hold)
		time.Sleep(*hold)
	}

	if *delay > 0 {
		fmt.Printf("%s region=%s event=WAITING_BEFORE_SECOND_WRITE duration=%s\n",
			time.Now().Format(time.RFC3339Nano), *region, *delay)
		time.Sleep(*delay)

		secondCtx, cancelSecond := context.WithTimeout(context.Background(), 5*time.Second)
		err = leadership.FencedPut(secondCtx, *resourceKey, *secondValue)
		cancelSecond()

		if errors.Is(err, consensus.ErrNotLeader) {
			fmt.Printf("%s region=%s event=SECOND_WRITE_REJECTED fencing_token=%d reason=stale_leadership\n",
				time.Now().Format(time.RFC3339Nano), *region, token)
			return consensus.ErrNotLeader
		}
		if err != nil {
			return fmt.Errorf("second protected write: %w", err)
		}

		fmt.Printf("%s region=%s event=SECOND_WRITE_ACCEPTED fencing_token=%d value=%s\n",
			time.Now().Format(time.RFC3339Nano), *region, token, *secondValue)
	}

	fmt.Printf("%s region=%s event=EXPERIMENT_COMPLETE fencing_token=%d\n",
		time.Now().Format(time.RFC3339Nano), *region, token)

	return nil
}
