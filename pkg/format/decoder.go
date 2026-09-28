package format

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"redis-cli-tool/pkg/client"
)

// MaxPreviewBytes defines the 256KB preview ceiling to prevent CLI memory bloat
const MaxPreviewBytes = 256 * 1024

// MaxCollectionItems defines the bounded iterator ceiling (100 items)
const MaxCollectionItems = 100

type DecodedValueType string

const (
	TypeJSON   DecodedValueType = "JSON"
	TypeString DecodedValueType = "STRING"
	TypeBinary DecodedValueType = "BINARY"
	TypeHash   DecodedValueType = "HASH"
	TypeList   DecodedValueType = "LIST"
	TypeSet    DecodedValueType = "SET"
	TypeZSet   DecodedValueType = "ZSET"
	TypeStream DecodedValueType = "STREAM"
)

type DecodedResult struct {
	ValueType DecodedValueType `json:"value_type"`
	RawBytes  int64            `json:"raw_bytes"`
	Truncated bool             `json:"truncated"`
	ItemCount int64            `json:"item_count"`
	Formatted string           `json:"formatted"`
	Data      interface{}      `json:"data,omitempty"`
}

var (
	JSONKeyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("39")) // Cyan
	JSONStringStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42")) // Green
	JSONNumberStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // Yellow
	JSONBoolStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("205")) // Pink/Magenta
)

// IsBinaryData detects non-printable or null byte content
func IsBinaryData(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	if bytes.IndexByte(data, 0) != -1 {
		return true
	}
	if !utf8.Valid(data) {
		return true
	}

	nonPrintable := 0
	for _, r := range string(data) {
		if !unicode.IsPrint(r) && !unicode.IsSpace(r) {
			nonPrintable++
		}
	}
	return float64(nonPrintable)/float64(len(data)) > 0.05
}

// FormatPrettyJSON formats JSON bytes with 2-space indentation
func FormatPrettyJSON(data []byte) (string, error) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// DecodeStringPayload decodes raw bytes into a DecodedResult (checking JSON, string, or binary)
func DecodeStringPayload(raw []byte, full bool) *DecodedResult {
	res := &DecodedResult{
		RawBytes: int64(len(raw)),
	}

	displayBytes := raw
	if len(raw) > MaxPreviewBytes && !full {
		res.Truncated = true
		displayBytes = raw[:MaxPreviewBytes]
	}

	if json.Valid(displayBytes) {
		res.ValueType = TypeJSON
		if pretty, err := FormatPrettyJSON(displayBytes); err == nil {
			res.Formatted = pretty
		} else {
			res.Formatted = string(displayBytes)
		}
	} else if IsBinaryData(displayBytes) {
		res.ValueType = TypeBinary
		res.Formatted = hex.Dump(displayBytes)
	} else {
		res.ValueType = TypeString
		res.Formatted = string(displayBytes)
	}

	if res.Truncated {
		res.Formatted += fmt.Sprintf("\n\n... [Truncated: showing 256KB of %d bytes. Use --full to display entire payload]", res.RawBytes)
	}

	return res
}

// DecodeRedisKey inspects any Redis key with bounded collection protection
func DecodeRedisKey(ctx context.Context, cl *client.Client, key, keyType string, full bool) (*DecodedResult, error) {
	if cl == nil {
		return nil, fmt.Errorf("client is nil")
	}

	lowerType := strings.ToLower(keyType)
	switch lowerType {
	case "string":
		raw, err := cl.Get(ctx, key).Bytes()
		if err != nil {
			return nil, err
		}
		return DecodeStringPayload(raw, full), nil

	case "hash":
		totalCount, _ := cl.HLen(ctx, key).Result()
		fields, _, err := cl.HScan(ctx, key, 0, "*", MaxCollectionItems).Result()
		if err != nil {
			return nil, err
		}

		// HScan returns pairs: field, value, field, value...
		hashMap := make(map[string]string)
		for i := 0; i < len(fields)-1; i += 2 {
			hashMap[fields[i]] = fields[i+1]
		}

		headers := []string{"FIELD", "VALUE"}
		rows := make([][]string, 0, len(hashMap))
		for k, v := range hashMap {
			if len(v) > 80 {
				v = v[:77] + "..."
			}
			rows = append(rows, []string{k, v})
		}

		formatted := RenderTable(headers, rows)
		if totalCount > int64(len(rows)) {
			formatted += fmt.Sprintf("\n(Showing %d of %d total hash fields)", len(rows), totalCount)
		}

		return &DecodedResult{
			ValueType: TypeHash,
			ItemCount: totalCount,
			Formatted: formatted,
			Data:      hashMap,
		}, nil

	case "list":
		totalCount, _ := cl.LLen(ctx, key).Result()
		items, err := cl.LRange(ctx, key, 0, MaxCollectionItems-1).Result()
		if err != nil {
			return nil, err
		}

		headers := []string{"INDEX", "VALUE"}
		rows := make([][]string, len(items))
		for i, item := range items {
			if len(item) > 80 {
				item = item[:77] + "..."
			}
			rows[i] = []string{fmt.Sprintf("[%d]", i), item}
		}

		formatted := RenderTable(headers, rows)
		if totalCount > int64(len(items)) {
			formatted += fmt.Sprintf("\n(Showing %d of %d total list elements)", len(items), totalCount)
		}

		return &DecodedResult{
			ValueType: TypeList,
			ItemCount: totalCount,
			Formatted: formatted,
			Data:      items,
		}, nil

	case "set":
		totalCount, _ := cl.SCard(ctx, key).Result()
		members, _, err := cl.SScan(ctx, key, 0, "*", MaxCollectionItems).Result()
		if err != nil {
			return nil, err
		}

		headers := []string{"MEMBER"}
		rows := make([][]string, len(members))
		for i, m := range members {
			if len(m) > 80 {
				m = m[:77] + "..."
			}
			rows[i] = []string{m}
		}

		formatted := RenderTable(headers, rows)
		if totalCount > int64(len(members)) {
			formatted += fmt.Sprintf("\n(Showing %d of %d total set members)", len(members), totalCount)
		}

		return &DecodedResult{
			ValueType: TypeSet,
			ItemCount: totalCount,
			Formatted: formatted,
			Data:      members,
		}, nil

	case "zset":
		totalCount, _ := cl.ZCard(ctx, key).Result()
		zItems, err := cl.ZRangeWithScores(ctx, key, 0, MaxCollectionItems-1).Result()
		if err != nil {
			return nil, err
		}

		headers := []string{"SCORE", "MEMBER"}
		rows := make([][]string, len(zItems))
		for i, z := range zItems {
			memberStr := fmt.Sprintf("%v", z.Member)
			if len(memberStr) > 80 {
				memberStr = memberStr[:77] + "..."
			}
			rows[i] = []string{fmt.Sprintf("%.2f", z.Score), memberStr}
		}

		formatted := RenderTable(headers, rows)
		if totalCount > int64(len(zItems)) {
			formatted += fmt.Sprintf("\n(Showing %d of %d total sorted set members)", len(zItems), totalCount)
		}

		return &DecodedResult{
			ValueType: TypeZSet,
			ItemCount: totalCount,
			Formatted: formatted,
			Data:      zItems,
		}, nil

	default:
		// Fallback to GET
		raw, err := cl.Get(ctx, key).Bytes()
		if err != nil {
			return nil, err
		}
		return DecodeStringPayload(raw, full), nil
	}
}
