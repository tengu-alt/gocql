//go:build all || unit
// +build all unit

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
/*
 * Content before git sha 34fdeebefcbf183ed7f916f931aa0586fdaa1b40
 * Copyright (c) 2016, The Gocql authors,
 * provided under the BSD-3-Clause License.
 * See the NOTICE file distributed with this work for additional information.
 */

package gocql

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"gopkg.in/inf.v0"

	"github.com/stretchr/testify/require"
)

type AliasInt int
type AliasUint uint
type AliasUint8 uint8
type AliasUint16 uint16
type AliasUint32 uint32
type AliasUint64 uint64

var marshalTests = []struct {
	Info           TypeInfo
	Data           []byte
	Value          interface{}
	MarshalError   error
	UnmarshalError error
}{
	{
		NativeType{proto: 2, typ: TypeVarchar},
		[]byte("hello world"),
		[]byte("hello world"),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeVarchar},
		[]byte("hello world"),
		"hello world",
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeVarchar},
		[]byte(nil),
		[]byte(nil),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeVarchar},
		[]byte("hello world"),
		MyString("hello world"),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeVarchar},
		[]byte("HELLO WORLD"),
		CustomString("hello world"),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeBlob},
		[]byte("hello\x00"),
		[]byte("hello\x00"),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeBlob},
		[]byte(nil),
		[]byte(nil),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeTimeUUID},
		[]byte{0x3d, 0xcd, 0x98, 0x0, 0xf3, 0xd9, 0x11, 0xbf, 0x86, 0xd4, 0xb8, 0xe8, 0x56, 0x2c, 0xc, 0xd0},
		func() UUID {
			x, _ := UUIDFromBytes([]byte{0x3d, 0xcd, 0x98, 0x0, 0xf3, 0xd9, 0x11, 0xbf, 0x86, 0xd4, 0xb8, 0xe8, 0x56, 0x2c, 0xc, 0xd0})
			return x
		}(),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeTimeUUID},
		[]byte{0xb8, 0xe8, 0x56, 0x2c, 0xc, 0xd0},
		[]byte{0xb8, 0xe8, 0x56, 0x2c, 0xc, 0xd0},
		MarshalError("can not marshal []byte 6 bytes long into timeuuid, must be exactly 16 bytes long"),
		UnmarshalError("unable to parse UUID: UUIDs must be exactly 16 bytes long"),
	},
	{
		NativeType{proto: 2, typ: TypeTimeUUID},
		[]byte{0x3d, 0xcd, 0x98, 0x0, 0xf3, 0xd9, 0x11, 0xbf, 0x86, 0xd4, 0xb8, 0xe8, 0x56, 0x2c, 0xc, 0xd0},
		[16]byte{0x3d, 0xcd, 0x98, 0x0, 0xf3, 0xd9, 0x11, 0xbf, 0x86, 0xd4, 0xb8, 0xe8, 0x56, 0x2c, 0xc, 0xd0},
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeInt},
		[]byte("\x00\x00\x00\x00"),
		0,
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeInt},
		[]byte("\x01\x02\x03\x04"),
		int(16909060),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeInt},
		[]byte("\x01\x02\x03\x04"),
		AliasInt(16909060),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeInt},
		[]byte("\x80\x00\x00\x00"),
		int32(math.MinInt32),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeInt},
		[]byte("\x7f\xff\xff\xff"),
		int32(math.MaxInt32),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeInt},
		[]byte("\x00\x00\x00\x00"),
		"0",
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeInt},
		[]byte("\x01\x02\x03\x04"),
		"16909060",
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeInt},
		[]byte("\x80\x00\x00\x00"),
		"-2147483648", // math.MinInt32
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeInt},
		[]byte("\x7f\xff\xff\xff"),
		"2147483647", // math.MaxInt32
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"),
		0,
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\x01\x02\x03\x04\x05\x06\x07\x08"),
		72623859790382856,
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\x80\x00\x00\x00\x00\x00\x00\x00"),
		int64(math.MinInt64),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\x7f\xff\xff\xff\xff\xff\xff\xff"),
		int64(math.MaxInt64),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\x00\x00\x00\x00\x00\x00\x00\x00"),
		"0",
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\x01\x02\x03\x04\x05\x06\x07\x08"),
		"72623859790382856",
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\x80\x00\x00\x00\x00\x00\x00\x00"),
		"-9223372036854775808", // math.MinInt64
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\x7f\xff\xff\xff\xff\xff\xff\xff"),
		"9223372036854775807", // math.MaxInt64
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeBoolean},
		[]byte("\x00"),
		false,
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeBoolean},
		[]byte("\x01"),
		true,
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeFloat},
		[]byte("\x40\x49\x0f\xdb"),
		float32(3.14159265),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeDouble},
		[]byte("\x40\x09\x21\xfb\x53\xc8\xd4\xf1"),
		float64(3.14159265),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeDecimal},
		[]byte("\x00\x00\x00\x00\x00"),
		inf.NewDec(0, 0),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeDecimal},
		[]byte("\x00\x00\x00\x00\x64"),
		inf.NewDec(100, 0),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeDecimal},
		[]byte("\x00\x00\x00\x02\x19"),
		decimalize("0.25"),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeDecimal},
		[]byte("\x00\x00\x00\x13\xD5\a;\x20\x14\xA2\x91"),
		decimalize("-0.0012095473475870063"), // From the iconara/cql-rb test suite
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeDecimal},
		[]byte("\x00\x00\x00\x13*\xF8\xC4\xDF\xEB]o"),
		decimalize("0.0012095473475870063"), // From the iconara/cql-rb test suite
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeDecimal},
		[]byte("\x00\x00\x00\x12\xF2\xD8\x02\xB6R\x7F\x99\xEE\x98#\x99\xA9V"),
		decimalize("-1042342234234.123423435647768234"), // From the iconara/cql-rb test suite
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeDecimal},
		[]byte("\x00\x00\x00\r\nJ\x04\"^\x91\x04\x8a\xb1\x18\xfe"),
		decimalize("1243878957943.1234124191998"), // From the datastax/python-driver test suite
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeDecimal},
		[]byte("\x00\x00\x00\x06\xe5\xde]\x98Y"),
		decimalize("-112233.441191"), // From the datastax/python-driver test suite
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeDecimal},
		[]byte("\x00\x00\x00\x14\x00\xfa\xce"),
		decimalize("0.00000000000000064206"), // From the datastax/python-driver test suite
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeDecimal},
		[]byte("\x00\x00\x00\x14\xff\x052"),
		decimalize("-0.00000000000000064206"), // From the datastax/python-driver test suite
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeDecimal},
		[]byte("\xff\xff\xff\x9c\x00\xfa\xce"),
		inf.NewDec(64206, -100), // From the datastax/python-driver test suite
		nil,
		nil,
	},
	{
		NativeType{proto: 4, typ: TypeTime},
		[]byte("\x00\x00\x01\x40\x77\x16\xe1\xb8"),
		time.Duration(int64(1376387523000)),
		nil,
		nil,
	},
	{
		NativeType{proto: 4, typ: TypeTime},
		[]byte("\x00\x00\x01\x40\x77\x16\xe1\xb8"),
		int64(1376387523000),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeTimestamp},
		[]byte("\x00\x00\x01\x40\x77\x16\xe1\xb8"),
		time.Date(2013, time.August, 13, 9, 52, 3, 0, time.UTC),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeTimestamp},
		[]byte("\x00\x00\x01\x40\x77\x16\xe1\xb8"),
		int64(1376387523000),
		nil,
		nil,
	},
	{
		NativeType{proto: 5, typ: TypeDuration},
		[]byte("\x89\xa2\xc3\xc2\x9a\xe0F\x91\x06"),
		Duration{Months: 1233, Days: 123213, Nanoseconds: 2312323},
		nil,
		nil,
	},
	{
		NativeType{proto: 5, typ: TypeDuration},
		[]byte("\x89\xa1\xc3\xc2\x99\xe0F\x91\x05"),
		Duration{Months: -1233, Days: -123213, Nanoseconds: -2312323},
		nil,
		nil,
	},
	{
		NativeType{proto: 5, typ: TypeDuration},
		[]byte("\x02\x04\x80\xe6"),
		Duration{Months: 1, Days: 2, Nanoseconds: 115},
		nil,
		nil,
	},
	{
		CollectionType{
			NativeType: NativeType{proto: 2, typ: TypeList},
			Elem:       NativeType{proto: 2, typ: TypeInt},
		},
		[]byte("\x00\x02\x00\x04\x00\x00\x00\x01\x00\x04\x00\x00\x00\x02"),
		[]int{1, 2},
		nil,
		nil,
	},
	{
		CollectionType{
			NativeType: NativeType{proto: 2, typ: TypeList},
			Elem:       NativeType{proto: 2, typ: TypeInt},
		},
		[]byte("\x00\x02\x00\x04\x00\x00\x00\x01\x00\x04\x00\x00\x00\x02"),
		[2]int{1, 2},
		nil,
		nil,
	},
	{
		CollectionType{
			NativeType: NativeType{proto: 2, typ: TypeSet},
			Elem:       NativeType{proto: 2, typ: TypeInt},
		},
		[]byte("\x00\x02\x00\x04\x00\x00\x00\x01\x00\x04\x00\x00\x00\x02"),
		[]int{1, 2},
		nil,
		nil,
	},
	{
		CollectionType{
			NativeType: NativeType{proto: 2, typ: TypeSet},
			Elem:       NativeType{proto: 2, typ: TypeInt},
		},
		[]byte{0, 0}, // encoding of a list should always include the size of the collection
		[]int{},
		nil,
		nil,
	},
	{
		CollectionType{
			NativeType: NativeType{proto: 2, typ: TypeMap},
			Key:        NativeType{proto: 2, typ: TypeVarchar},
			Elem:       NativeType{proto: 2, typ: TypeInt},
		},
		[]byte("\x00\x01\x00\x03foo\x00\x04\x00\x00\x00\x01"),
		map[string]int{"foo": 1},
		nil,
		nil,
	},
	{
		CollectionType{
			NativeType: NativeType{proto: 2, typ: TypeMap},
			Key:        NativeType{proto: 2, typ: TypeVarchar},
			Elem:       NativeType{proto: 2, typ: TypeInt},
		},
		[]byte{0, 0},
		map[string]int{},
		nil,
		nil,
	},
	{
		CollectionType{
			NativeType: NativeType{proto: 2, typ: TypeList},
			Elem:       NativeType{proto: 2, typ: TypeVarchar},
		},
		bytes.Join([][]byte{
			[]byte("\x00\x01\xFF\xFF"),
			bytes.Repeat([]byte("X"), math.MaxUint16)}, []byte("")),
		[]string{strings.Repeat("X", math.MaxUint16)},
		nil,
		nil,
	},
	{
		CollectionType{
			NativeType: NativeType{proto: 2, typ: TypeMap},
			Key:        NativeType{proto: 2, typ: TypeVarchar},
			Elem:       NativeType{proto: 2, typ: TypeVarchar},
		},
		bytes.Join([][]byte{
			[]byte("\x00\x01\xFF\xFF"),
			bytes.Repeat([]byte("X"), math.MaxUint16),
			[]byte("\xFF\xFF"),
			bytes.Repeat([]byte("Y"), math.MaxUint16)}, []byte("")),
		map[string]string{
			strings.Repeat("X", math.MaxUint16): strings.Repeat("Y", math.MaxUint16),
		},
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeVarint},
		[]byte("\x00"),
		0,
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeVarint},
		[]byte("\x37\xE2\x3C\xEC"),
		int32(937573612),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeVarint},
		[]byte("\x37\xE2\x3C\xEC"),
		big.NewInt(937573612),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeVarint},
		[]byte("\x03\x9EV \x15\f\x03\x9DK\x18\xCDI\\$?\a["),
		bigintize("1231312312331283012830129382342342412123"), // From the iconara/cql-rb test suite
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeVarint},
		[]byte("\xC9v\x8D:\x86"),
		big.NewInt(-234234234234), // From the iconara/cql-rb test suite
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeVarint},
		[]byte("f\x1e\xfd\xf2\xe3\xb1\x9f|\x04_\x15"),
		bigintize("123456789123456789123456789"), // From the datastax/python-driver test suite
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeVarint},
		[]byte(nil),
		nil,
		nil,
		UnmarshalError("can not unmarshal into non-pointer <nil>"),
	},
	{
		NativeType{proto: 2, typ: TypeInet},
		[]byte("\x7F\x00\x00\x01"),
		net.ParseIP("127.0.0.1").To4(),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeInet},
		[]byte("\xFF\xFF\xFF\xFF"),
		net.ParseIP("255.255.255.255").To4(),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeInet},
		[]byte("\x7F\x00\x00\x01"),
		"127.0.0.1",
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeInet},
		[]byte("\xFF\xFF\xFF\xFF"),
		"255.255.255.255",
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeInet},
		[]byte("\x21\xDA\x00\xd3\x00\x00\x2f\x3b\x02\xaa\x00\xff\xfe\x28\x9c\x5a"),
		"21da:d3:0:2f3b:2aa:ff:fe28:9c5a",
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeInet},
		[]byte("\xfe\x80\x00\x00\x00\x00\x00\x00\x02\x02\xb3\xff\xfe\x1e\x83\x29"),
		"fe80::202:b3ff:fe1e:8329",
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeInet},
		[]byte("\x21\xDA\x00\xd3\x00\x00\x2f\x3b\x02\xaa\x00\xff\xfe\x28\x9c\x5a"),
		net.ParseIP("21da:d3:0:2f3b:2aa:ff:fe28:9c5a"),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeInet},
		[]byte("\xfe\x80\x00\x00\x00\x00\x00\x00\x02\x02\xb3\xff\xfe\x1e\x83\x29"),
		net.ParseIP("fe80::202:b3ff:fe1e:8329"),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeInt},
		[]byte(nil),
		nil,
		nil,
		UnmarshalError("can not unmarshal into non-pointer <nil>"),
	},
	{
		NativeType{proto: 2, typ: TypeVarchar},
		[]byte("nullable string"),
		func() *string {
			value := "nullable string"
			return &value
		}(),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeVarchar},
		[]byte(nil),
		(*string)(nil),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeInt},
		[]byte("\x7f\xff\xff\xff"),
		func() *int {
			var value int = math.MaxInt32
			return &value
		}(),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeInt},
		[]byte(nil),
		(*int)(nil),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeTimeUUID},
		[]byte{0x3d, 0xcd, 0x98, 0x0, 0xf3, 0xd9, 0x11, 0xbf, 0x86, 0xd4, 0xb8, 0xe8, 0x56, 0x2c, 0xc, 0xd0},
		&UUID{0x3d, 0xcd, 0x98, 0x0, 0xf3, 0xd9, 0x11, 0xbf, 0x86, 0xd4, 0xb8, 0xe8, 0x56, 0x2c, 0xc, 0xd0},
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeTimeUUID},
		[]byte(nil),
		(*UUID)(nil),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeTimestamp},
		[]byte("\x00\x00\x01\x40\x77\x16\xe1\xb8"),
		func() *time.Time {
			t := time.Date(2013, time.August, 13, 9, 52, 3, 0, time.UTC)
			return &t
		}(),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeTimestamp},
		[]byte(nil),
		(*time.Time)(nil),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeBoolean},
		[]byte("\x00"),
		func() *bool {
			b := false
			return &b
		}(),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeBoolean},
		[]byte("\x01"),
		func() *bool {
			b := true
			return &b
		}(),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeBoolean},
		[]byte(nil),
		(*bool)(nil),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeFloat},
		[]byte("\x40\x49\x0f\xdb"),
		func() *float32 {
			f := float32(3.14159265)
			return &f
		}(),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeFloat},
		[]byte(nil),
		(*float32)(nil),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeDouble},
		[]byte("\x40\x09\x21\xfb\x53\xc8\xd4\xf1"),
		func() *float64 {
			d := float64(3.14159265)
			return &d
		}(),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeDouble},
		[]byte(nil),
		(*float64)(nil),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeInet},
		[]byte("\x7F\x00\x00\x01"),
		func() *net.IP {
			ip := net.ParseIP("127.0.0.1").To4()
			return &ip
		}(),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeInet},
		[]byte(nil),
		(*net.IP)(nil),
		nil,
		nil,
	},
	{
		CollectionType{
			NativeType: NativeType{proto: 2, typ: TypeList},
			Elem:       NativeType{proto: 2, typ: TypeInt},
		},
		[]byte("\x00\x02\x00\x04\x00\x00\x00\x01\x00\x04\x00\x00\x00\x02"),
		func() *[]int {
			l := []int{1, 2}
			return &l
		}(),
		nil,
		nil,
	},
	{
		CollectionType{
			NativeType: NativeType{proto: 3, typ: TypeList},
			Elem:       NativeType{proto: 3, typ: TypeInt},
		},
		[]byte("\x00\x00\x00\x02\x00\x00\x00\x04\x00\x00\x00\x01\x00\x00\x00\x04\x00\x00\x00\x02"),
		func() *[]int {
			l := []int{1, 2}
			return &l
		}(),
		nil,
		nil,
	},
	{
		CollectionType{
			NativeType: NativeType{proto: 2, typ: TypeList},
			Elem:       NativeType{proto: 2, typ: TypeInt},
		},
		[]byte(nil),
		(*[]int)(nil),
		nil,
		nil,
	},
	{
		CollectionType{
			NativeType: NativeType{proto: 2, typ: TypeMap},
			Key:        NativeType{proto: 2, typ: TypeVarchar},
			Elem:       NativeType{proto: 2, typ: TypeInt},
		},
		[]byte("\x00\x01\x00\x03foo\x00\x04\x00\x00\x00\x01"),
		func() *map[string]int {
			m := map[string]int{"foo": 1}
			return &m
		}(),
		nil,
		nil,
	},
	{
		CollectionType{
			NativeType: NativeType{proto: 2, typ: TypeMap},
			Key:        NativeType{proto: 2, typ: TypeVarchar},
			Elem:       NativeType{proto: 2, typ: TypeInt},
		},
		[]byte(nil),
		(*map[string]int)(nil),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeVarchar},
		[]byte("HELLO WORLD"),
		func() *CustomString {
			customString := CustomString("hello world")
			return &customString
		}(),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeVarchar},
		[]byte(nil),
		(*CustomString)(nil),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeSmallInt},
		[]byte("\x7f\xff"),
		32767, // math.MaxInt16
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeSmallInt},
		[]byte("\x7f\xff"),
		"32767", // math.MaxInt16
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeSmallInt},
		[]byte("\x00\x01"),
		int16(1),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeSmallInt},
		[]byte("\xff\xff"),
		int16(-1),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeSmallInt},
		[]byte("\x00\xff"),
		uint8(255),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeSmallInt},
		[]byte("\xff\xff"),
		uint16(65535),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeSmallInt},
		[]byte("\xff\xff"),
		uint32(65535),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeSmallInt},
		[]byte("\xff\xff"),
		uint64(65535),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeSmallInt},
		[]byte("\x00\xff"),
		AliasUint8(255),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeSmallInt},
		[]byte("\xff\xff"),
		AliasUint16(65535),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeSmallInt},
		[]byte("\xff\xff"),
		AliasUint32(65535),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeSmallInt},
		[]byte("\xff\xff"),
		AliasUint64(65535),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeSmallInt},
		[]byte("\xff\xff"),
		AliasUint(65535),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeTinyInt},
		[]byte("\x7f"),
		127, // math.MaxInt8
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeTinyInt},
		[]byte("\x7f"),
		"127", // math.MaxInt8
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeTinyInt},
		[]byte("\x01"),
		int16(1),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeTinyInt},
		[]byte("\xff"),
		int16(-1),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeTinyInt},
		[]byte("\xff"),
		uint8(255),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeTinyInt},
		[]byte("\xff"),
		uint64(255),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeTinyInt},
		[]byte("\xff"),
		uint32(255),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeTinyInt},
		[]byte("\xff"),
		uint16(255),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeTinyInt},
		[]byte("\xff"),
		uint(255),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeTinyInt},
		[]byte("\xff"),
		AliasUint8(255),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeTinyInt},
		[]byte("\xff"),
		AliasUint64(255),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeTinyInt},
		[]byte("\xff"),
		AliasUint32(255),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeTinyInt},
		[]byte("\xff"),
		AliasUint16(255),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeTinyInt},
		[]byte("\xff"),
		AliasUint(255),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\x00\x00\x00\x00\x00\x00\x00\xff"),
		uint8(math.MaxUint8),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\x00\x00\x00\x00\x00\x00\xff\xff"),
		uint64(math.MaxUint16),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\x00\x00\x00\x00\xff\xff\xff\xff"),
		uint64(math.MaxUint32),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\xff\xff\xff\xff\xff\xff\xff\xff"),
		uint64(math.MaxUint64),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeInt},
		[]byte("\xff\xff\xff\xff"),
		uint32(math.MaxUint32),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeInt},
		[]byte("\xff\xff\xff\xff"),
		uint64(math.MaxUint32),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeBlob},
		[]byte(nil),
		([]byte)(nil),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeVarchar},
		[]byte{},
		func() interface{} {
			var s string
			return &s
		}(),
		nil,
		nil,
	},
	{
		NativeType{proto: 2, typ: TypeTime},
		encBigInt(1000),
		time.Duration(1000),
		nil,
		nil,
	},
}

