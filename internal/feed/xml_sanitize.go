package feed

import (
	"bytes"
	"fmt"
	"io"
)

var (
	cdataStart = []byte("<![CDATA[")
	cdataEnd   = []byte("]]>")
	xmlAmp     = []byte("&amp;")
	xmlApos    = []byte("&apos;")
	xmlGt      = []byte("&gt;")
	xmlLt      = []byte("&lt;")
	xmlQuot    = []byte("&quot;")
)

func readerWithEscapedAmpersands(reader io.Reader) (io.Reader, error) {
	raw, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read feed XML: %w", err)
	}
	return bytes.NewReader(escapeInvalidAmpersands(raw)), nil
}

func escapeInvalidAmpersands(raw []byte) []byte {
	var out bytes.Buffer
	for i := 0; i < len(raw); i++ {
		if bytes.HasPrefix(raw[i:], cdataStart) {
			end := bytes.Index(raw[i+len(cdataStart):], cdataEnd)
			if end < 0 {
				out.Write(raw[i:])
				break
			}
			cdataLen := len(cdataStart) + end + len(cdataEnd)
			out.Write(raw[i : i+cdataLen])
			i += cdataLen - 1
			continue
		}
		if raw[i] == '&' && !isXMLEntity(raw[i:]) {
			out.WriteString("&amp;")
			continue
		}
		out.WriteByte(raw[i])
	}
	return out.Bytes()
}

func isXMLEntity(raw []byte) bool {
	if bytes.HasPrefix(raw, xmlAmp) || bytes.HasPrefix(raw, xmlApos) || bytes.HasPrefix(raw, xmlGt) ||
		bytes.HasPrefix(raw, xmlLt) || bytes.HasPrefix(raw, xmlQuot) {
		return true
	}
	if len(raw) < 4 || raw[1] != '#' {
		return false
	}
	return isNumericEntity(raw)
}

func isNumericEntity(raw []byte) bool {
	if raw[2] == 'x' || raw[2] == 'X' {
		return hasEntityDigits(raw[3:], isHexDigit)
	}
	return hasEntityDigits(raw[2:], isDecimalDigit)
}

func hasEntityDigits(raw []byte, valid func(byte) bool) bool {
	for i, value := range raw {
		if value == ';' {
			return i > 0
		}
		if !valid(value) {
			return false
		}
	}
	return false
}

func isDecimalDigit(value byte) bool {
	return value >= '0' && value <= '9'
}

func isHexDigit(value byte) bool {
	return isDecimalDigit(value) || value >= 'a' && value <= 'f' || value >= 'A' && value <= 'F'
}
