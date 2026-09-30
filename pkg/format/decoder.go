package format

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
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
	JSONKeyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))  // Cyan
	JSONStringStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))  // Green
	JSONNumberStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // Yellow
	JSONBoolStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("205")) // Pink/Magenta
)

// truncateRunes safely slices a string to maxRunes respecting UTF-8 boundaries
func truncateRunes(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes-3]) + "..."
	}
	return s
}

// DisarmANSI neutralizes ANSI escape characters to prevent terminal injection
func DisarmANSI(s string) string {
	return strings.ReplaceAll(s, "\x1b", "^[")
}

// ColorizeJSON highlights JSON syntax using lipgloss styles
func ColorizeJSON(s string) string {
	lines := strings.Split(s, "\n")
	var result strings.Builder
	for lineIdx, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		indent := line[:len(line)-len(trimmed)]
		if strings.Contains(trimmed, `": `) {
			parts := strings.SplitN(trimmed, `": `, 2)
			keyPart := parts[0] + `"`
			valPart := parts[1]
			coloredKey := JSONKeyStyle.Render(keyPart)

			var coloredVal string
			if strings.HasPrefix(valPart, `"`) {
				comma := ""
				if strings.HasSuffix(valPart, ",") {
					comma = ","
					valPart = valPart[:len(valPart)-1]
				}
				coloredVal = JSONStringStyle.Render(valPart) + comma
			} else if valPart == "true" || valPart == "false" || valPart == "true," || valPart == "false," || valPart == "null" || valPart == "null," {
				comma := ""
				if strings.HasSuffix(valPart, ",") {
					comma = ","
					valPart = valPart[:len(valPart)-1]
				}
				coloredVal = JSONBoolStyle.Render(valPart) + comma
			} else if len(valPart) > 0 && (valPart[0] >= '0' && valPart[0] <= '9' || valPart[0] == '-') {
				comma := ""
				if strings.HasSuffix(valPart, ",") {
					comma = ","
					valPart = valPart[:len(valPart)-1]
				}
				coloredVal = JSONNumberStyle.Render(valPart) + comma
			} else {
				coloredVal = valPart
			}
			result.WriteString(indent + coloredKey + ": " + coloredVal)
		} else {
			result.WriteString(line)
		}
		if lineIdx < len(lines)-1 {
			result.WriteString("\n")
		}
	}
	return result.String()
}

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
	runeCount := 0
	for _, r := range string(data) {
		runeCount++
		if !unicode.IsPrint(r) && !unicode.IsSpace(r) {
			nonPrintable++
		}
	}
	if runeCount == 0 {
		return false
	}
	return float64(nonPrintable)/float64(runeCount) > 0.05
}

// FormatPrettyJSON formats JSON bytes with 2-space indentation
func FormatPrettyJSON(data []byte) (string, error) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// DecodeStringPayload decodes raw bytes into a DecodedResult
func DecodeStringPayload(raw []byte, full bool) *DecodedResult {
	return DecodeStringPayloadWithTotal(raw, int64(len(raw)), full)
}

