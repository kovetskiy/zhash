package libdeploy

import (
	"errors"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

type recoverCallback func(r interface{})

func failOnRecover(callback recoverCallback) {
	if r := recover(); r != nil {
		callback(r)
	}
}

func readConfig(path string) (Config, error) {
	conf := NewConfig()
	fd, err := os.Open(path)
	if err != nil {
		return conf, err
	}
	defer fd.Close()
	err = conf.ReadConfig(fd)
	if err != nil {
		return NewConfig(), err
	}
	return conf, nil
}

var configs = map[string]string{
	"valid":   "test_valid.toml",
	"invalid": "test_invalid.toml",
}

func TestValidateValid(t *testing.T) {
	conf, err := readConfig(configs["valid"])
	if err != nil {
		t.Errorf("Error loading config: %v", err)
	}
	errs := conf.Validate()

	if len(errs) > 0 {
		t.Errorf("config doesn't validates, but it should; Errors: %v", errs)
	}
}

func TestValidateInvalid(t *testing.T) {
	conf, err := readConfig(configs["invalid"])
	if err != nil {
		t.Errorf(" Error loading config: %v", err)
	}
	errs := conf.Validate()

	if len(errs) == 0 {
		t.Errorf("config validates, but it should fail")
	} else {
		for _, err = range errs {
			t.Log(err.Error())
		}
	}
}

var parseSetArgsTests = []struct {
	pathval, path string
	val           interface{}
}{
	{"setter.time:2014-05-09T12:01:05Z", "setter.time", time.Date(2014, 05, 9, 12, 01, 05, 0, time.UTC)},
	{"setter.int:214", "setter.int", 214},
	{"setter.float:21.4", "setter.float", 21.4},
	{"setter.bool:true", "setter.bool", true},
	{"setter.string:Tests env", "setter.string", "Tests env"},
}

func TestParseSetArgs(t *testing.T) {
	for i, test := range parseSetArgsTests {
		v, p := ParseSetArgument(test.pathval)
		if v != test.val || p != test.path {
			t.Errorf("#%d: ParseSetArgument(%s)=%#v,%#v; want %#v, %#v", i, test.pathval, v, p, test.val, test.path)
		}
	}
}

var setTests = []struct {
	path  string
	value interface{}
}{
	{"meta.email", "s@t.r"},
	{"meta.bar", 10},
	{"resources.foo", map[string]interface{}{"provider": "bar", "pool": "baz"}},
	{"foo.bar.baz", 10.1},
}

func TestSetPath(t *testing.T) {
	conf, err := readConfig(configs["valid"])
	if err != nil {
		t.Errorf("Error loading config: %v", err)
	}
	for _, test := range setTests {
		conf.SetPath(test.value, test.path)
	}

	for i, test := range setTests {
		val := conf.GetPath(strings.Split(test.path, ".")...)
		if !reflect.DeepEqual(val, test.value) {
			t.Errorf("#%d: conf[%s]%#v != %#v", i, test.path, val, test.value)
		}
	}
}

func TestWriteConfig(t *testing.T) {
	conf, err := readConfig(configs["valid"])
	if err != nil {
		t.Errorf("Error loading config: %v", err)
	}
	f, _ := os.OpenFile(os.DevNull, os.O_RDWR, 0666)
	defer f.Close()
	if err := conf.WriteConfig(f); err != nil {
		t.Errorf("Errors while writing config: %s", err.Error())
	}
}

func TestConfigReader(t *testing.T) {
	conf, err := readConfig(configs["valid"])
	if err != nil {
		t.Errorf("Error loading config: %v", err)
	}
	r := conf.Reader()
	f, err := os.OpenFile(os.DevNull, os.O_RDWR, 0666)
	if err != nil {
		t.Error("Error opening DevNull:", err)
	}
	defer f.Close()
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		t.Error("Error reading from conf.Reader():", err)
	}
	f.Write(buf)
}

type TestGet struct {
	path  []string
	value interface{}
}

func TestGetPath(t *testing.T) {
	var getTests []TestGet = []TestGet{
		{[]string{"domain"}, "t6"},
		{[]string{"meta", "owner"}, "e.persienko"},
		{[]string{"resources", "mongo_single", "provider"}, "dbfarm"},
		{[]string{"meta", "foo", "bar"}, nil},
		{[]string{}, nil},
	}

	conf, err := readConfig(configs["valid"])
	if err != nil {
		t.Errorf("Error loading config: %v", err)
	}

	for i, test := range getTests {
		value := conf.GetPath(test.path...)
		if value != test.value {
			t.Errorf("#%d: GetPath(%s)=%#v; want %#v", i, test.path, value, test.value)
		}
	}
}

