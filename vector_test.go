//go:build all || cassandra
// +build all cassandra

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
	"fmt"
	"github.com/stretchr/testify/require"
	"gopkg.in/inf.v0"
	"net"
	"reflect"
	"testing"
	"time"
)

type person struct {
	FirstName string `cql:"first_name"`
	LastName  string `cql:"last_name"`
	Age       int    `cql:"age"`
}

func (p person) String() string {
	return fmt.Sprintf("Person{firstName: %s, lastName: %s, Age: %d}", p.FirstName, p.LastName, p.Age)
}

func TestVector_Marshaler(t *testing.T) {
	session := createSession(t)
	defer session.Close()

	if flagCassVersion.Before(5, 0, 0) {
		t.Skip("Vector types have been introduced in Cassandra 5.0")
	}

	err := createTable(session, `CREATE TABLE IF NOT EXISTS gocql_test.vector_fixed(id int primary key, vec vector<float, 3>);`)
	if err != nil {
		t.Fatal(err)
	}

	err = createTable(session, `CREATE TABLE IF NOT EXISTS gocql_test.vector_variable(id int primary key, vec vector<text, 4>);`)
	if err != nil {
		t.Fatal(err)
	}

	insertFixVec := []float32{8, 2.5, -5.0}
	err = session.Query("INSERT INTO vector_fixed(id, vec) VALUES(?, ?)", 1, insertFixVec).Exec()
	if err != nil {
		t.Fatal(err)
	}
	var selectFixVec []float32
	err = session.Query("SELECT vec FROM vector_fixed WHERE id = ?", 1).Scan(&selectFixVec)
	if err != nil {
		t.Fatal(err)
	}
	assertDeepEqual(t, "fixed size element vector", insertFixVec, selectFixVec)

	longText := randomText(500)
	insertVarVec := []string{"apache", "cassandra", longText, "gocql"}
	err = session.Query("INSERT INTO vector_variable(id, vec) VALUES(?, ?)", 1, insertVarVec).Exec()
	if err != nil {
		t.Fatal(err)
	}
	var selectVarVec []string
	err = session.Query("SELECT vec FROM vector_variable WHERE id = ?", 1).Scan(&selectVarVec)
	if err != nil {
		t.Fatal(err)
	}
	assertDeepEqual(t, "variable size element vector", insertVarVec, selectVarVec)
}