var unmarshalTests = []struct {
	Info           TypeInfo
	Data           []byte
	Value          interface{}
	UnmarshalError error
}{
	{
		NativeType{proto: 2, typ: TypeSmallInt},
		[]byte("\xff\xff"),
		uint8(0),
		UnmarshalError("unmarshal int: value -1 out of range for uint8"),
	},
	{
		NativeType{proto: 2, typ: TypeSmallInt},
		[]byte("\x01\x00"),
		uint8(0),
		UnmarshalError("unmarshal int: value 256 out of range for uint8"),
	},
	{
		NativeType{proto: 2, typ: TypeInt},
		[]byte("\xff\xff\xff\xff"),
		uint8(0),
		UnmarshalError("unmarshal int: value -1 out of range for uint8"),
	},
	{
		NativeType{proto: 2, typ: TypeInt},
		[]byte("\x00\x00\x01\x00"),
		uint8(0),
		UnmarshalError("unmarshal int: value 256 out of range for uint8"),
	},
	{
		NativeType{proto: 2, typ: TypeInt},
		[]byte("\xff\xff\xff\xff"),
		uint16(0),
		UnmarshalError("unmarshal int: value -1 out of range for uint16"),
	},
	{
		NativeType{proto: 2, typ: TypeInt},
		[]byte("\x00\x01\x00\x00"),
		uint16(0),
		UnmarshalError("unmarshal int: value 65536 out of range for uint16"),
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\xff\xff\xff\xff\xff\xff\xff\xff"),
		uint8(0),
		UnmarshalError("unmarshal int: value -1 out of range for uint8"),
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\x00\x00\x00\x00\x00\x00\x01\x00"),
		uint8(0),
		UnmarshalError("unmarshal int: value 256 out of range for uint8"),
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\xff\xff\xff\xff\xff\xff\xff\xff"),
		uint8(0),
		UnmarshalError("unmarshal int: value -1 out of range for uint8"),
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\x00\x00\x00\x00\x00\x00\x01\x00"),
		uint8(0),
		UnmarshalError("unmarshal int: value 256 out of range for uint8"),
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\xff\xff\xff\xff\xff\xff\xff\xff"),
		uint16(0),
		UnmarshalError("unmarshal int: value -1 out of range for uint16"),
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\x00\x00\x00\x00\x00\x01\x00\x00"),
		uint16(0),
		UnmarshalError("unmarshal int: value 65536 out of range for uint16"),
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\xff\xff\xff\xff\xff\xff\xff\xff"),
		uint32(0),
		UnmarshalError("unmarshal int: value -1 out of range for uint32"),
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\x00\x00\x00\x01\x00\x00\x00\x00"),
		uint32(0),
		UnmarshalError("unmarshal int: value 4294967296 out of range for uint32"),
	},
	{
		NativeType{proto: 2, typ: TypeSmallInt},
		[]byte("\xff\xff"),
		AliasUint8(0),
		UnmarshalError("unmarshal int: value -1 out of range for gocql.AliasUint8"),
	},
	{
		NativeType{proto: 2, typ: TypeSmallInt},
		[]byte("\x01\x00"),
		AliasUint8(0),
		UnmarshalError("unmarshal int: value 256 out of range for gocql.AliasUint8"),
	},
	{
		NativeType{proto: 2, typ: TypeInt},
		[]byte("\xff\xff\xff\xff"),
		AliasUint8(0),
		UnmarshalError("unmarshal int: value -1 out of range for gocql.AliasUint8"),
	},
	{
		NativeType{proto: 2, typ: TypeInt},
		[]byte("\x00\x00\x01\x00"),
		AliasUint8(0),
		UnmarshalError("unmarshal int: value 256 out of range for gocql.AliasUint8"),
	},
	{
		NativeType{proto: 2, typ: TypeInt},
		[]byte("\xff\xff\xff\xff"),
		AliasUint16(0),
		UnmarshalError("unmarshal int: value -1 out of range for gocql.AliasUint16"),
	},
	{
		NativeType{proto: 2, typ: TypeInt},
		[]byte("\x00\x01\x00\x00"),
		AliasUint16(0),
		UnmarshalError("unmarshal int: value 65536 out of range for gocql.AliasUint16"),
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\xff\xff\xff\xff\xff\xff\xff\xff"),
		AliasUint8(0),
		UnmarshalError("unmarshal int: value -1 out of range for gocql.AliasUint8"),
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\x00\x00\x00\x00\x00\x00\x01\x00"),
		AliasUint8(0),
		UnmarshalError("unmarshal int: value 256 out of range for gocql.AliasUint8"),
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\xff\xff\xff\xff\xff\xff\xff\xff"),
		AliasUint8(0),
		UnmarshalError("unmarshal int: value -1 out of range for gocql.AliasUint8"),
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\x00\x00\x00\x00\x00\x00\x01\x00"),
		AliasUint8(0),
		UnmarshalError("unmarshal int: value 256 out of range for gocql.AliasUint8"),
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\xff\xff\xff\xff\xff\xff\xff\xff"),
		AliasUint16(0),
		UnmarshalError("unmarshal int: value -1 out of range for gocql.AliasUint16"),
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\x00\x00\x00\x00\x00\x01\x00\x00"),
		AliasUint16(0),
		UnmarshalError("unmarshal int: value 65536 out of range for gocql.AliasUint16"),
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\xff\xff\xff\xff\xff\xff\xff\xff"),
		AliasUint32(0),
		UnmarshalError("unmarshal int: value -1 out of range for gocql.AliasUint32"),
	},
	{
		NativeType{proto: 2, typ: TypeBigInt},
		[]byte("\x00\x00\x00\x01\x00\x00\x00\x00"),
		AliasUint32(0),
		UnmarshalError("unmarshal int: value 4294967296 out of range for gocql.AliasUint32"),
	},
	{
		CollectionType{
			NativeType: NativeType{proto: 3, typ: TypeList},
			Elem:       NativeType{proto: 3, typ: TypeInt},
		},
		[]byte("\x00\x00\x00\x02\x00\x00\x00\x04\x00\x00"), // truncated data
		func() *[]int {
			l := []int{1, 2}
			return &l
		}(),
		UnmarshalError("unmarshal list: unexpected eof"),
	},
	{
		CollectionType{
			NativeType: NativeType{proto: 2, typ: TypeMap},
			Key:        NativeType{proto: 2, typ: TypeVarchar},
			Elem:       NativeType{proto: 2, typ: TypeInt},
		},
		[]byte("\x00\x01\x00\x03fo"),
		map[string]int{"foo": 1},
		UnmarshalError("unmarshal map: unexpected eof"),
	},
	{
		CollectionType{
			NativeType: NativeType{proto: 2, typ: TypeMap},
			Key:        NativeType{proto: 2, typ: TypeVarchar},
			Elem:       NativeType{proto: 2, typ: TypeInt},
		},
		[]byte("\x00\x01\x00\x03foo\x00\x04\x00\x00"),
		map[string]int{"foo": 1},
		UnmarshalError("unmarshal map: unexpected eof"),
	},
	{
		NativeType{proto: 2, typ: TypeDecimal},
		[]byte("\xff\xff\xff"),
		inf.NewDec(0, 0), // From the datastax/python-driver test suite
		UnmarshalError("inf.Dec needs at least 4 bytes, while value has only 3"),
	},
	{
		NativeType{proto: 5, typ: TypeDuration},
		[]byte("\x89\xa2\xc3\xc2\x9a\xe0F\x91"),
		Duration{},
		UnmarshalError("failed to unmarshal duration into *gocql.Duration: failed to extract nanoseconds: data expect to have 9 bytes, but it has only 8"),
	},
	{
		NativeType{proto: 5, typ: TypeDuration},
		[]byte("\x89\xa2\xc3\xc2\x9a"),
		Duration{},
		UnmarshalError("failed to unmarshal duration into *gocql.Duration: failed to extract nanoseconds: unexpected eof"),
	},
	{
		NativeType{proto: 5, typ: TypeDuration},
		[]byte("\x89\xa2\xc3\xc2"),
		Duration{},
		UnmarshalError("failed to unmarshal duration into *gocql.Duration: failed to extract days: data expect to have 5 bytes, but it has only 4"),
	},
	{
		NativeType{proto: 5, typ: TypeDuration},
		[]byte("\x89\xa2"),
		Duration{},
		UnmarshalError("failed to unmarshal duration into *gocql.Duration: failed to extract days: unexpected eof"),
	},
	{
		NativeType{proto: 5, typ: TypeDuration},
		[]byte("\x89"),
		Duration{},
		UnmarshalError("failed to unmarshal duration into *gocql.Duration: failed to extract month: data expect to have 2 bytes, but it has only 1"),
	},
}