// DecodeStringPayloadWithTotal decodes bytes with true known total length
func DecodeStringPayloadWithTotal(raw []byte, totalBytes int64, full bool) *DecodedResult {
	res := &DecodedResult{
		RawBytes: totalBytes,
	}

	displayBytes := raw
	if (totalBytes > MaxPreviewBytes || len(raw) > MaxPreviewBytes) && !full {
		res.Truncated = true
		cut := len(raw)
		if cut > MaxPreviewBytes {
			cut = MaxPreviewBytes
		}
		for cut > 0 && !utf8.RuneStart(raw[cut]) {
			cut--
		}
		displayBytes = raw[:cut]
	}

	if json.Valid(displayBytes) {
		res.ValueType = TypeJSON
		if pretty, err := FormatPrettyJSON(displayBytes); err == nil {
			res.Formatted = ColorizeJSON(pretty)
		} else {
			res.Formatted = string(displayBytes)
		}
	} else if IsBinaryData(displayBytes) {
		res.ValueType = TypeBinary
		res.Formatted = hex.Dump(displayBytes)
	} else {
		res.ValueType = TypeString
		res.Formatted = DisarmANSI(string(displayBytes))
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
		if !full {
			strLen, err := cl.StrLen(ctx, key).Result()
			if err == nil && strLen > MaxPreviewBytes {
				raw, err := cl.GetRange(ctx, key, 0, MaxPreviewBytes-1).Bytes()
				if err != nil {
					return nil, err
				}
				return DecodeStringPayloadWithTotal(raw, strLen, false), nil
			}
		}
		raw, err := cl.Get(ctx, key).Bytes()
		if err != nil {
			return nil, err
		}
		return DecodeStringPayload(raw, full), nil

	case "hash":
		totalCount, _ := cl.HLen(ctx, key).Result()
		var cursor uint64
		hashMap := make(map[string]string)

		for {
			fields, nextCursor, err := cl.HScan(ctx, key, cursor, "*", MaxCollectionItems).Result()
			if err != nil {
				return nil, err
			}
			for i := 0; i < len(fields)-1; i += 2 {
				hashMap[fields[i]] = fields[i+1]
				if len(hashMap) >= MaxCollectionItems {
					break
				}
			}
			cursor = nextCursor
			if cursor == 0 || len(hashMap) >= MaxCollectionItems {
				break
			}
		}

		keys := make([]string, 0, len(hashMap))
		for k := range hashMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		headers := []string{"FIELD", "VALUE"}
		rows := make([][]string, 0, len(keys))
		for _, k := range keys {
			rows = append(rows, []string{k, truncateRunes(hashMap[k], 80)})
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
			rows[i] = []string{fmt.Sprintf("[%d]", i), truncateRunes(item, 80)}
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
		var cursor uint64
		var members []string

		for {
			batch, nextCursor, err := cl.SScan(ctx, key, cursor, "*", MaxCollectionItems).Result()
			if err != nil {
				return nil, err
			}
			members = append(members, batch...)
			cursor = nextCursor
			if cursor == 0 || len(members) >= MaxCollectionItems {
				break
			}
		}

		if len(members) > MaxCollectionItems {
			members = members[:MaxCollectionItems]
		}
		sort.Strings(members)

		headers := []string{"MEMBER"}
		rows := make([][]string, len(members))
		for i, m := range members {
			rows[i] = []string{truncateRunes(m, 80)}
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
			rows[i] = []string{fmt.Sprintf("%.2f", z.Score), truncateRunes(memberStr, 80)}
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

	case "stream":
		totalCount, _ := cl.XLen(ctx, key).Result()
		entries, err := cl.XRevRangeN(ctx, key, "+", "-", MaxCollectionItems).Result()
		if err != nil {
			return nil, err
		}

		headers := []string{"ID", "FIELDS"}
		rows := make([][]string, len(entries))
		for i, entry := range entries {
			var parts []string
			for fk, fv := range entry.Values {
				parts = append(parts, fmt.Sprintf("%s: %v", fk, fv))
			}
			rows[i] = []string{entry.ID, truncateRunes(strings.Join(parts, ", "), 80)}
		}

		formatted := RenderTable(headers, rows)
		if totalCount > int64(len(entries)) {
			formatted += fmt.Sprintf("\n(Showing %d of %d total stream entries)", len(entries), totalCount)
		}

		return &DecodedResult{
			ValueType: TypeStream,
			ItemCount: totalCount,
			Formatted: formatted,
			Data:      entries,
		}, nil

	default:
		return &DecodedResult{
			ValueType: DecodedValueType(strings.ToUpper(keyType)),
			Formatted: fmt.Sprintf("[Key type %q does not support value preview]", keyType),
		}, nil
	}
}