func TestGetMapSuccess(t *testing.T) {
	var mapGetTests []TestGet = []TestGet{
		{[]string{"meta"}, map[string]interface{}{
			"owner":       "e.persienko",
			"email":       "some@test.ru",
			"description": "Tests",
			"bool":        true,
			"bool_f":      false,
		}},
	}

	conf, err := readConfig(configs["valid"])
	if err != nil {
		t.Errorf("Error loading config: %v", err)
	}

	for i, test := range mapGetTests {
		m, err := conf.GetMap(test.path...)
		if err != nil {
			t.Errorf("#%d: GetMap(%s) caused error: %v", i, test.path, err)
		}
		if !reflect.DeepEqual(m, test.value) {
			t.Errorf("#%d: GetMap(%s)=%#v; want %#v", i, test.path, m, test.value)
		}
	}
}

func TestGetMapFail(t *testing.T) {
	var mapGetTests []TestGet = []TestGet{
		{[]string{"meta", "foo", "bar"}, map[string]interface{}{}},
		{[]string{"domain"}, map[string]interface{}{}},
	}
	conf, err := readConfig(configs["valid"])
	if err != nil {
		t.Errorf("Error loading config: %v", err)
	}

	for i, test := range mapGetTests {
		m, err := conf.GetMap(test.path...)
		if err == nil {
			t.Errorf("#%d: GetMap(%s) doesn't cause error, but it should", i, test.path)
		} else {
			t.Logf("Err: %s", err)
		}
		if !reflect.DeepEqual(m, test.value) {
			t.Errorf("#%d: GetMap(%s)=%#v; want %#v", i, test.path, m, test.value)
		}
	}
}

func TestGetSliceSuccess(t *testing.T) {
	var sliceGetTests []TestGet = []TestGet{
		{[]string{"resources", "conf", "depends"}, []interface{}{"mysql_single", "mongo_single", "node_single"}},
	}

	conf, err := readConfig(configs["valid"])
	if err != nil {
		t.Errorf("Error loading config: %v", err)
	}

	for i, test := range sliceGetTests {
		m, err := conf.GetSlice(test.path...)
		if err != nil {
			t.Errorf("#%d: GetSlice(%s) caused error: %v", i, test.path, err)
		}
		if !reflect.DeepEqual(m, test.value) {
			t.Errorf("#%d: GetSlice(%s)=%#v; want %#v", i, test.path, m, test.value)
		}
	}
}

func TestGetSliceFail(t *testing.T) {
	var sliceGetTests []TestGet = []TestGet{
		{[]string{"meta", "foo", "bar"}, []interface{}{}},
		{[]string{"domain"}, []interface{}{}},
	}

	conf, err := readConfig(configs["valid"])
	if err != nil {
		t.Errorf("Error loading config: %v", err)
	}

	for i, test := range sliceGetTests {
		m, err := conf.GetSlice(test.path...)
		if err == nil {
			t.Errorf("#%d: GetSlice(%s) doesn't cause error, but it should", i, test.path)
		} else {
			t.Logf("Err: %s", err)
		}
		if !reflect.DeepEqual(m, test.value) {
			t.Errorf("#%d: GetSlice(%s)=%#v; want %#v", i, test.path, m, test.value)
		}
	}
}

func TestGetStringSliceSuccess(t *testing.T) {
	var stringSliceGetTests []TestGet = []TestGet{
		{[]string{"resources", "conf", "depends"}, []string{"mysql_single", "mongo_single", "node_single"}},
	}
	conf, err := readConfig(configs["valid"])
	if err != nil {
		t.Errorf("Error loading config: %v", err)
	}

	for i, test := range stringSliceGetTests {
		m, err := conf.GetStringSlice(test.path...)
		if err != nil {
			t.Errorf("#%d: GetStringSlice(%s) caused error: %v", i, test.path, err)
		}
		if !reflect.DeepEqual(m, test.value) {
			t.Errorf("#%d: GetStringSlice(%s)=%#v; want %#v", i, test.path, m, test.value)
		}
	}
}

