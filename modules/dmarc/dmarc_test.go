/*
 * ZDNS Copyright 2022 Regents of the University of Michigan
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not
 * use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
 * implied. See the License for the specific language governing
 * permissions and limitations under the License.
 */

package dmarc

import (
	"testing"

	"github.com/zmap/dns"
	"github.com/zmap/zdns/core"
	"gotest.tools/v3/assert"
)

type QueryRecord struct {
	core.Question
	NameServer string
}

var mockResults = make(map[string]*core.SingleQueryResult)
var queries []QueryRecord

type MockLookup struct{}

func (ml MockLookup) DoSingleDstServerLookup(r *core.Resolver, question core.Question, nameServer string, isIterative bool) (*core.SingleQueryResult, core.Trace, core.Status, error) {
	queries = append(queries, QueryRecord{question, nameServer})
	if res, ok := mockResults[question.Name]; ok {
		return res, nil, core.STATUS_NOERROR, nil
	} else {
		return &core.SingleQueryResult{}, nil, core.STATUS_NO_ANSWER, nil
	}
}

func InitTest(t *testing.T) *core.Resolver {
	mockResults = make(map[string]*core.SingleQueryResult)
	rc := core.ResolverConfig{
		ExternalNameServers: []string{"127.0.0.1"},
		LookupClient:        MockLookup{}}
	r, err := core.InitResolver(&rc)
	assert.NilError(t, err)

	return r
}

func TestDmarcLookup_Valid_1(t *testing.T) {
	resolver := InitTest(t)
	mockResults["_dmarc.zdns-testing.com"] = &core.SingleQueryResult{
		Answers: []interface{}{
			core.Answer{Name: "_dmarc.zdns-testing.com", Answer: "some TXT record"},
			core.Answer{Name: "_dmarc.zdns-testing.com", Answer: "v=DMARC1; p=none; rua=mailto:postmaster@censys.io"}},
	}
	dmarcMod := DmarcLookupModule{}
	dmarcMod.CLIInit(nil, &core.ResolverConfig{}, nil)
	res, _, status, _ := dmarcMod.Lookup(resolver, "_dmarc.zdns-testing.com", "")
	assert.Equal(t, queries[0].Class, uint16(dns.ClassINET))
	assert.Equal(t, queries[0].Type, dns.TypeTXT)
	assert.Equal(t, queries[0].Name, "_dmarc.zdns-testing.com")
	assert.Equal(t, queries[0].NameServer, "127.0.0.1")

	assert.Equal(t, core.STATUS_NOERROR, status)
	assert.Equal(t, res.(Result).Dmarc, "v=DMARC1; p=none; rua=mailto:postmaster@censys.io")
}

func TestDmarcLookup_Valid_2(t *testing.T) {
	resolver := InitTest(t)
	mockResults["_dmarc.zdns-testing.com"] = &core.SingleQueryResult{
		Answers: []interface{}{
			core.Answer{Name: "_dmarc.zdns-testing.com", Answer: "some TXT record"},
			// Capital V in V=DMARC1; should pass
			core.Answer{Name: "_dmarc.zdns-testing.com", Answer: "V=DMARC1; p=none; rua=mailto:postmaster@censys.io"}},
	}
	dmarcMod := DmarcLookupModule{}
	dmarcMod.CLIInit(nil, &core.ResolverConfig{}, nil)
	res, _, status, _ := dmarcMod.Lookup(resolver, "_dmarc.zdns-testing.com", "")
	assert.Equal(t, queries[0].Class, uint16(dns.ClassINET))
	assert.Equal(t, queries[0].Type, dns.TypeTXT)
	assert.Equal(t, queries[0].Name, "_dmarc.zdns-testing.com")
	assert.Equal(t, queries[0].NameServer, "127.0.0.1")

	assert.Equal(t, core.STATUS_NOERROR, status)
	assert.Equal(t, res.(Result).Dmarc, "V=DMARC1; p=none; rua=mailto:postmaster@censys.io")
}