func TestVector_Types(t *testing.T) {
	session := createSession(t)
	defer session.Close()

	if flagCassVersion.Before(5, 0, 0) {
		t.Skip("Vector types have been introduced in Cassandra 5.0")
	}

	timestamp1, _ := time.Parse("2006-01-02", "2000-01-01")
	timestamp2, _ := time.Parse("2006-01-02 15:04:05", "2024-01-01 10:31:45")
	timestamp3, _ := time.Parse("2006-01-02 15:04:05.000", "2024-05-01 10:31:45.987")

	date1, _ := time.Parse("2006-01-02", "2000-01-01")
	date2, _ := time.Parse("2006-01-02", "2022-03-14")
	date3, _ := time.Parse("2006-01-02", "2024-12-31")

	time1, _ := time.Parse("15:04:05", "01:00:00")
	time2, _ := time.Parse("15:04:05", "15:23:59")
	time3, _ := time.Parse("15:04:05.000", "10:31:45.987")

	duration1 := Duration{0, 1, 1920000000000}
	duration2 := Duration{1, 1, 1920000000000}
	duration3 := Duration{31, 0, 60000000000}

	map1 := make(map[string]int)
	map1["a"] = 1
	map1["b"] = 2
	map1["c"] = 3
	map2 := make(map[string]int)
	map2["abc"] = 123
	map3 := make(map[string]int)

	tests := []struct {
		name       string
		cqlType    string
		value      interface{}
		comparator func(interface{}, interface{})
	}{
		{name: "ascii", cqlType: TypeAscii.String(), value: []string{"a", "1", "Z"}},
		{name: "bigint", cqlType: TypeBigInt.String(), value: []int64{1, 2, 3}},
		{name: "blob", cqlType: TypeBlob.String(), value: [][]byte{[]byte{1, 2, 3}, []byte{4, 5, 6, 7}, []byte{8, 9}}},
		{name: "boolean", cqlType: TypeBoolean.String(), value: []bool{true, false, true}},
		{name: "counter", cqlType: TypeCounter.String(), value: []int64{5, 6, 7}},
		{name: "decimal", cqlType: TypeDecimal.String(), value: []inf.Dec{*inf.NewDec(1, 0), *inf.NewDec(2, 1), *inf.NewDec(-3, 2)}},
		{name: "double", cqlType: TypeDouble.String(), value: []float64{0.1, -1.2, 3}},
		{name: "float", cqlType: TypeFloat.String(), value: []float32{0.1, -1.2, 3}},
		{name: "int", cqlType: TypeInt.String(), value: []int32{1, 2, 3}},
		{name: "text", cqlType: TypeText.String(), value: []string{"a", "b", "c"}},
		{name: "timestamp", cqlType: TypeTimestamp.String(), value: []time.Time{timestamp1, timestamp2, timestamp3}},
		{name: "uuid", cqlType: TypeUUID.String(), value: []UUID{MustRandomUUID(), MustRandomUUID(), MustRandomUUID()}},
		{name: "varchar", cqlType: TypeVarchar.String(), value: []string{"abc", "def", "ghi"}},
		{name: "varint", cqlType: TypeVarint.String(), value: []uint64{uint64(1234), uint64(123498765), uint64(18446744073709551615)}},
		{name: "timeuuid", cqlType: TypeTimeUUID.String(), value: []UUID{TimeUUID(), TimeUUID(), TimeUUID()}},
		{
			name:    "inet",
			cqlType: TypeInet.String(),
			value:   []net.IP{net.IPv4(127, 0, 0, 1), net.IPv4(192, 168, 1, 1), net.IPv4(8, 8, 8, 8)},
			comparator: func(e interface{}, a interface{}) {
				expected := e.([]net.IP)
				actual := a.([]net.IP)
				assertEqual(t, "vector size", len(expected), len(actual))
				for i, _ := range expected {
					assertTrue(t, "vector", expected[i].Equal(actual[i]))
				}
			},
		},
		{name: "date", cqlType: TypeDate.String(), value: []time.Time{date1, date2, date3}},
		{name: "time", cqlType: TypeTimestamp.String(), value: []time.Time{time1, time2, time3}},
		{name: "smallint", cqlType: TypeSmallInt.String(), value: []int16{127, 256, -1234}},
		{name: "tinyint", cqlType: TypeTinyInt.String(), value: []int8{127, 9, -123}},
		{name: "duration", cqlType: TypeDuration.String(), value: []Duration{duration1, duration2, duration3}},
		{name: "vector_vector_float", cqlType: "vector<float, 5>", value: [][]float32{{0.1, -1.2, 3, 5, 5}, {10.1, -122222.0002, 35.0, 1, 1}, {0, 0, 0, 0, 0}}},
		{name: "vector_vector_set_float", cqlType: "vector<set<float>, 5>", value: [][][]float32{
			{{1, 2}, {2, -1}, {3}, {0}, {-1.3}},
			{{2, 3}, {2, -1}, {3}, {0}, {-1.3}},
			{{1, 1000.0}, {0}, {}, {12, 14, 15, 16}, {-1.3}},
		}},
		{name: "vector_tuple_text_int_float", cqlType: "tuple<text, int, float>", value: [][]interface{}{{"a", 1, float32(0.5)}, {"b", 2, float32(-1.2)}, {"c", 3, float32(0)}}},
		{name: "vector_tuple_text_list_text", cqlType: "tuple<text, list<text>>", value: [][]interface{}{{"a", []string{"b", "c"}}, {"d", []string{"e", "f", "g"}}, {"h", []string{"i"}}}},
		{name: "vector_set_text", cqlType: "set<text>", value: [][]string{{"a", "b"}, {"c", "d"}, {"e", "f"}}},
		{name: "vector_list_int", cqlType: "list<int>", value: [][]int32{{1, 2, 3}, {-1, -2, -3}, {0, 0, 0}}},
		{name: "vector_map_text_int", cqlType: "map<text, int>", value: []map[string]int{map1, map2, map3}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tableName := fmt.Sprintf("vector_%s", test.name)
			err := createTable(session, fmt.Sprintf(`CREATE TABLE IF NOT EXISTS gocql_test.%s(id int primary key, vec vector<%s, 3>);`, tableName, test.cqlType))
			if err != nil {
				t.Fatal(err)
			}

			err = session.Query(fmt.Sprintf("INSERT INTO %s(id, vec) VALUES(?, ?)", tableName), 1, test.value).Exec()
			if err != nil {
				t.Fatal(err)
			}

			v := reflect.New(reflect.TypeOf(test.value))
			err = session.Query(fmt.Sprintf("SELECT vec FROM %s WHERE id = ?", tableName), 1).Scan(v.Interface())
			if err != nil {
				t.Fatal(err)
			}
			if test.comparator != nil {
				test.comparator(test.value, v.Elem().Interface())
			} else {
				assertDeepEqual(t, "vector", test.value, v.Elem().Interface())
			}
		})
	}
}