func TestGetStringSliceFail(t *testing.T) {
	var stringSliceGetTests []TestGet = []TestGet{
		{[]string{"getters", "intSlice"}, []string{}},
		{[]string{"meta", "foo", "bar"}, []string{}},
		{[]string{"domain"}, []string{}},
	}
	conf, err := readConfig(configs["valid"])
	if err != nil {
		t.Errorf("Error loading config: %v", err)
	}

	for i, test := range stringSliceGetTests {
		m, err := conf.GetStringSlice(test.path...)
		if err == nil {
			t.Errorf("#%d: GetStringSlice(%s) doesn't cause error, but it should", i, test.path)
		} else {
			t.Logf("Err: %s", err)
		}
		if !reflect.DeepEqual(m, test.value) {
			t.Errorf("#%d: GetStringSlice(%s)=%#v; want %#v", i, test.path, m, test.value)
		}
	}
}

func TestGetIntSuccess(t *testing.T) {
	var intGetTests []TestGet = []TestGet{
		{[]string{"getters", "int"}, int64(10)},
	}
	conf, err := readConfig(configs["valid"])
	if err != nil {
		t.Errorf("Error loading config: %v", err)
	}

	for i, test := range intGetTests {
		in, err := conf.GetInt(test.path...)
		if err != nil {
			v := conf.GetPath(test.path...)
			t.Errorf("#%d: GetInt(%s) caused error: %v, %s.(type) = %T", i, test.path, err, test.path, v)
		}
		if in != test.value {
			t.Errorf("#%d: GetInt(%s)=%d; want %d", i, test.path, in, test.value)
		}
	}
}

func TestGetIntFail(t *testing.T) {
	var intGetTests []TestGet = []TestGet{
		{[]string{"meta", "foo", "bar"}, int64(0)},
		{[]string{"domain"}, int64(0)},
	}
	conf, err := readConfig(configs["valid"])
	if err != nil {
		t.Errorf("Error loading config: %v", err)
	}

	for i, test := range intGetTests {
		in, err := conf.GetInt(test.path...)
		if err == nil {
			t.Errorf("#%d: GetInt(%s) doesn't cause error, but it should", i, test.path)
		} else {
			t.Logf("Err: %s", err)
		}
		if in != test.value {
			t.Errorf("#%d: GetInt(%s)=%d; want %d", i, test.path, in, test.value)
		}
	}
}

func TestGetFloatSuccess(t *testing.T) {
	var floatGetTests []TestGet = []TestGet{
		{[]string{"getters", "float"}, 10.1},
		{[]string{"getters", "int"}, 10.0},
	}

	conf, err := readConfig(configs["valid"])
	if err != nil {
		t.Errorf("Error loading config: %v", err)
	}

	for i, test := range floatGetTests {
		f, err := conf.GetFloat(test.path...)
		if f != test.value {
			t.Errorf("#%d: GetFloat(%s)=%f; want %v", i, test.path, f, test.value)
		}
		if err != nil {
			t.Errorf("#%d: GetFloat(%s) returned error: %s", i, test.path, err)
		}
	}
}

func TestGetFloatFail(t *testing.T) {
	var floatGetTests []TestGet = []TestGet{
		{[]string{"meta", "foo", "bar"}, 0.0},
		{[]string{"meta", "bool"}, 0.0},
	}

	conf, err := readConfig(configs["valid"])
	if err != nil {
		t.Errorf("Error loading config: %v", err)
	}

	for i, test := range floatGetTests {
		f, err := conf.GetFloat(test.path...)
		if f != test.value {
			t.Errorf("#%d: GetFloat(%s)=%f; want %v", i, test.path, f, test.value)
		}
		if err == nil {
			t.Errorf("#%d: GetFloat(%s) doesn't return any error, but it should fail", i, test.path)
		}
	}
}

func TestGetStringSuccess(t *testing.T) {
	var stringGetTests []TestGet = []TestGet{
		{[]string{"domain"}, "t6"},
	}
	conf, err := readConfig(configs["valid"])
	if err != nil {
		t.Errorf("Error loading config: %v", err)
	}

	for i, test := range stringGetTests {
		f, err := conf.GetString(test.path...)
		if f != test.value {
			t.Errorf("#%d: GetString(%s)=%s; want %v", i, test.path, f, test.value)
		}
		if err != nil {
			t.Errorf("#%d: GetString(%s) returned error: %s", i, test.path, err)
		}
	}
}

