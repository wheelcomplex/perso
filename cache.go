package main

import (
	"fmt"
	"time"
	"sort"
	"strings"
	"net/mail"
)

type cacheString map[string][]mailID
type cacheTime map[time.Time][]mailID

type caches struct {
	str map[string]cacheString
	time map[string]cacheTime
	cancelCh chan struct{}
	mailCh chan cacheMail
	requestCh chan cacheRequest
	sweepCh chan []mailID
}

type cacheRequest struct {
	name string
	time time.Time
	str  string
	submatch bool
	lower bool
	data chan<- []string
}

type cacheMail struct {
	id mailID
	headers mail.Header
}

func newCacheRequest() *cacheRequest {
	return &cacheRequest{
		data: make(chan []string),
	}
}

func makeCaches() *caches {
    return &caches{
        str: make(map[string]cacheString),
        time: make(map[string]cacheTime),
	    cancelCh: make(chan struct{}),
		mailCh: make(chan cacheMail),
		requestCh: make(chan cacheRequest),
		sweepCh: make(chan []mailID),
	}
}

func (c *caches) initCachesString(name string) {
	c.str[name] = make(map[string][]mailID)
}

func (c *caches) initCachesTime(name string) {
	c.time[name] = make(map[time.Time][]mailID)
}

func (m *cacheMail) getHeader(h string) []string {
	for header, v := range m.headers {
		header = strings.ToLower(header)
		if header == h {
			return v
		}
	}

	return nil
}

func (c *caches) indexMail(m cacheMail) {
	for name := range c.str {
		// TODO: Special case for 'to' and 'from': parse addresses
		if val := m.getHeader(name); val != nil {
			for _, v := range val {
				c.addString(name, v, m.id)
			}
		}
	}
}

func (c *caches) addString(name string, key string, value mailID) {
    if _, found := c.str[name][key]; !found {
        c.str[name][key] = make([]mailID, 0)
    }

    c.str[name][key] = append(c.str[name][key], value)
}

func (c *caches) getString(name string, key string) []mailID {
    return c.str[name][key]
}

func (c *caches) addTime(name string, key time.Time, value mailID) {
	if _, found := c.time[name][key]; !found {
        c.time[name][key] = make([]mailID, 0)
    }

    c.time[name][key] = append(c.time[name][key], value)
}

func (c *caches) getKeysString(name string) []string {
	keys := make([]string, 0)

	for k := range c.str[name] {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	return keys
}

func (c *caches) getKeysTime(name string) []time.Time {
	keys := make([]time.Time, 0)

	for k := range c.time[name] {
		keys = append(keys, k)
	}

	sort.Sort(timeSlice(keys))

	return keys
}

func slicePresent(m mailID, elements []mailID) bool {
	for _, e := range elements {
		if e == m {
			return true
		}
	}
	return false
}

func sliceDiff(a, b []mailID) []mailID {
	r := make([]mailID, 0)

	for _, e := range a {
		if !slicePresent(e, b) {
			r = append(r, e)
		}
	}

	return r
}

func (c *caches) sweepCacheStr(name string, removedIDs []mailID) {
	for k := range c.str[name] {
		sort.Sort(mailIDSlice(c.str[name][k]))
		c.str[name][k] = sliceDiff(c.str[name][k], removedIDs)
	}
}

func (c *caches) sweepCacheTime(name string, removedIDs []mailID) {
	for k := range c.str[name] {
		sort.Sort(mailIDSlice(c.str[name][k]))
		c.str[name][k] = sliceDiff(c.str[name][k], removedIDs)
	}
}

func (c *caches) sweep(removedIDs []mailID) {
	for k := range c.str {
		c.sweepCacheStr(k, removedIDs)
	}

	for k := range c.time {
		c.sweepCacheTime(k, removedIDs)
	}
}

func (c *caches) cancel() {
	c.cancelCh <- struct{}{}
}

func (c *caches) request(r cacheRequest) {
	c.requestCh <- r
}

func (c *caches) index(id mailID, headers mail.Header) {
	c.mailCh <- cacheMail{ id: id, headers: headers }
}

func (c *caches) run() {
	for {
		select {
		case <-c.cancelCh:
			return
		case m := <-c.mailCh:
			// Index this mail
			fmt.Printf("Indexing %s\n", m.id)
			c.indexMail(m)
		case <-c.requestCh:
			// Send back []mailID
		case mailIDs := <-c.sweepCh:
			c.sweep(mailIDs)
		}
	}
}