func TestDmarcLookup_Valid_3(t *testing.T) {
	resolver := InitTest(t)
	mockResults["_dmarc.zdns-testing.com"] = &core.SingleQueryResult{
		Answers: []interface{}{
			core.Answer{Name: "_dmarc.zdns-testing.com", Answer: "some TXT record"},
			// spaces and tabs should pass
			core.Answer{Name: "_dmarc.zdns-testing.com", Answer: "v\t\t\t=\t\t  DMARC1\t\t; p=none; rua=mailto:postmaster@censys.io"}},
	}
	dmarcMod := DmarcLookupModule{}
	dmarcMod.CLIInit(nil, &core.ResolverConfig{}, nil)
	res, _, status, _ := dmarcMod.Lookup(resolver, "_dmarc.zdns-testing.com", "")
	assert.Equal(t, queries[0].Class, uint16(dns.ClassINET))
	assert.Equal(t, queries[0].Type, dns.TypeTXT)
	assert.Equal(t, queries[0].Name, "_dmarc.zdns-testing.com")
	assert.Equal(t, queries[0].NameServer, "127.0.0.1")

	assert.Equal(t, core.STATUS_NOERROR, status)
	assert.Equal(t, res.(Result).Dmarc, "v\t\t\t=\t\t  DMARC1\t\t; p=none; rua=mailto:postmaster@censys.io")
}

func TestDmarcLookup_NotValid_1(t *testing.T) {
	resolver := InitTest(t)
	mockResults["_dmarc.zdns-testing.com"] = &core.SingleQueryResult{
		Answers: []interface{}{
			core.Answer{Name: "_dmarc.zdns-testing.com", Answer: "some TXT record"},
			// spaces before "v" should not be accepted
			core.Answer{Name: "_dmarc.zdns-testing.com", Answer: "\t\t   v   =DMARC1; p=none; rua=mailto:postmaster@censys.io"}},
	}
	dmarcMod := DmarcLookupModule{}
	dmarcMod.CLIInit(nil, &core.ResolverConfig{}, nil)
	res, _, status, _ := dmarcMod.Lookup(resolver, "_dmarc.zdns-testing.com", "")
	assert.Equal(t, queries[0].Class, uint16(dns.ClassINET))
	assert.Equal(t, queries[0].Type, dns.TypeTXT)
	assert.Equal(t, queries[0].Name, "_dmarc.zdns-testing.com")
	assert.Equal(t, queries[0].NameServer, "127.0.0.1")

	assert.Equal(t, core.STATUS_NO_RECORD, status)
	assert.Equal(t, res.(Result).Dmarc, "")
}

func TestDmarcLookup_NotValid_2(t *testing.T) {
	resolver := InitTest(t)
	mockResults["_dmarc.zdns-testing.com"] = &core.SingleQueryResult{
		Answers: []interface{}{
			core.Answer{Name: "_dmarc.zdns-testing.com", Answer: "some TXT record"},
			// DMARC1 should be capital letters
			core.Answer{Name: "_dmarc.zdns-testing.com", Answer: "v=DMARc1; p=none; rua=mailto:postmaster@censys.io"}},
	}
	dmarcMod := DmarcLookupModule{}
	dmarcMod.CLIInit(nil, &core.ResolverConfig{}, nil)
	res, _, status, _ := dmarcMod.Lookup(resolver, "_dmarc.zdns-testing.com", "")
	assert.Equal(t, queries[0].Class, uint16(dns.ClassINET))
	assert.Equal(t, queries[0].Type, dns.TypeTXT)
	assert.Equal(t, queries[0].Name, "_dmarc.zdns-testing.com")
	assert.Equal(t, queries[0].NameServer, "127.0.0.1")

	assert.Equal(t, core.STATUS_NO_RECORD, status)
	assert.Equal(t, res.(Result).Dmarc, "")
}

func TestDmarcLookup_NotValid_3(t *testing.T) {
	resolver := InitTest(t)
	mockResults["_dmarc.zdns-testing.com"] = &core.SingleQueryResult{
		Answers: []interface{}{
			core.Answer{Name: "_dmarc.zdns-testing.com", Answer: "some TXT record"},
			// ; has to be present after DMARC1
			core.Answer{Name: "_dmarc.zdns-testing.com", Answer: "v=DMARc1. p=none; rua=mailto:postmaster@censys.io"}},
	}
	dmarcMod := DmarcLookupModule{}
	dmarcMod.CLIInit(nil, &core.ResolverConfig{}, nil)
	res, _, status, _ := dmarcMod.Lookup(resolver, "_dmarc.zdns-testing.com", "")
	assert.Equal(t, queries[0].Class, uint16(dns.ClassINET))
	assert.Equal(t, queries[0].Type, dns.TypeTXT)
	assert.Equal(t, queries[0].Name, "_dmarc.zdns-testing.com")
	assert.Equal(t, queries[0].NameServer, "127.0.0.1")

	assert.Equal(t, core.STATUS_NO_RECORD, status)
	assert.Equal(t, res.(Result).Dmarc, "")
}
