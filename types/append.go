package types

import (
	"database/sql/driver"
	"encoding/hex"
	"reflect"
	"strconv"
	"time"

	"gopkg.in/pg.v4/internal/parser"
)

func Append(b []byte, v interface{}, quote int) []byte {
	switch v := v.(type) {
	case nil:
		return AppendNull(b, quote)
	case bool:
		return appendBool(b, v)
	case int8:
		return strconv.AppendInt(b, int64(v), 10)
	case int16:
		return strconv.AppendInt(b, int64(v), 10)
	case int32:
		return strconv.AppendInt(b, int64(v), 10)
	case int64:
		return strconv.AppendInt(b, int64(v), 10)
	case int:
		return strconv.AppendInt(b, int64(v), 10)
	case uint8:
		return strconv.AppendUint(b, uint64(v), 10)
	case uint16:
		return strconv.AppendUint(b, uint64(v), 10)
	case uint32:
		return strconv.AppendUint(b, uint64(v), 10)
	case uint64:
		return strconv.AppendUint(b, v, 10)
	case uint:
		return strconv.AppendUint(b, uint64(v), 10)
	case float32:
		return appendFloat(b, float64(v))
	case float64:
		return appendFloat(b, v)
	case string:
		return AppendString(b, v, quote)
	case time.Time:
		return AppendTime(b, v, quote)
	case []byte:
		return appendBytes(b, v, quote)
	case ValueAppender:
		return appendAppender(b, v, quote)
	case driver.Valuer:
		return appendDriverValuer(b, v, quote)
	default:
		return appendValue(b, reflect.ValueOf(v), quote)
	}
}

func appendError(b []byte, err error) []byte {
	b = append(b, "?!("...)
	b = append(b, err.Error()...)
	b = append(b, ')')
	return b
}

func AppendNull(b []byte, quote int) []byte {
	if quote == 1 {
		return append(b, "NULL"...)
	} else {
		return nil
	}
}

func appendBool(dst []byte, v bool) []byte {
	if v {
		return append(dst, "TRUE"...)
	}
	return append(dst, "FALSE"...)
}

func appendFloat(dst []byte, v float64) []byte {
	return strconv.AppendFloat(dst, v, 'f', -1, 64)
}

func AppendString(b []byte, s string, quote int) []byte {
	if quote == 2 {
		b = append(b, '"')
	} else if quote == 1 {
		b = append(b, '\'')
	}

	for i := 0; i < len(s); i++ {
		c := s[i]

		if c == '\000' {
			continue
		}

		if quote >= 1 {
			if c == '\'' {
				b = append(b, '\'', '\'')
				continue
			}
		}

		if quote == 2 {
			if c == '"' {
				b = append(b, '\\', '"')
				continue
			}
			if c == '\\' {
				b = append(b, '\\', '\\')
				continue
			}
		}

		b = append(b, c)
	}

	if quote >= 2 {
		b = append(b, '"')
	} else if quote == 1 {
		b = append(b, '\'')
	}

	return b
}

func appendBytes(b []byte, bytes []byte, quote int) []byte {
	if bytes == nil {
		return AppendNull(b, quote)
	}

	if quote == 1 {
		b = append(b, '\'')
	}

	tmp := make([]byte, hex.EncodedLen(len(bytes)))
	hex.Encode(tmp, bytes)
	b = append(b, "\\x"...)
	b = append(b, tmp...)

	if quote == 1 {
		b = append(b, '\'')
	}

	return b
}

func AppendStringStringMap(b []byte, m map[string]string, quote int) []byte {
	if m == nil {
		return AppendNull(b, quote)
	}

	if quote == 1 {
		b = append(b, '\'')
	}

	for key, value := range m {
		b = AppendString(b, key, 2)
		b = append(b, '=', '>')
		b = AppendString(b, value, 2)
		b = append(b, ',')
	}
	if len(m) > 0 {
		b = b[:len(b)-1] // Strip trailing comma.
	}

	if quote == 1 {
		b = append(b, '\'')
	}

	return b
}

func appendDriverValuer(b []byte, v driver.Valuer, quote int) []byte {
	value, err := v.Value()
	if err != nil {
		return appendError(b, err)
	}
	return Append(b, value, quote)
}

func AppendField(b []byte, field string, quote int) []byte {
	return appendField(b, parser.NewString(field), quote)
}

func AppendFieldBytes(b []byte, field []byte, quote int) []byte {
	return appendField(b, parser.New(field), quote)
}

func appendField(b []byte, p *parser.Parser, quote int) []byte {
	var quoted bool
	for p.Valid() {
		c := p.Read()

		switch c {
		case '*':
			if !quoted {
				b = append(b, '*')
				continue
			}
		case '.':
			if quoted && quote == 1 {
				b = append(b, '"')
				quoted = false
			}
			b = append(b, '.')
			if p.Skip('*') {
				b = append(b, '*')
			} else if quote == 1 {
				b = append(b, '"')
				quoted = true
			}
			continue
		}

		if !quoted && quote == 1 {
			b = append(b, '"')
			quoted = true
		}
		if quote == 1 && c == '"' {
			b = append(b, '"', '"')
		} else {
			b = append(b, c)
		}

	}
	if quote == 1 && quoted {
		b = append(b, '"')
	}
	return b
}

func appendAppender(b []byte, v ValueAppender, quote int) []byte {
	bb, err := v.AppendValue(b, quote)
	if err != nil {
		return appendError(b, err)
	}
	return bb
}