func decimalize(s string) *inf.Dec {
	i, _ := new(inf.Dec).SetString(s)
	return i
}

func bigintize(s string) *big.Int {
	i, _ := new(big.Int).SetString(s, 10)
	return i
}

func TestMarshal_Encode(t *testing.T) {
	for i, test := range marshalTests {
		if test.MarshalError == nil {
			data, err := Marshal(test.Info, test.Value)
			if err != nil {
				t.Errorf("marshalTest[%d]: %v", i, err)
				continue
			}
			if !bytes.Equal(data, test.Data) {
				t.Errorf("marshalTest[%d]: expected %q, got %q (%#v)", i, test.Data, data, test.Value)
			}
		} else {
			if _, err := Marshal(test.Info, test.Value); err != test.MarshalError {
				t.Errorf("marshalTest[%d] (%v=>%t): %#v returned error %#v, want %#v.", i, test.Info, test.Value, test.Value, err, test.MarshalError)
			}
		}
	}
}

func TestMarshal_Decode(t *testing.T) {
	for i, test := range marshalTests {
		if test.UnmarshalError == nil {
			v := reflect.New(reflect.TypeOf(test.Value))
			err := Unmarshal(test.Info, test.Data, v.Interface())
			if err != nil {
				t.Errorf("unmarshalTest[%d] (%v=>%T): %v", i, test.Info, test.Value, err)
				continue
			}
			if !reflect.DeepEqual(v.Elem().Interface(), test.Value) {
				t.Errorf("unmarshalTest[%d] (%v=>%T): expected %#v, got %#v.", i, test.Info, test.Value, test.Value, v.Elem().Interface())
			}
		} else {
			if err := Unmarshal(test.Info, test.Data, test.Value); err != test.UnmarshalError {
				t.Errorf("unmarshalTest[%d] (%v=>%T): %#v returned error %#v, want %#v.", i, test.Info, test.Value, test.Value, err, test.UnmarshalError)
			}
		}
	}
	for i, test := range unmarshalTests {
		v := reflect.New(reflect.TypeOf(test.Value))
		if test.UnmarshalError == nil {
			err := Unmarshal(test.Info, test.Data, v.Interface())
			if err != nil {
				t.Errorf("unmarshalTest[%d] (%v=>%T): %v", i, test.Info, test.Value, err)
				continue
			}
			if !reflect.DeepEqual(v.Elem().Interface(), test.Value) {
				t.Errorf("unmarshalTest[%d] (%v=>%T): expected %#v, got %#v.", i, test.Info, test.Value, test.Value, v.Elem().Interface())
			}
		} else {
			if err := Unmarshal(test.Info, test.Data, v.Interface()); err != test.UnmarshalError {
				t.Errorf("unmarshalTest[%d] (%v=>%T): %#v returned error %#v, want %#v.", i, test.Info, test.Value, test.Value, err, test.UnmarshalError)
			}
		}
	}
}

