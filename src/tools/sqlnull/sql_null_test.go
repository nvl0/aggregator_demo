package sqlnull_test

import (
	"testing"
	"time"

	"aggregator/src/tools/sqlnull"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNullInt64Scan(t *testing.T) {
	tests := []struct {
		name       string
		input      any
		wantValid  bool
		wantInt64  int64
		assertFunc func(t *testing.T, ni sqlnull.NullInt64, err error)
	}{
		{
			name:      "int64 значение делает Valid=true",
			input:     int64(42),
			wantValid: true,
			wantInt64: 42,
		},
		{
			name:      "nil делает Valid=false",
			input:     nil,
			wantValid: false,
			wantInt64: 0,
		},
		{
			// sql.NullInt64.Scan напрямую ошибается на дробном float64
			// (strconv.ParseInt("7.9", ...) не парсится), поэтому срабатывает
			// fallback: NullFloat64.Scan + int64(f.Float64) с усечением дробной части
			name:      "дробный float64 конвертируется в int64 через fallback с усечением",
			input:     7.9,
			wantValid: true,
			wantInt64: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ni sqlnull.NullInt64

			err := ni.Scan(tt.input)

			require.NoError(t, err)
			assert.Equal(t, tt.wantValid, ni.Valid)
			assert.Equal(t, tt.wantInt64, ni.Int64)
		})
	}
}

func TestNullInt64GetInt(t *testing.T) {
	valid := sqlnull.NewInt64(5)
	assert.Equal(t, 5, valid.GetInt())

	var invalid sqlnull.NullInt64
	assert.Equal(t, 0, invalid.GetInt())
}

func TestNullInt64Value(t *testing.T) {
	valid := sqlnull.NewInt64(5)
	v, err := valid.Value()
	require.NoError(t, err)
	assert.Equal(t, int64(5), v)

	var invalid sqlnull.NullInt64
	v, err = invalid.Value()
	require.NoError(t, err)
	assert.Nil(t, v)
}

func TestNullInt64JSONRoundTrip(t *testing.T) {
	valid := sqlnull.NewInt64(5)
	b, err := valid.MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, "5", string(b))

	var decoded sqlnull.NullInt64
	require.NoError(t, decoded.UnmarshalJSON(b))
	assert.Equal(t, valid, decoded)

	var invalid sqlnull.NullInt64
	b, err = invalid.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, "null", string(b))
}

func TestNullBoolScan(t *testing.T) {
	var nb sqlnull.NullBool

	require.NoError(t, nb.Scan(true))
	assert.True(t, nb.Valid)
	assert.True(t, nb.Bool)

	var nilBool sqlnull.NullBool
	require.NoError(t, nilBool.Scan(nil))
	assert.False(t, nilBool.Valid)
}

func TestNullBoolValue(t *testing.T) {
	var nb sqlnull.NullBool
	require.NoError(t, nb.Scan(true))

	v, err := nb.Value()
	require.NoError(t, err)
	assert.Equal(t, true, v)

	var invalid sqlnull.NullBool
	v, err = invalid.Value()
	require.NoError(t, err)
	assert.Nil(t, v)
}

func TestNullBoolJSONRoundTrip(t *testing.T) {
	var nb sqlnull.NullBool
	require.NoError(t, nb.Scan(true))

	b, err := nb.MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, "true", string(b))

	var decoded sqlnull.NullBool
	require.NoError(t, decoded.UnmarshalJSON(b))
	assert.Equal(t, nb, decoded)

	var invalid sqlnull.NullBool
	b, err = invalid.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, "null", string(b))
}

func TestNullFloat64Scan(t *testing.T) {
	var nf sqlnull.NullFloat64

	require.NoError(t, nf.Scan(3.14))
	assert.True(t, nf.Valid)
	assert.InDelta(t, 3.14, nf.Float64, 0.0001)

	var nilFloat sqlnull.NullFloat64
	require.NoError(t, nilFloat.Scan(nil))
	assert.False(t, nilFloat.Valid)
}

