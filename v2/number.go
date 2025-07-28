package go_ora

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"unsafe"

	"github.com/sijms/go-ora/v2/converters"
	"github.com/sijms/go-ora/v2/util"
)

var (
	EMPTYSLICE = []byte{}
	ZERO1SLICE = []byte{'0'}
	INFINSLICE = []byte{'I', 'n', 'f', 'i', 'n', 'i', 't', 'y'}
)

type Number struct {
	data []byte
}

func (num Number) SetDataType(conn *Connection, par *ParameterInfo) error {
	par.DataType = NUMBER
	par.MaxLen = converters.MAX_LEN_NUMBER
	return nil
}
func (num *Number) isZero() bool {
	return len(num.data) > 0 && num.data[0] == 0x80
}

func (num *Number) isPositive() bool {
	return len(num.data) > 0 && num.data[0]&0x80 != 0
}

func NewNumberFromString(val string) (*Number, error) {
	mantissa := []byte(val)
	negative := mantissa[0] == '-'
	if negative {
		mantissa = mantissa[1:]
	}
	var (
		exp int
		err error
	)
	isFloat := false
	if i := bytes.Index(mantissa, []byte{'e'}); i >= 0 {
		exp, err = strconv.Atoi(string(mantissa[i+1:]))
		if err != nil {
			return nil, err
		}
		mantissa = mantissa[:i]
	}
	if i := bytes.Index(mantissa, []byte{'.'}); i >= 0 {
		mantissa = append(mantissa[:i], mantissa[i+1:]...)
		exp += i - 1
		isFloat = true
	}
	if !isFloat {
		exp += len(mantissa) - 1
	}
	ret := new(Number)
	err = ret.encode(mantissa, exp, negative)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func NewNumberFromInt64(val int64) (*Number, error) {
	mantissa := []byte(strconv.FormatInt(val, 10))
	negative := mantissa[0] == '-'
	if negative {
		mantissa = mantissa[1:]
	}
	exp := len(mantissa) - 1
	ret := new(Number)
	err := ret.encode(mantissa, exp, negative)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func NewNumberFromUint64(val uint64) (*Number, error) {
	mantissa := []byte(strconv.FormatUint(val, 10))
	exponent := len(mantissa) - 1
	ret := new(Number)
	err := ret.encode(mantissa, exponent, false)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func NewNumberFromFloat(val float64) (*Number, error) {
	if val == 0.0 {
		return &Number{data: []byte{128}}, nil
	}
	var (
		exponent int
		err      error
	)
	mantissa := []byte(strconv.FormatFloat(val, 'e', -1, 64))
	if i := bytes.Index(mantissa, []byte{'e'}); i >= 0 {
		exponent, err = strconv.Atoi(string(mantissa[i+1:]))
		if err != nil {
			return nil, err
		}
		mantissa = mantissa[:i]
	}

	negative := mantissa[0] == '-'
	if negative {
		mantissa = mantissa[1:]
	}

	if i := bytes.Index(mantissa, []byte{'.'}); i >= 0 {
		mantissa = append(mantissa[:i], mantissa[i+1:]...)
	}
	ret := new(Number)
	err = ret.encode(mantissa, exponent, negative)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (num *Number) encode(mantissa []byte, exp int, negative bool) error {
	trailingZeros := 0
	for i := len(mantissa) - 1; i >= 0 && mantissa[i] == '0'; i-- {
		trailingZeros++
	}
	mantissa = mantissa[:len(mantissa)-trailingZeros]
	if len(mantissa) == 0 {
		num.data = []byte{0x80}
		return nil
	}
	if exp%2 == 0 {
		mantissa = append([]byte{'0'}, mantissa...)
	}
	mantissaLen := len(mantissa)
	size := 1 + (mantissaLen+1)/2
	if negative && mantissaLen < 21 {
		size++
	}
	num.data = make([]byte, size)

	for i := 0; i < mantissaLen; i += 2 {
		b := 10 * (mantissa[i] - '0')
		if i < mantissaLen-1 {
			b += mantissa[i+1] - '0'
		}
		if negative {
			b = 100 - b
		}
		num.data[1+i/2] = b + 1
	}

	if negative && mantissaLen < 21 {
		num.data[len(num.data)-1] = 0x66
	}

	if exp < 0 {
		exp--
	}
	exp = (exp / 2) + 1
	if negative {
		num.data[0] = byte(exp+64) ^ 0x7f
	} else {
		num.data[0] = byte(exp+64) | 0x80
	}
	return nil
}

func (num *Number) decode() (strNum string, exp int, negative bool, err error) {
	if len(num.data) == 0 {
		err = fmt.Errorf("invalid NUMBER")
		return
	}
	if num.isZero() {
		strNum = "0"
		return
	}
	negative = num.data[0]&0x80 == 0
	if negative {
		exp = int(num.data[0]^0x7F) - 64
	} else {
		exp = int(num.data[0]&0x7F) - 64
	}

	if _isPosInf(num.data) || _isNegInf(num.data) {
		strNum = "Infinity"
		exp = 0
		return
	}

	buf := num.data[1:]
	if len(buf) == 0 {
		err = fmt.Errorf("invalid NUMBER")
		return
	}
	if negative && buf[len(buf)-1] == 0x66 {
		buf = buf[:len(buf)-1]
	}
	var output []byte = make([]byte, 0, 2*len(buf))
	for _, digit := range buf {
		digit--
		if negative {
			digit = 100 - digit
		}
		output = append(output, (digit/10)+'0', (digit%10)+'0')
	}
	exp = exp*2 - len(output)
	strNum = unsafe.String(unsafe.SliceData(output), len(output))
	return
}

func _isNegInf(b []byte) bool {
	return b[0] == 0 && len(b) == 1
}

func _isPosInf(b []byte) bool {
	// -1 =255
	return len(b) == 2 && b[0] == 255 && b[1] == 101
}

func (num *Number) Int64() (int64, error) {
	strNum, exp, negative, err := num.decode()
	if err != nil {
		return 0, err
	}
	mantissa, err := strconv.ParseInt(strNum, 10, 64)
	if err != nil {
		return 0, err
	}
	for exp > 0 {
		mantissa *= 10
		exp--
	}
	if negative && (mantissa>>63) == 0 {
		return -mantissa, nil
	}
	return mantissa, nil
}

func (num *Number) Uint64() (uint64, error) {
	strNum, exp, _, err := num.decode()
	if err != nil {
		return 0, err
	}
	mantissa, err := strconv.ParseUint(strNum, 10, 64)
	if err != nil {
		return 0, err
	}
	for exp > 0 {
		mantissa *= 10
		exp--
	}
	return mantissa, nil
}

func (num *Number) Float64() (float64, error) {
	strNum, exp, negative, err := num.decode()
	if err != nil {
		return 0, err
	}
	mantissa, err := strconv.ParseFloat(strNum, 64)
	if err != nil {
		return 0, err
	}
	absExponent := int(math.Abs(float64(exp)))
	if negative {
		return -math.Round(float64(mantissa)*math.Pow10(exp)*math.Pow10(absExponent)) / math.Pow10(absExponent), nil
	}
	return math.Round(float64(mantissa)*math.Pow10(exp)*math.Pow10(absExponent)) / math.Pow10(absExponent), nil
}

func (num *Number) String() (string, error) {
	strNum, exp, negative, err := num.decode()
	if err != nil {
		return "", err
	}
	// remove zeros from begining
	if len(strNum) > 1 {
		strNum = strings.TrimLeft(strNum, "0")
	}

	if exp > 0 {
		strNum += strings.Repeat("0", exp)
	} else if exp < 0 {
		pos := len(strNum) + exp // exp is negative
		if pos < 0 {
			pos = -pos
			strNum = strings.Repeat("0", pos) + strNum
			pos = 0
		}
		strNum = strNum[:pos] + "." + strNum[pos:]
		// remove zeros at rignt
		strNum = strings.TrimRight(strNum, "0")
	}
	if strNum[0] == '.' {
		strNum = "0" + strNum
	}
	if negative {
		strNum = "-" + strNum
	}
	return strNum, nil
}

func (num *Number) Slice2( /*tempIntBuffer []byte*/ sbh *util.SlideBufferHolder) ([]byte, error) {
	//fmt.Printf("num.data: %v\n", num.data)
	switch len(num.data) {
	case 0:
		return EMPTYSLICE, fmt.Errorf("invalid NUMBER")
	case 1:
		switch num.data[0] {
		case 0x80: // isZero
			return ZERO1SLICE, nil
		case 0: // _isNegInf
			return INFINSLICE, nil
		default:
			return EMPTYSLICE, fmt.Errorf("invalid NUMBER")
		}
	case 2:
		if num.data[0] == 255 && num.data[1] == 101 { //_isPosInf
			return INFINSLICE, nil
		}
		/*
			default:
				if num.data[0] == 0x80 { // isZero ? seems not the case, but: isZero-> return len(num.data) > 0 && num.data[0] == 0x80
					return "0", nil
				}
		*/
	}

	var neg int
	var exp int
	var buf []byte
	if num.data[0]&0x80 == 0 {
		if num.data[len(num.data)-1] == 0x66 {
			buf = num.data[1 : len(num.data)-1]
		} else {
			buf = num.data[1:]
		}
		neg = 1
		exp = 2 * (int(num.data[0]^0x7F) - 64 - len(buf))
	} else {
		buf = num.data[1:]
		neg = 0
		exp = 2 * (int(num.data[0]&0x7F) - 64 - len(buf))
	}

	var output []byte // = tempIntBuffer
	if exp >= 0 {     // eg 42e2 -> 4200, ints
		output = sbh.AllocBytes(2*len(buf) + neg + exp) // sbh.NewSlideBuffer().Alloc(2*len(buf) + neg + exp).Bytes()
		// output = make([]byte, 0, 2*len(buf)+neg+exp)
		//fmt.Printf("exp: %d, neg: %d, output.cap: %d, buf: %v\n", exp, neg, cap(output), buf)
		p := 0
		if neg == 1 {
			output[p] = '-'
			p += 1
		}

		// lead0s := true
		for i, digit := range buf {
			if neg == 1 {
				digit = 100 - digit - 1
			} else {
				digit -= 1
			}
			if i == 0 /* lead0s */ { // stripping leading 0
				if digit/10 != 0 {
					output[p] = (digit / 10) + '0'
					output[p+1] = (digit % 10) + '0'
					p += 2
				} else {
					output[p] = (digit % 10) + '0'
					p += 1
				}
				//lead0s = false
			} else {
				output[p] = (digit / 10) + '0'
				output[p+1] = (digit % 10) + '0'
				p += 2
			}
		}

		// adding trailing 0s
		for i := exp; i > 0; i-- {
			output[p] = '0'
			p += 1
		}

		// fmt.Printf("converted number: %s\n", string(output))
		return output[:p], nil //unsafe.String(unsafe.SliceData(output[:p]), len(output[:p]))
	} else { // exp < 0, like 42e-2, floats
		dotAt := len(buf) + exp/2 // '.' is always between num.data bytes, exp is always even (2x)

		if /* 2*len(buf)+exp */ dotAt > 0 { // eg 42e-1 => 4.2 (+dot), but technically it is [04][20e]-2, 2*len(buf)+exp == 2
			output = sbh.AllocBytes(2*len(buf) + neg + 1) // sbh.NewSlideBuffer().Alloc(2*len(buf) + neg + 1).Bytes() // +1 for '.'
		} else { // 2*len(buf)+exp <= 0
			// if 2*len(buf) + exp == 0 eg 42e-2=0.42 -> 2*len(buf)+neg+2 or -exp+neg+2 (+dot+leading 0)
			// if 2*len(buf) + exp < 0, eg 42e-3=0.042 -> -exp+neg+2 (+dot+leading 0)
			output = sbh.AllocBytes(neg + 2 - exp) //sbh.NewSlideBuffer().Alloc(neg + 2 - exp).Bytes() // +2 for '0.'
		}

		p := 0
		if neg == 1 {
			output[p] = '-'
			p += 1
		}

		//fmt.Printf("exp: %d, neg: %d, dotAt: %d, 2len+exp: %d, output.cap: %d, buf: %v\n", exp, neg, dotAt, 2*len(buf)+exp, cap(output), buf)

		if dotAt == 0 {
			output[p] = '0'
			p += 1
		} else if dotAt < 0 { // adding leading 0s after '0.'
			output[p] = '0'
			output[p+1] = '.'
			p += 2
			for i := -dotAt * 2; i > 0; i-- {
				output[p] = '0'
				p += 1
			}
		}

		// lead0s := dotAt > 0
		for i, digit := range buf {
			if neg == 1 {
				digit = 100 - digit - 1
			} else {
				digit -= 1
			}
			if i == 0 && dotAt > 0 /*lead0s*/ { // stripping leading 0
				if digit/10 != 0 {
					output[p] = (digit / 10) + '0'
					output[p+1] = (digit % 10) + '0'
					p += 2
				} else {
					output[p] = (digit % 10) + '0'
					p += 1
				}
				// lead0s = false
			} else if dotAt == 0 {
				output[p] = '.'
				output[p+1] = (digit / 10) + '0'
				output[p+2] = (digit % 10) + '0'
				p += 3
			} else {
				output[p] = (digit / 10) + '0'
				output[p+1] = (digit % 10) + '0'
				p += 2
			}
			dotAt -= 1
		}

		// stripping trailing 0, let's hope there is no '42.0' -> '42.'
		p = p - 1
		for ; p >= 0 && output[p] == '0'; p-- {
		}
		/*
			if output[j] == '.' {
				j --
			}
		*/

		// fmt.Printf("converted number: %s, %s\n", string(output), string(output[:j+1]))
		return output[:p+1], nil //unsafe.String(unsafe.SliceData(output), len(output))
	}
}

func NewNumber(n interface{}) (*Number, error) {
	var err error
	n, err = getValue(n)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, nil
	}
	rType := reflect.TypeOf(n)
	rValue := reflect.ValueOf(n)
	if tSigned(rType) {
		return NewNumberFromInt64(rValue.Int())
	}
	if tUnsigned(rType) {
		return NewNumberFromUint64(rValue.Uint())
	}
	//if f32, ok := col.(float32); ok {
	//	return strconv.ParseFloat(fmt.Sprint(f32), 64)
	//}
	if tFloat(rType) {
		return NewNumberFromFloat(rValue.Float())
	}
	if rType == tyNumber {
		if num, ok := n.(Number); ok {
			return &num, nil
		}
		return nil, errors.New("conversion of unsupported type to number")
	}
	switch rType.Kind() {
	case reflect.Bool:
		if rValue.Bool() {
			return NewNumberFromInt64(1)
		} else {
			return NewNumberFromInt64(0)
		}
	case reflect.String:
		return NewNumberFromString(rValue.String())
	default:
		return nil, errors.New("conversion of unsupported type to number")
	}
}