func TestMarshalVarint(t *testing.T) {
	varintTests := []struct {
		Value       interface{}
		Marshaled   []byte
		Unmarshaled *big.Int
	}{
		{
			Value:       int8(0),
			Marshaled:   []byte("\x00"),
			Unmarshaled: big.NewInt(0),
		},
		{
			Value:       uint8(255),
			Marshaled:   []byte("\x00\xFF"),
			Unmarshaled: big.NewInt(255),
		},
		{
			Value:       int8(-1),
			Marshaled:   []byte("\xFF"),
			Unmarshaled: big.NewInt(-1),
		},
		{
			Value:       big.NewInt(math.MaxInt32),
			Marshaled:   []byte("\x7F\xFF\xFF\xFF"),
			Unmarshaled: big.NewInt(math.MaxInt32),
		},
		{
			Value:       big.NewInt(int64(math.MaxInt32) + 1),
			Marshaled:   []byte("\x00\x80\x00\x00\x00"),
			Unmarshaled: big.NewInt(int64(math.MaxInt32) + 1),
		},
		{
			Value:       big.NewInt(math.MinInt32),
			Marshaled:   []byte("\x80\x00\x00\x00"),
			Unmarshaled: big.NewInt(math.MinInt32),
		},
		{
			Value:       big.NewInt(int64(math.MinInt32) - 1),
			Marshaled:   []byte("\xFF\x7F\xFF\xFF\xFF"),
			Unmarshaled: big.NewInt(int64(math.MinInt32) - 1),
		},
		{
			Value:       math.MinInt64,
			Marshaled:   []byte("\x80\x00\x00\x00\x00\x00\x00\x00"),
			Unmarshaled: big.NewInt(math.MinInt64),
		},
		{
			Value:       uint64(math.MaxInt64) + 1,
			Marshaled:   []byte("\x00\x80\x00\x00\x00\x00\x00\x00\x00"),
			Unmarshaled: bigintize("9223372036854775808"),
		},
		{
			Value:       bigintize("2361183241434822606848"), // 2**71
			Marshaled:   []byte("\x00\x80\x00\x00\x00\x00\x00\x00\x00\x00"),
			Unmarshaled: bigintize("2361183241434822606848"),
		},
		{
			Value:       bigintize("-9223372036854775809"), // -2**63 - 1
			Marshaled:   []byte("\xFF\x7F\xFF\xFF\xFF\xFF\xFF\xFF\xFF"),
			Unmarshaled: bigintize("-9223372036854775809"),
		},
	}

	for i, test := range varintTests {
		data, err := Marshal(NativeType{proto: 2, typ: TypeVarint}, test.Value)
		if err != nil {
			t.Errorf("error marshaling varint: %v (test #%d)", err, i)
		}

		if !bytes.Equal(test.Marshaled, data) {
			t.Errorf("marshaled varint mismatch: expected %v, got %v (test #%d)", test.Marshaled, data, i)
		}

		binder := new(big.Int)
		err = Unmarshal(NativeType{proto: 2, typ: TypeVarint}, test.Marshaled, binder)
		if err != nil {
			t.Errorf("error unmarshaling varint: %v (test #%d)", err, i)
		}

		if test.Unmarshaled.Cmp(binder) != 0 {
			t.Errorf("unmarshaled varint mismatch: expected %v, got %v (test #%d)", test.Unmarshaled, binder, i)
		}
	}

	varintUint64Tests := []struct {
		Value       interface{}
		Marshaled   []byte
		Unmarshaled uint64
	}{
		{
			Value:       int8(0),
			Marshaled:   []byte("\x00"),
			Unmarshaled: 0,
		},
		{
			Value:       uint8(255),
			Marshaled:   []byte("\x00\xFF"),
			Unmarshaled: 255,
		},
		{
			Value:       big.NewInt(math.MaxInt32),
			Marshaled:   []byte("\x7F\xFF\xFF\xFF"),
			Unmarshaled: uint64(math.MaxInt32),
		},
		{
			Value:       big.NewInt(int64(math.MaxInt32) + 1),
			Marshaled:   []byte("\x00\x80\x00\x00\x00"),
			Unmarshaled: uint64(int64(math.MaxInt32) + 1),
		},
		{
			Value:       uint64(math.MaxInt64) + 1,
			Marshaled:   []byte("\x00\x80\x00\x00\x00\x00\x00\x00\x00"),
			Unmarshaled: 9223372036854775808,
		},
		{
			Value:       uint64(math.MaxUint64),
			Marshaled:   []byte("\x00\xFF\xFF\xFF\xFF\xFF\xFF\xFF\xFF"),
			Unmarshaled: uint64(math.MaxUint64),
		},
	}

	for i, test := range varintUint64Tests {
		data, err := Marshal(NativeType{proto: 2, typ: TypeVarint}, test.Value)
		if err != nil {
			t.Errorf("error marshaling varint: %v (test #%d)", err, i)
		}

		if !bytes.Equal(test.Marshaled, data) {
			t.Errorf("marshaled varint mismatch: expected %v, got %v (test #%d)", test.Marshaled, data, i)
		}

		var binder uint64
		err = Unmarshal(NativeType{proto: 2, typ: TypeVarint}, test.Marshaled, &binder)
		if err != nil {
			t.Errorf("error unmarshaling varint to uint64: %v (test #%d)", err, i)
		}

		if test.Unmarshaled != binder {
			t.Errorf("unmarshaled varint mismatch: expected %v, got %v (test #%d)", test.Unmarshaled, binder, i)
		}
	}
}

