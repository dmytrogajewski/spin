package suggest

import "strings"

// Filter ranks items whose Label or Insert contains query (prefix first).
func Filter(items []Item, query string, limit int) []Item {
	if limit <= 0 {
		return nil
	}

	needle := strings.ToLower(query)
	prefixed := make([]Item, 0, limit)
	rest := make([]Item, 0, limit)

	for _, item := range items {
		prefixed, rest = rankItem(item, needle, limit, prefixed, rest)
	}

	return mergeLimited(prefixed, rest, limit)
}

func rankItem(item Item, needle string, limit int, prefixed, rest []Item) (nextPrefixed, nextRest []Item) {
	if needle == "" {
		return appendLimited(prefixed, item, limit), rest
	}

	insert := strings.ToLower(item.Insert)
	bare := strings.ToLower(strings.TrimLeft(item.Insert, "/@"))

	if strings.HasPrefix(insert, needle) || strings.HasPrefix(bare, needle) {
		return appendLimited(prefixed, item, limit), rest
	}

	hay := strings.ToLower(item.Insert + " " + item.Label)
	if strings.Contains(hay, needle) {
		return prefixed, appendLimited(rest, item, limit)
	}

	return prefixed, rest
}

func appendLimited(dst []Item, item Item, limit int) []Item {
	if len(dst) >= limit {
		return dst
	}

	return append(dst, item)
}

func mergeLimited(first, second []Item, limit int) []Item {
	out := make([]Item, 0, limit)
	out = append(out, first...)

	for _, item := range second {
		if len(out) >= limit {
			break
		}

		out = append(out, item)
	}

	return out
}