func TestGetStringFail(t *testing.T) {
	var stringGetTests []TestGet = []TestGet{
		{[]string{"meta", "bar", "bazzar"}, ""},
		{[]string{"meta"}, ""},
	}
	conf, err := readConfig(configs["valid"])
	if err != nil {
		t.Errorf("Error loading config: %v", err)
	}

	for i, test := range stringGetTests {
		f, err := conf.GetString(test.path...)
		if f != test.value {
			t.Errorf("#%d: GetString(%s)=%s; want %v", i, test.path, f, test.value)
		}
		if err == nil {
			t.Errorf("#%d: GetString(%s) doesn't return any error, but it should fail", i, test.path)
		}
	}
}

func TestGetBoolSuccess(t *testing.T) {
	var stringGetTests []TestGet = []TestGet{
		{[]string{"meta", "bool"}, true},
		{[]string{"meta", "bool_f"}, false},
	}
	conf, err := readConfig(configs["valid"])
	if err != nil {
		t.Errorf("Error loading config: %v", err)
	}

	for i, test := range stringGetTests {
		f, err := conf.GetBool(test.path...)
		if f != test.value {
			t.Errorf("#%d: GetBool(%s)=%s; want %v", i, test.path, f, test.value)
		}
		if err != nil {
			t.Errorf("#%d: GetBool(%s) returned error: %s", i, test.path, err)
		}
	}
}

func TestGetBoolFail(t *testing.T) {
	var stringGetTests []TestGet = []TestGet{
		{[]string{"meta", "bar", "bazzar"}, false},
		{[]string{"meta"}, false},
	}
	conf, err := readConfig(configs["valid"])
	if err != nil {
		t.Errorf("Error loading config: %v", err)
	}

	for i, test := range stringGetTests {
		b, err := conf.GetBool(test.path...)
		t.Logf("b=\"%#v\", b.(type)=\"%T\"", b, b)
		if b != test.value {
			t.Errorf("#%d: GetBool(%s)=%v; want %v", i, test.path, b, test.value)
		}
		if err == nil {
			t.Errorf("#%d: GetBool(%s) doesn't return any error, but it should fail", i, test.path)
		}
	}
}

func TestResolveDomainNameSuccess(t *testing.T) {
	ips, err := ResolveDomainName("ns.s")
	if err != nil {
		t.Errorf("Error getting ips for ns.s: %s", err)
	}

	if !reflect.DeepEqual(ips, []string{"192.168.20.2"}) {
		t.Errorf("ResolveDomainName(\"ns.s\") returned wrong ips: %s", ips)
	}
}

func TestResolveDomainNameFail(t *testing.T) {
	ips, err := ResolveDomainName("nasos.s")
	if ips != nil {
		t.Errorf("ResolveDomainName(\"nasos.s\") returned wrong ips: %s", ips)
	}
	if err == nil {
		t.Errorf("ResolveDomainName(\"nasos.s\") doesn't returned any error, but it should!")
	}

}

func TestMarshalToJsonReaderSuccess(t *testing.T) {
	value := map[string]string{"a": "b", "c": "d"}
	marshaledValue := "{\"a\":\"b\",\"c\":\"d\"}"
	r, err := MarshalToJsonReader(value)
	if err != nil {
		t.Errorf("Error marshaling %#v: %s", value, err)
	}

	res, _ := ioutil.ReadAll(r)

	if string(res) != marshaledValue {
		t.Errorf("Marshalled data is wrong: %s != %s", res, marshaledValue)
	}
}

type buggyStruct struct {
	Id int `json:"id"`
}

func (b buggyStruct) MarshalJSON() ([]byte, error) {
	return nil, errors.New("Baka!")
}

func TestMarshalToJsonReaderFail(t *testing.T) {
	value := buggyStruct{Id: 10}
	_, err := MarshalToJsonReader(value)
	if err == nil {
		t.Errorf("Marshal should return error, but it returns nil")
	}
}

func TestToStringSuccess(t *testing.T) {
	conf, err := readConfig(configs["valid"])
	if err != nil {
		t.Errorf("Error reading config")
	}

	t.Logf("Config: %s", conf)
}

func TestToStringFail(t *testing.T) {
	conf, err := readConfig(configs["valid"])
	if err != nil {
		t.Errorf("Error reading config")
	}

	value := buggyStruct{Id: 10}
	conf.Set(value, "meta", "bug")

	t.Logf("Config: %s", conf)
}