func TestMarshalBigInt(t *testing.T) {
	var testStruct = []struct {
		Info         TypeInfo
		Value        interface{}
		MarshalError error
	}{
		{
			NativeType{proto: 2, typ: TypeBigInt},
			"-78635384813432117863538481343211",
			MarshalError("can not marshal string to bigint: strconv.ParseInt: parsing \"-78635384813432117863538481343211\": value out of range"),
		},
		{
			NativeType{proto: 2, typ: TypeBigInt},
			"922337203685477692259749625974294",
			MarshalError("can not marshal string to bigint: strconv.ParseInt: parsing \"922337203685477692259749625974294\": value out of range"),
		},
	}

	t.Run("testMarshalBigInt", func(t *testing.T) {
		for _, tc := range testStruct {
			_, err := Marshal(tc.Info, tc.Value)
			require.Equal(t, tc.MarshalError, err)
		}
	})
}

func equalStringPointerSlice(leftList, rightList []*string) bool {
	if len(leftList) != len(rightList) {
		return false
	}
	for index := range leftList {
		if !reflect.DeepEqual(rightList[index], leftList[index]) {
			return false
		}
	}
	return true
}

func TestMarshalList(t *testing.T) {
	typeInfoV2 := CollectionType{
		NativeType: NativeType{proto: 2, typ: TypeList},
		Elem:       NativeType{proto: 2, typ: TypeVarchar},
	}
	typeInfoV3 := CollectionType{
		NativeType: NativeType{proto: 3, typ: TypeList},
		Elem:       NativeType{proto: 3, typ: TypeVarchar},
	}

	type tc struct {
		typeInfo CollectionType
		input    []*string
		expected []*string
	}

	valueA := "valueA"
	valueB := "valueB"
	valueEmpty := ""
	testCases := []tc{
		{
			typeInfo: typeInfoV2,
			input:    []*string{&valueA},
			expected: []*string{&valueA},
		},
		{
			typeInfo: typeInfoV2,
			input:    []*string{&valueA, &valueB},
			expected: []*string{&valueA, &valueB},
		},
		{
			typeInfo: typeInfoV2,
			input:    []*string{&valueA, &valueEmpty, &valueB},
			expected: []*string{&valueA, &valueEmpty, &valueB},
		},
		{
			typeInfo: typeInfoV2,
			input:    []*string{&valueEmpty},
			expected: []*string{&valueEmpty},
		},
		{
			// nil values are marshalled to empty values for protocol < 3
			typeInfo: typeInfoV2,
			input:    []*string{nil},
			expected: []*string{&valueEmpty},
		},
		{
			typeInfo: typeInfoV2,
			input:    []*string{&valueA, nil, &valueB},
			expected: []*string{&valueA, &valueEmpty, &valueB},
		},
		{
			typeInfo: typeInfoV3,
			input:    []*string{&valueEmpty},
			expected: []*string{&valueEmpty},
		},
		{
			typeInfo: typeInfoV3,
			input:    []*string{nil},
			expected: []*string{nil},
		},
		{
			typeInfo: typeInfoV3,
			input:    []*string{&valueA, nil, &valueB},
			expected: []*string{&valueA, nil, &valueB},
		},
	}

	listDatas := [][]byte{}
	for _, c := range testCases {
		listData, marshalErr := Marshal(c.typeInfo, c.input)
		if nil != marshalErr {
			t.Errorf("Error marshal %+v of type %+v: %s", c.input, c.typeInfo, marshalErr)
		}
		listDatas = append(listDatas, listData)
	}

	outputLists := [][]*string{}

	var outputList []*string

	for i, listData := range listDatas {
		if unmarshalErr := Unmarshal(testCases[i].typeInfo, listData, &outputList); nil != unmarshalErr {
			t.Error(unmarshalErr)
		}
		resultList := []interface{}{}
		for i := range outputList {
			if outputList[i] != nil {
				resultList = append(resultList, *outputList[i])
			} else {
				resultList = append(resultList, nil)
			}
		}
		outputLists = append(outputLists, outputList)
	}

	for index, c := range testCases {
		outputList := outputLists[index]
		if !equalStringPointerSlice(c.expected, outputList) {
			t.Errorf("Lists %+v not equal to lists %+v, but should", c.expected, outputList)
		}
	}
}

type CustomString string

func (c CustomString) MarshalCQL(info TypeInfo) ([]byte, error) {
	return []byte(strings.ToUpper(string(c))), nil
}
func (c *CustomString) UnmarshalCQL(info TypeInfo, data []byte) error {
	*c = CustomString(strings.ToLower(string(data)))
	return nil
}

type MyString string

var typeLookupTest = []struct {
	TypeName     string
	ExpectedType Type
}{
	{"AsciiType", TypeAscii},
	{"LongType", TypeBigInt},
	{"BytesType", TypeBlob},
	{"BooleanType", TypeBoolean},
	{"CounterColumnType", TypeCounter},
	{"DecimalType", TypeDecimal},
	{"DoubleType", TypeDouble},
	{"FloatType", TypeFloat},
	{"Int32Type", TypeInt},
	{"DateType", TypeTimestamp},
	{"TimestampType", TypeTimestamp},
	{"UUIDType", TypeUUID},
	{"UTF8Type", TypeVarchar},
	{"IntegerType", TypeVarint},
	{"TimeUUIDType", TypeTimeUUID},
	{"InetAddressType", TypeInet},
	{"MapType", TypeMap},
	{"ListType", TypeList},
	{"SetType", TypeSet},
	{"unknown", TypeCustom},
	{"ShortType", TypeSmallInt},
	{"ByteType", TypeTinyInt},
}

func testType(t *testing.T, cassType string, expectedType Type) {
	if computedType := getApacheCassandraType(apacheCassandraTypePrefix + cassType); computedType != expectedType {
		t.Errorf("Cassandra custom type lookup for %s failed. Expected %s, got %s.", cassType, expectedType.String(), computedType.String())
	}
}