func TestNullFloat64Value(t *testing.T) {
	valid := sqlnull.NewFloat64(3.14)
	v, err := valid.Value()
	require.NoError(t, err)
	assert.InDelta(t, 3.14, v.(float64), 0.0001)

	var invalid sqlnull.NullFloat64
	v, err = invalid.Value()
	require.NoError(t, err)
	assert.Nil(t, v)
}

func TestNullFloat64JSONRoundTrip(t *testing.T) {
	valid := sqlnull.NewFloat64(3.5)
	b, err := valid.MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, "3.5", string(b))

	var decoded sqlnull.NullFloat64
	require.NoError(t, decoded.UnmarshalJSON(b))
	assert.Equal(t, valid, decoded)

	var invalid sqlnull.NullFloat64
	b, err = invalid.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, "null", string(b))
}

func TestNullStringScan(t *testing.T) {
	var ns sqlnull.NullString

	require.NoError(t, ns.Scan("hello"))
	assert.True(t, ns.Valid)
	assert.Equal(t, "hello", ns.String)

	var nilString sqlnull.NullString
	require.NoError(t, nilString.Scan(nil))
	assert.False(t, nilString.Valid)
}

func TestNullStringValue(t *testing.T) {
	valid := sqlnull.NewString("hello")
	v, err := valid.Value()
	require.NoError(t, err)
	assert.Equal(t, "hello", v)

	var invalid sqlnull.NullString
	v, err = invalid.Value()
	require.NoError(t, err)
	assert.Nil(t, v)
}

func TestNullStringOptionalResult(t *testing.T) {
	valid := sqlnull.NewString("hello")
	assert.Equal(t, "hello", valid.OptionalResult())

	var invalid sqlnull.NullString
	assert.Equal(t, "-", invalid.OptionalResult())
}

func TestNullStringJSONRoundTrip(t *testing.T) {
	valid := sqlnull.NewString("hello")
	b, err := valid.MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `"hello"`, string(b))

	var decoded sqlnull.NullString
	require.NoError(t, decoded.UnmarshalJSON(b))
	assert.Equal(t, valid, decoded)

	var invalid sqlnull.NullString
	b, err = invalid.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, "null", string(b))
}

func TestNullTimeScan(t *testing.T) {
	now := time.Now()

	var nt sqlnull.NullTime
	require.NoError(t, nt.Scan(now))
	assert.True(t, nt.Valid)
	assert.True(t, nt.Time.Equal(now))

	var nilTime sqlnull.NullTime
	require.NoError(t, nilTime.Scan(nil))
	assert.False(t, nilTime.Valid)
}

func TestNullTimeValue(t *testing.T) {
	now := time.Now()
	valid := sqlnull.NewNullTime(now)

	v, err := valid.Value()
	require.NoError(t, err)
	assert.True(t, v.(time.Time).Equal(now))

	var invalid sqlnull.NullTime
	v, err = invalid.Value()
	require.NoError(t, err)
	assert.Nil(t, v)
}

func TestNullTimeFormat(t *testing.T) {
	valid := sqlnull.NewNullTime(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC))
	assert.Equal(t, "2026-01-02", valid.Format("2006-01-02"))

	var invalid sqlnull.NullTime
	assert.Equal(t, "-", invalid.Format("2006-01-02"))
}

func TestNullTimeJSONRoundTrip(t *testing.T) {
	valid := sqlnull.NewNullTime(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))
	b, err := valid.MarshalJSON()
	require.NoError(t, err)

	var decoded sqlnull.NullTime
	require.NoError(t, decoded.UnmarshalJSON(b))
	assert.True(t, decoded.Valid)
	assert.True(t, decoded.Time.Equal(valid.Time))

	var invalid sqlnull.NullTime
	b, err = invalid.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, "null", string(b))

	var decodedInvalid sqlnull.NullTime
	require.NoError(t, decodedInvalid.UnmarshalJSON([]byte("null")))
	assert.False(t, decodedInvalid.Valid)
}