func TestVector_MarshalerUDT(t *testing.T) {
	session := createSession(t)
	defer session.Close()

	if flagCassVersion.Before(5, 0, 0) {
		t.Skip("Vector types have been introduced in Cassandra 5.0")
	}

	err := createTable(session, `CREATE TYPE gocql_test.person(
		first_name text,
		last_name text,
		age int);`)
	if err != nil {
		t.Fatal(err)
	}

	err = createTable(session, `CREATE TABLE gocql_test.vector_relatives(
		id int,
		couple vector<person, 2>,
		primary key(id)
	);`)
	if err != nil {
		t.Fatal(err)
	}

	p1 := person{"Johny", "Bravo", 25}
	p2 := person{"Capitan", "Planet", 5}
	insVec := []person{p1, p2}

	err = session.Query("INSERT INTO vector_relatives(id, couple) VALUES(?, ?)", 1, insVec).Exec()
	if err != nil {
		t.Fatal(err)
	}

	var selVec []person

	err = session.Query("SELECT couple FROM vector_relatives WHERE id = ?", 1).Scan(&selVec)
	if err != nil {
		t.Fatal(err)
	}

	assertDeepEqual(t, "udt", &insVec, &selVec)
}

func TestVector_Empty(t *testing.T) {
	session := createSession(t)
	defer session.Close()

	if flagCassVersion.Before(5, 0, 0) {
		t.Skip("Vector types have been introduced in Cassandra 5.0")
	}

	err := createTable(session, `CREATE TABLE IF NOT EXISTS gocql_test.vector_fixed_null(id int primary key, vec vector<float, 3>);`)
	if err != nil {
		t.Fatal(err)
	}

	err = createTable(session, `CREATE TABLE IF NOT EXISTS gocql_test.vector_variable_null(id int primary key, vec vector<text, 4>);`)
	if err != nil {
		t.Fatal(err)
	}

	err = session.Query("INSERT INTO vector_fixed_null(id) VALUES(?)", 1).Exec()
	if err != nil {
		t.Fatal(err)
	}
	var selectFixVec []float32
	err = session.Query("SELECT vec FROM vector_fixed_null WHERE id = ?", 1).Scan(&selectFixVec)
	if err != nil {
		t.Fatal(err)
	}
	assertTrue(t, "fixed size element vector is empty", selectFixVec == nil)

	err = session.Query("INSERT INTO vector_variable_null(id) VALUES(?)", 1).Exec()
	if err != nil {
		t.Fatal(err)
	}
	var selectVarVec []string
	err = session.Query("SELECT vec FROM vector_variable_null WHERE id = ?", 1).Scan(&selectVarVec)
	if err != nil {
		t.Fatal(err)
	}
	assertTrue(t, "variable size element vector is empty", selectVarVec == nil)
}

func TestVector_MissingDimension(t *testing.T) {
	session := createSession(t)
	defer session.Close()

	if flagCassVersion.Before(5, 0, 0) {
		t.Skip("Vector types have been introduced in Cassandra 5.0")
	}

	err := createTable(session, `CREATE TABLE IF NOT EXISTS gocql_test.vector_fixed(id int primary key, vec vector<float, 3>);`)
	if err != nil {
		t.Fatal(err)
	}

	err = session.Query("INSERT INTO vector_fixed(id, vec) VALUES(?, ?)", 1, []float32{8, -5.0}).Exec()
	require.Error(t, err, "expected vector with 3 dimensions, received 2")

	err = session.Query("INSERT INTO vector_fixed(id, vec) VALUES(?, ?)", 1, []float32{8, -5.0, 1, 3}).Exec()
	require.Error(t, err, "expected vector with 3 dimensions, received 4")
}