func TestLookupCassType(t *testing.T) {
	for _, lookupTest := range typeLookupTest {
		testType(t, lookupTest.TypeName, lookupTest.ExpectedType)
	}
}

type MyPointerMarshaler struct{}

func (m *MyPointerMarshaler) MarshalCQL(_ TypeInfo) ([]byte, error) {
	return []byte{42}, nil
}

func TestMarshalPointer(t *testing.T) {
	m := &MyPointerMarshaler{}
	typ := NativeType{proto: 2, typ: TypeInt}

	data, err := Marshal(typ, m)

	if err != nil {
		t.Errorf("Pointer marshaling failed. Error: %s", err)
	}
	if len(data) != 1 || data[0] != 42 {
		t.Errorf("Pointer marshaling failed. Expected %+v, got %+v", []byte{42}, data)
	}
}

func TestMarshalTime(t *testing.T) {
	durationS := "1h10m10s"
	duration, _ := time.ParseDuration(durationS)
	expectedData := encBigInt(duration.Nanoseconds())
	var marshalTimeTests = []struct {
		Info  TypeInfo
		Data  []byte
		Value interface{}
	}{
		{
			NativeType{proto: 4, typ: TypeTime},
			expectedData,
			duration.Nanoseconds(),
		},
		{
			NativeType{proto: 4, typ: TypeTime},
			expectedData,
			duration,
		},
		{
			NativeType{proto: 4, typ: TypeTime},
			expectedData,
			&duration,
		},
	}

	for i, test := range marshalTimeTests {
		t.Log(i, test)
		data, err := Marshal(test.Info, test.Value)
		if err != nil {
			t.Errorf("marshalTest[%d]: %v", i, err)
			continue
		}
		if !bytes.Equal(data, test.Data) {
			t.Errorf("marshalTest[%d]: expected %x (%v), got %x (%v) for time %s", i,
				test.Data, decInt(test.Data), data, decInt(data), test.Value)
		}
	}
}

func TestMarshalTimestamp(t *testing.T) {
	var marshalTimestampTests = []struct {
		Info  TypeInfo
		Data  []byte
		Value interface{}
	}{
		{
			NativeType{proto: 2, typ: TypeTimestamp},
			[]byte("\x00\x00\x01\x40\x77\x16\xe1\xb8"),
			time.Date(2013, time.August, 13, 9, 52, 3, 0, time.UTC),
		},
		{
			NativeType{proto: 2, typ: TypeTimestamp},
			[]byte("\x00\x00\x01\x40\x77\x16\xe1\xb8"),
			int64(1376387523000),
		},
		{
			// 9223372036854 is the maximum time representable in ms since the epoch
			// with int64 if using UnixNano to convert
			NativeType{proto: 2, typ: TypeTimestamp},
			[]byte("\x00\x00\x08\x63\x7b\xd0\x5a\xf6"),
			time.Date(2262, time.April, 11, 23, 47, 16, 854775807, time.UTC),
		},
		{
			// One nanosecond after causes overflow when using UnixNano
			// Instead it should resolve to the same time in ms
			NativeType{proto: 2, typ: TypeTimestamp},
			[]byte("\x00\x00\x08\x63\x7b\xd0\x5a\xf6"),
			time.Date(2262, time.April, 11, 23, 47, 16, 854775808, time.UTC),
		},
		{
			// -9223372036855 is the minimum time representable in ms since the epoch
			// with int64 if using UnixNano to convert
			NativeType{proto: 2, typ: TypeTimestamp},
			[]byte("\xff\xff\xf7\x9c\x84\x2f\xa5\x09"),
			time.Date(1677, time.September, 21, 00, 12, 43, 145224192, time.UTC),
		},
		{
			// One nanosecond earlier causes overflow when using UnixNano
			// it should resolve to the same time in ms
			NativeType{proto: 2, typ: TypeTimestamp},
			[]byte("\xff\xff\xf7\x9c\x84\x2f\xa5\x09"),
			time.Date(1677, time.September, 21, 00, 12, 43, 145224191, time.UTC),
		},
		{
			// Store the zero time as a blank slice
			NativeType{proto: 2, typ: TypeTimestamp},
			[]byte{},
			time.Time{},
		},
	}

	for i, test := range marshalTimestampTests {
		t.Log(i, test)
		data, err := Marshal(test.Info, test.Value)
		if err != nil {
			t.Errorf("marshalTest[%d]: %v", i, err)
			continue
		}
		if !bytes.Equal(data, test.Data) {
			t.Errorf("marshalTest[%d]: expected %x (%v), got %x (%v) for time %s", i,
				test.Data, decBigInt(test.Data), data, decBigInt(data), test.Value)
		}
	}
}

