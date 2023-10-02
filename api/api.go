package api

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/manojnakp/scount/api/internal"
)

// ErrPaginator defines parsing error for Paginator.
var ErrPaginator = errors.New("invalid pagination parameter")

// PageSize is the default number of items limit to a page.
const PageSize = 10

// Paginator represents the pagination query parameters.
type Paginator struct {
	Page int
	Size int
}

// DefaultPaginator is a paginator with default values.
var DefaultPaginator, _ = ParsePaginator(nil)

// ParsePaginator parses query parameters into Paginator,
// emitting errors in case of failure.
func ParsePaginator(query url.Values) (Paginator, error) {
	var zero Paginator
	page, err := internal.ParseInt(query.Get("page"), 0)
	if err != nil {
		log.Println("failed to parse 'page' parameter", err)
		return zero, fmt.Errorf("%w: 'page' not a number", ErrPaginator)
	}
	size, err := internal.ParseInt(query.Get("size"), PageSize)
	if err != nil {
		log.Println("failed to parse 'size' parameter", err)
		return zero, fmt.Errorf("%w: 'size' not a number", ErrPaginator)
	}
	return Paginator{
		Page: page,
		Size: size,
	}, nil
}

// WebLink defines a link as per [RFC8288].
//
// [RFC8288]: https://www.rfc-editor.org/rfc/rfc8288
type WebLink struct {
	Target     *url.URL
	Attributes map[string]string
}

// NewWebLink constructs a simple web link with target URI obtained from
// the given path and query, and containing given rel attribute.
func NewWebLink(path string, query url.Values, rel string) WebLink {
	return WebLink{
		Target: &url.URL{
			Path:     path,
			RawQuery: query.Encode(),
		},
		Attributes: map[string]string{
			"rel": rel,
		},
	}
}

// Encode gives a string representation of the web link as per
// Link serialization in HTTP headers [[RFC8288:Section-3]].
//
// [Section3]: https://www.rfc-editor.org/rfc/rfc8288#section-3
func (linker WebLink) Encode() string {
	uri := linker.Target.String()
	if uri == "" {
		return ""
	}
	items := make([]string, 0, len(linker.Attributes)+1)
	items = append(items, fmt.Sprintf("<%s>", uri))
	for param, value := range linker.Attributes {
		items = append(items, fmt.Sprintf("%s=%q", param, value))
	}
	return strings.Join(items, ";")
}

// LinkHeader sets HTTP `Link` header using a list of WebLink.
func LinkHeader(w http.ResponseWriter, links []WebLink) {
	items := make([]string, 0, len(links))
	for _, link := range links {
		s := link.Encode()
		if s == "" {
			continue
		}
		items = append(items, s)
	}
	if len(items) == 0 {
		return
	}
	w.Header().Set("Link", strings.Join(items, ","))
}

// PagingLinks constructs a list of WebLink from paging parameters parsed from
// the query params, or fallback to DefaultPaginator in case of error.
func PagingLinks(base string, params url.Values, total int) []WebLink {
	if total <= 0 {
		return nil
	}
	links := make([]WebLink, 0, 4)
	paging, err := ParsePaginator(params)
	if err != nil {
		paging = DefaultPaginator
	}
	var (
		size  = paging.Size
		page  = paging.Page
		first = 0
		last  = total / size
	)
	params.Set("page", strconv.Itoa(first))
	links = append(links, NewWebLink(base, params, "first"))
	params.Set("page", strconv.Itoa(last))
	links = append(links, NewWebLink(base, params, "last"))
	if page > 0 {
		params.Set("page", strconv.Itoa(page-1))
		links = append(links, NewWebLink(base, params, "prev"))
	}
	if page < last {
		params.Set("page", strconv.Itoa(page+1))
		links = append(links, NewWebLink(base, params, "next"))
	}
	return links
}