func TestVector_SubTypeParsing(t *testing.T) {
	tests := []struct {
		name     string
		custom   string
		expected TypeInfo
	}{
		{name: "text", custom: "org.apache.cassandra.db.marshal.UTF8Type", expected: NativeType{typ: TypeVarchar}},
		{name: "set_int", custom: "org.apache.cassandra.db.marshal.SetType(org.apache.cassandra.db.marshal.Int32Type)", expected: CollectionType{NativeType{typ: TypeSet}, nil, NativeType{typ: TypeInt}}},
		{
			name:   "udt",
			custom: "org.apache.cassandra.db.marshal.UserType(gocql_test,706572736f6e,66697273745f6e616d65:org.apache.cassandra.db.marshal.UTF8Type,6c6173745f6e616d65:org.apache.cassandra.db.marshal.UTF8Type,616765:org.apache.cassandra.db.marshal.Int32Type)",
			expected: UDTTypeInfo{
				NativeType{typ: TypeUDT},
				"gocql_test",
				"person",
				[]UDTField{
					UDTField{"first_name", NativeType{typ: TypeVarchar}},
					UDTField{"last_name", NativeType{typ: TypeVarchar}},
					UDTField{"age", NativeType{typ: TypeInt}},
				},
			},
		},
		{
			name:   "tuple",
			custom: "org.apache.cassandra.db.marshal.TupleType(org.apache.cassandra.db.marshal.UTF8Type,org.apache.cassandra.db.marshal.Int32Type,org.apache.cassandra.db.marshal.UTF8Type)",
			expected: TupleTypeInfo{
				NativeType{typ: TypeTuple},
				[]TypeInfo{
					NativeType{typ: TypeVarchar},
					NativeType{typ: TypeInt},
					NativeType{typ: TypeVarchar},
				},
			},
		},
		{
			name:   "vector_vector_inet",
			custom: "org.apache.cassandra.db.marshal.VectorType(org.apache.cassandra.db.marshal.VectorType(org.apache.cassandra.db.marshal.InetAddressType, 2), 3)",
			expected: VectorType{
				NativeType{typ: TypeCustom, custom: VECTOR_TYPE},
				VectorType{
					NativeType{typ: TypeCustom, custom: VECTOR_TYPE},
					NativeType{typ: TypeInet},
					2,
				},
				3,
			},
		},
		{
			name:   "map_int_vector_text",
			custom: "org.apache.cassandra.db.marshal.MapType(org.apache.cassandra.db.marshal.Int32Type,org.apache.cassandra.db.marshal.VectorType(org.apache.cassandra.db.marshal.UTF8Type, 10))",
			expected: CollectionType{
				NativeType{typ: TypeMap},
				NativeType{typ: TypeInt},
				VectorType{
					NativeType{typ: TypeCustom, custom: VECTOR_TYPE},
					NativeType{typ: TypeVarchar},
					10,
				},
			},
		},
		{
			name:   "set_map_vector_text_text",
			custom: "org.apache.cassandra.db.marshal.SetType(org.apache.cassandra.db.marshal.MapType(org.apache.cassandra.db.marshal.VectorType(org.apache.cassandra.db.marshal.Int32Type, 10),org.apache.cassandra.db.marshal.UTF8Type))",
			expected: CollectionType{
				NativeType{typ: TypeSet},
				nil,
				CollectionType{
					NativeType{typ: TypeMap},
					VectorType{
						NativeType{typ: TypeCustom, custom: VECTOR_TYPE},
						NativeType{typ: TypeInt},
						10,
					},
					NativeType{typ: TypeVarchar},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newFramer(nil, 0)
			f.writeShort(0)
			f.writeString(fmt.Sprintf("org.apache.cassandra.db.marshal.VectorType(%s, 2)", test.custom))
			parsedType := f.readTypeInfo()
			require.IsType(t, parsedType, VectorType{})
			vectorType := parsedType.(VectorType)
			assertEqual(t, "dimensions", 2, vectorType.Dimensions)
			assertDeepEqual(t, "vector", test.expected, vectorType.SubType)
		})
	}
}