func TestMarshalTuple(t *testing.T) {
	info := TupleTypeInfo{
		NativeType: NativeType{proto: 3, typ: TypeTuple},
		Elems: []TypeInfo{
			NativeType{proto: 3, typ: TypeVarchar},
			NativeType{proto: 3, typ: TypeVarchar},
		},
	}

	stringToPtr := func(s string) *string { return &s }
	checkString := func(t *testing.T, exp string, got string) {
		if got != exp {
			t.Errorf("expected string to be %v, got %v", exp, got)
		}
	}

	type tupleStruct struct {
		A string
		B *string
	}
	var (
		s1 *string
		s2 *string
	)

	testCases := []struct {
		name       string
		expected   []byte
		value      interface{}
		checkValue interface{}
		check      func(*testing.T, interface{})
	}{
		{
			name:       "interface-slice:two-strings",
			expected:   []byte("\x00\x00\x00\x03foo\x00\x00\x00\x03bar"),
			value:      []interface{}{"foo", "bar"},
			checkValue: []interface{}{&s1, &s2},
			check: func(t *testing.T, v interface{}) {
				checkString(t, "foo", *s1)
				checkString(t, "bar", *s2)
			},
		},
		{
			name:       "interface-slice:one-string-one-nil-string",
			expected:   []byte("\x00\x00\x00\x03foo\xff\xff\xff\xff"),
			value:      []interface{}{"foo", nil},
			checkValue: []interface{}{&s1, &s2},
			check: func(t *testing.T, v interface{}) {
				checkString(t, "foo", *s1)
				if s2 != nil {
					t.Errorf("expected string to be nil, got %v", *s2)
				}
			},
		},
		{
			name:     "struct:two-strings",
			expected: []byte("\x00\x00\x00\x03foo\x00\x00\x00\x03bar"),
			value: tupleStruct{
				A: "foo",
				B: stringToPtr("bar"),
			},
			checkValue: &tupleStruct{},
			check: func(t *testing.T, v interface{}) {
				got := v.(*tupleStruct)
				if got.A != "foo" {
					t.Errorf("expected A string to be %v, got %v", "foo", got.A)
				}
				if got.B == nil {
					t.Errorf("expected B string to be %v, got nil", "bar")
				}
				if *got.B != "bar" {
					t.Errorf("expected B string to be %v, got %v", "bar", got.B)
				}
			},
		},
		{
			name:       "struct:one-string-one-nil-string",
			expected:   []byte("\x00\x00\x00\x03foo\xff\xff\xff\xff"),
			value:      tupleStruct{A: "foo", B: nil},
			checkValue: &tupleStruct{},
			check: func(t *testing.T, v interface{}) {
				got := v.(*tupleStruct)
				if got.A != "foo" {
					t.Errorf("expected A string to be %v, got %v", "foo", got.A)
				}
				if got.B != nil {
					t.Errorf("expected B string to be nil, got %v", *got.B)
				}
			},
		},
		{
			name:     "arrayslice:two-strings",
			expected: []byte("\x00\x00\x00\x03foo\x00\x00\x00\x03bar"),
			value: [2]*string{
				stringToPtr("foo"),
				stringToPtr("bar"),
			},
			checkValue: &[2]*string{},
			check: func(t *testing.T, v interface{}) {
				got := v.(*[2]*string)
				checkString(t, "foo", *(got[0]))
				checkString(t, "bar", *(got[1]))
			},
		},
		{
			name:     "arrayslice:one-string-one-nil-string",
			expected: []byte("\x00\x00\x00\x03foo\xff\xff\xff\xff"),
			value: [2]*string{
				stringToPtr("foo"),
				nil,
			},
			checkValue: &[2]*string{},
			check: func(t *testing.T, v interface{}) {
				got := v.(*[2]*string)
				checkString(t, "foo", *(got[0]))
				if got[1] != nil {
					t.Errorf("expected string to be nil, got %v", *got[1])
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := Marshal(info, tc.value)
			if err != nil {
				t.Errorf("marshalTest: %v", err)
				return
			}

			if !bytes.Equal(data, tc.expected) {
				t.Errorf("marshalTest: expected %x (%v), got %x (%v)",
					tc.expected, decBigInt(tc.expected), data, decBigInt(data))
				return
			}

			err = Unmarshal(info, data, tc.checkValue)
			if err != nil {
				t.Errorf("unmarshalTest: %v", err)
				return
			}

			tc.check(t, tc.checkValue)
		})
	}
}

func TestUnmarshalTuple(t *testing.T) {
	info := TupleTypeInfo{
		NativeType: NativeType{proto: 3, typ: TypeTuple},
		Elems: []TypeInfo{
			NativeType{proto: 3, typ: TypeVarchar},
			NativeType{proto: 3, typ: TypeVarchar},
		},
	}

	// As per the CQL spec, a tuple is a sequence of "bytes" values.
	// Here we encode a null value (length -1) and the "foo" string (length 3)

	data := []byte("\xff\xff\xff\xff\x00\x00\x00\x03foo")

	t.Run("struct-ptr", func(t *testing.T) {
		var tmp struct {
			A *string
			B *string
		}

		err := Unmarshal(info, data, &tmp)
		if err != nil {
			t.Errorf("unmarshalTest: %v", err)
			return
		}

		if tmp.A != nil || *tmp.B != "foo" {
			t.Errorf("unmarshalTest: expected [nil, foo], got [%v, %v]", *tmp.A, *tmp.B)
		}
	})
	t.Run("struct-nonptr", func(t *testing.T) {
		var tmp struct {
			A string
			B string
		}

		err := Unmarshal(info, data, &tmp)
		if err != nil {
			t.Errorf("unmarshalTest: %v", err)
			return
		}

		if tmp.A != "" || tmp.B != "foo" {
			t.Errorf("unmarshalTest: expected [nil, foo], got [%v, %v]", tmp.A, tmp.B)
		}
	})

	t.Run("array", func(t *testing.T) {
		var tmp [2]*string

		err := Unmarshal(info, data, &tmp)
		if err != nil {
			t.Errorf("unmarshalTest: %v", err)
			return
		}

		if tmp[0] != nil || *tmp[1] != "foo" {
			t.Errorf("unmarshalTest: expected [nil, foo], got [%v, %v]", *tmp[0], *tmp[1])
		}
	})
	t.Run("array-nonptr", func(t *testing.T) {
		var tmp [2]string

		err := Unmarshal(info, data, &tmp)
		if err != nil {
			t.Errorf("unmarshalTest: %v", err)
			return
		}

		if tmp[0] != "" || tmp[1] != "foo" {
			t.Errorf("unmarshalTest: expected [nil, foo], got [%v, %v]", tmp[0], tmp[1])
		}
	})
}

func TestMarshalUDTMap(t *testing.T) {
	typeInfo := UDTTypeInfo{NativeType{proto: 3, typ: TypeUDT}, "", "xyz", []UDTField{
		{Name: "x", Type: NativeType{proto: 3, typ: TypeInt}},
		{Name: "y", Type: NativeType{proto: 3, typ: TypeInt}},
		{Name: "z", Type: NativeType{proto: 3, typ: TypeInt}},
	}}

	t.Run("partially bound", func(t *testing.T) {
		value := map[string]interface{}{
			"y": 2,
			"z": 3,
		}
		expected := []byte("\xff\xff\xff\xff\x00\x00\x00\x04\x00\x00\x00\x02\x00\x00\x00\x04\x00\x00\x00\x03")

		data, err := Marshal(typeInfo, value)
		if err != nil {
			t.Errorf("got error %#v", err)
		}
		if !bytes.Equal(data, expected) {
			t.Errorf("got value %x", data)
		}
	})
	t.Run("partially bound from the beginning", func(t *testing.T) {
		value := map[string]interface{}{
			"x": 1,
			"y": 2,
		}
		expected := []byte("\x00\x00\x00\x04\x00\x00\x00\x01\x00\x00\x00\x04\x00\x00\x00\x02\xff\xff\xff\xff")

		data, err := Marshal(typeInfo, value)
		if err != nil {
			t.Errorf("got error %#v", err)
		}
		if !bytes.Equal(data, expected) {
			t.Errorf("got value %x", data)
		}
	})
	t.Run("fully bound", func(t *testing.T) {
		value := map[string]interface{}{
			"x": 1,
			"y": 2,
			"z": 3,
		}
		expected := []byte("\x00\x00\x00\x04\x00\x00\x00\x01\x00\x00\x00\x04\x00\x00\x00\x02\x00\x00\x00\x04\x00\x00\x00\x03")

		data, err := Marshal(typeInfo, value)
		if err != nil {
			t.Errorf("got error %#v", err)
		}
		if !bytes.Equal(data, expected) {
			t.Errorf("got value %x", data)
		}
	})
}

func TestMarshalUDTStruct(t *testing.T) {
	typeInfo := UDTTypeInfo{NativeType{proto: 3, typ: TypeUDT}, "", "xyz", []UDTField{
		{Name: "x", Type: NativeType{proto: 3, typ: TypeInt}},
		{Name: "y", Type: NativeType{proto: 3, typ: TypeInt}},
		{Name: "z", Type: NativeType{proto: 3, typ: TypeInt}},
	}}

	type xyzStruct struct {
		X int32 `cql:"x"`
		Y int32 `cql:"y"`
		Z int32 `cql:"z"`
	}
	type xyStruct struct {
		X int32 `cql:"x"`
		Y int32 `cql:"y"`
	}
	type yzStruct struct {
		Y int32 `cql:"y"`
		Z int32 `cql:"z"`
	}

	t.Run("partially bound", func(t *testing.T) {
		value := yzStruct{
			Y: 2,
			Z: 3,
		}
		expected := []byte("\xff\xff\xff\xff\x00\x00\x00\x04\x00\x00\x00\x02\x00\x00\x00\x04\x00\x00\x00\x03")

		data, err := Marshal(typeInfo, value)
		if err != nil {
			t.Errorf("got error %#v", err)
		}
		if !bytes.Equal(data, expected) {
			t.Errorf("got value %x", data)
		}
	})
	t.Run("partially bound from the beginning", func(t *testing.T) {
		value := xyStruct{
			X: 1,
			Y: 2,
		}
		expected := []byte("\x00\x00\x00\x04\x00\x00\x00\x01\x00\x00\x00\x04\x00\x00\x00\x02\xff\xff\xff\xff")

		data, err := Marshal(typeInfo, value)
		if err != nil {
			t.Errorf("got error %#v", err)
		}
		if !bytes.Equal(data, expected) {
			t.Errorf("got value %x", data)
		}
	})
	t.Run("fully bound", func(t *testing.T) {
		value := xyzStruct{
			X: 1,
			Y: 2,
			Z: 3,
		}
		expected := []byte("\x00\x00\x00\x04\x00\x00\x00\x01\x00\x00\x00\x04\x00\x00\x00\x02\x00\x00\x00\x04\x00\x00\x00\x03")

		data, err := Marshal(typeInfo, value)
		if err != nil {
			t.Errorf("got error %#v", err)
		}
		if !bytes.Equal(data, expected) {
			t.Errorf("got value %x", data)
		}
	})
}

func TestMarshalNil(t *testing.T) {
	types := []Type{
		TypeAscii,
		TypeBlob,
		TypeBoolean,
		TypeBigInt,
		TypeCounter,
		TypeDecimal,
		TypeDouble,
		TypeFloat,
		TypeInt,
		TypeTimestamp,
		TypeUUID,
		TypeVarchar,
		TypeVarint,
		TypeTimeUUID,
		TypeInet,
	}

	for _, typ := range types {
		data, err := Marshal(NativeType{proto: 3, typ: typ}, nil)
		if err != nil {
			t.Errorf("unable to marshal nil %v: %v\n", typ, err)
		} else if data != nil {
			t.Errorf("expected to get nil byte for nil %v got % X", typ, data)
		}
	}
}

func TestUnmarshalInetCopyBytes(t *testing.T) {
	data := []byte{127, 0, 0, 1}
	var ip net.IP
	if err := unmarshalInet(NativeType{proto: 2, typ: TypeInet}, data, &ip); err != nil {
		t.Fatal(err)
	}

	copy(data, []byte{0xFF, 0xFF, 0xFF, 0xFF})
	ip2 := net.IP(data)
	if !ip.Equal(net.IPv4(127, 0, 0, 1)) {
		t.Fatalf("IP memory shared with data: ip=%v ip2=%v", ip, ip2)
	}
}

func TestUnmarshalDate(t *testing.T) {
	data := []uint8{0x80, 0x0, 0x43, 0x31}
	var date time.Time
	if err := unmarshalDate(NativeType{proto: 2, typ: TypeDate}, data, &date); err != nil {
		t.Fatal(err)
	}

	expectedDate := "2017-02-04"
	formattedDate := date.Format("2006-01-02")
	if expectedDate != formattedDate {
		t.Errorf("marshalTest: expected %v, got %v", expectedDate, formattedDate)
		return
	}
	var stringDate string
	if err2 := unmarshalDate(NativeType{proto: 2, typ: TypeDate}, data, &stringDate); err2 != nil {
		t.Fatal(err2)
	}
	if expectedDate != stringDate {
		t.Errorf("marshalTest: expected %v, got %v", expectedDate, formattedDate)
		return
	}
}

func TestMarshalDate(t *testing.T) {
	now := time.Now().UTC()
	timestamp := now.UnixNano() / int64(time.Millisecond)
	expectedData := encInt(int32(timestamp/86400000 + int64(1<<31)))

	var marshalDateTests = []struct {
		Info  TypeInfo
		Data  []byte
		Value interface{}
	}{
		{
			NativeType{proto: 4, typ: TypeDate},
			expectedData,
			timestamp,
		},
		{
			NativeType{proto: 4, typ: TypeDate},
			expectedData,
			now,
		},
		{
			NativeType{proto: 4, typ: TypeDate},
			expectedData,
			&now,
		},
		{
			NativeType{proto: 4, typ: TypeDate},
			expectedData,
			now.Format("2006-01-02"),
		},
	}

	for i, test := range marshalDateTests {
		t.Log(i, test)
		data, err := Marshal(test.Info, test.Value)
		if err != nil {
			t.Errorf("marshalTest[%d]: %v", i, err)
			continue
		}
		if !bytes.Equal(data, test.Data) {
			t.Errorf("marshalTest[%d]: expected %x (%v), got %x (%v) for time %s", i,
				test.Data, decInt(test.Data), data, decInt(data), test.Value)
		}
	}
}

func TestLargeDate(t *testing.T) {
	farFuture := time.Date(999999, time.December, 31, 0, 0, 0, 0, time.UTC)
	expectedFutureData := encInt(int32(farFuture.UnixMilli()/86400000 + int64(1<<31)))

	farPast := time.Date(-999999, time.January, 1, 0, 0, 0, 0, time.UTC)
	expectedPastData := encInt(int32(farPast.UnixMilli()/86400000 + int64(1<<31)))

	var marshalDateTests = []struct {
		Data         []byte
		Value        interface{}
		ExpectedDate string
	}{
		{
			expectedFutureData,
			farFuture,
			"999999-12-31",
		},
		{
			expectedPastData,
			farPast,
			"-999999-01-01",
		},
	}

	nativeType := NativeType{proto: 4, typ: TypeDate}

	for i, test := range marshalDateTests {
		t.Log(i, test)

		data, err := Marshal(nativeType, test.Value)
		if err != nil {
			t.Errorf("largeDateTest[%d]: %v", i, err)
			continue
		}
		if !bytes.Equal(data, test.Data) {
			t.Errorf("largeDateTest[%d]: expected %x (%v), got %x (%v) for time %s", i,
				test.Data, decInt(test.Data), data, decInt(data), test.Value)
		}

		var date time.Time
		if err := Unmarshal(nativeType, data, &date); err != nil {
			t.Fatal(err)
		}

		formattedDate := date.Format("2006-01-02")
		if test.ExpectedDate != formattedDate {
			t.Fatalf("largeDateTest: expected %v, got %v", test.ExpectedDate, formattedDate)
		}
	}
}

func BenchmarkUnmarshalVarchar(b *testing.B) {
	b.ReportAllocs()
	src := make([]byte, 1024)
	dst := make([]byte, len(src))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := unmarshalVarchar(NativeType{}, src, &dst); err != nil {
			b.Fatal(err)
		}
	}
}

func TestMarshalDuration(t *testing.T) {
	durationS := "1h10m10s"
	duration, _ := time.ParseDuration(durationS)
	expectedData := append([]byte{0, 0}, encVint(duration.Nanoseconds())...)
	var marshalDurationTests = []struct {
		Info  TypeInfo
		Data  []byte
		Value interface{}
	}{
		{
			NativeType{proto: 5, typ: TypeDuration},
			expectedData,
			duration.Nanoseconds(),
		},
		{
			NativeType{proto: 5, typ: TypeDuration},
			expectedData,
			duration,
		},
		{
			NativeType{proto: 5, typ: TypeDuration},
			expectedData,
			durationS,
		},
		{
			NativeType{proto: 5, typ: TypeDuration},
			expectedData,
			&duration,
		},
	}

	for i, test := range marshalDurationTests {
		t.Log(i, test)
		data, err := Marshal(test.Info, test.Value)
		if err != nil {
			t.Errorf("marshalTest[%d]: %v", i, err)
			continue
		}
		if !bytes.Equal(data, test.Data) {
			t.Errorf("marshalTest[%d]: expected %x (%v), got %x (%v) for time %s", i,
				test.Data, decInt(test.Data), data, decInt(data), test.Value)
		}
	}
}

func TestReadCollectionSize(t *testing.T) {
	listV2 := CollectionType{
		NativeType: NativeType{proto: 2, typ: TypeList},
		Elem:       NativeType{proto: 2, typ: TypeVarchar},
	}
	listV3 := CollectionType{
		NativeType: NativeType{proto: 3, typ: TypeList},
		Elem:       NativeType{proto: 3, typ: TypeVarchar},
	}

	tests := []struct {
		name         string
		info         CollectionType
		data         []byte
		isError      bool
		expectedSize int
	}{
		{
			name:    "short read 0 proto 2",
			info:    listV2,
			data:    []byte{},
			isError: true,
		},
		{
			name:    "short read 1 proto 2",
			info:    listV2,
			data:    []byte{0x01},
			isError: true,
		},
		{
			name:         "good read proto 2",
			info:         listV2,
			data:         []byte{0x01, 0x38},
			expectedSize: 0x0138,
		},
		{
			name:    "short read 0 proto 3",
			info:    listV3,
			data:    []byte{},
			isError: true,
		},
		{
			name:    "short read 1 proto 3",
			info:    listV3,
			data:    []byte{0x01},
			isError: true,
		},
		{
			name:    "short read 2 proto 3",
			info:    listV3,
			data:    []byte{0x01, 0x38},
			isError: true,
		},
		{
			name:    "short read 3 proto 3",
			info:    listV3,
			data:    []byte{0x01, 0x38, 0x42},
			isError: true,
		},
		{
			name:         "good read proto 3",
			info:         listV3,
			data:         []byte{0x01, 0x38, 0x42, 0x22},
			expectedSize: 0x01384222,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			size, _, err := readCollectionSize(test.info, test.data)
			if test.isError {
				if err == nil {
					t.Fatal("Expected error, but it was nil")
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, got %v", err)
				}
				if size != test.expectedSize {
					t.Fatalf("Expected size of %d, but got %d", test.expectedSize, size)
				}
			}
		})
	}
}

func TestReadUnsignedVInt(t *testing.T) {
	tests := []struct {
		decodedInt  uint64
		encodedVint []byte
	}{
		{
			decodedInt:  0,
			encodedVint: []byte{0},
		},
		{
			decodedInt:  100,
			encodedVint: []byte{100},
		},
		{
			decodedInt:  256000,
			encodedVint: []byte{195, 232, 0},
		},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%d", test.decodedInt), func(t *testing.T) {
			actual, _, err := readUnsignedVInt(test.encodedVint)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
			if actual != test.decodedInt {
				t.Fatalf("Expected %d, but got %d", test.decodedInt, actual)
			}
		})
	}
}

func BenchmarkUnmarshalUUID(b *testing.B) {
	b.ReportAllocs()
	src := make([]byte, 16)
	dst := UUID{}
	var ti TypeInfo = NativeType{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := unmarshalUUID(ti, src, &dst); err != nil {
			b.Fatal(err)
		}
	}
}

func TestUnmarshalUDT(t *testing.T) {
	info := UDTTypeInfo{
		NativeType: NativeType{proto: 4, typ: TypeUDT},
		Name:       "myudt",
		KeySpace:   "myks",
		Elements: []UDTField{
			{
				Name: "first",
				Type: NativeType{proto: 4, typ: TypeAscii},
			},
			{
				Name: "second",
				Type: NativeType{proto: 4, typ: TypeSmallInt},
			},
		},
	}
	data := bytesWithLength( // UDT
		bytesWithLength([]byte("Hello")),    // first
		bytesWithLength([]byte("\x00\x2a")), // second
	)
	value := map[string]interface{}{}
	expectedErr := UnmarshalError("can not unmarshal into non-pointer map[string]interface {}")

	if err := Unmarshal(info, data, value); err != expectedErr {
		t.Errorf("(%v=>%T): %#v returned error %#v, want %#v.",
			info, value, value, err, expectedErr)
	}
}

// bytesWithLength concatenates all data slices and prepends the total length as uint32.
// The length does not count the size of the uint32 used for writing the size.
func bytesWithLength(data ...[]byte) []byte {
	totalLen := 0
	for i := range data {
		totalLen += len(data[i])
	}
	if totalLen > math.MaxUint32 {
		panic("total length overflows")
	}
	ret := make([]byte, totalLen+4)
	binary.BigEndian.PutUint32(ret[:4], uint32(totalLen))
	buf := ret[4:]
	for i := range data {
		n := copy(buf, data[i])
		buf = buf[n:]
	}
	return ret
}
