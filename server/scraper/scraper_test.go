package scraper

import (
	"net/http"
	"reflect"
	"testing"

	"recipe-extractor/server/extractor"
)

func TestExtractJSONLD(t *testing.T) {
	tests := []struct {
		name string
		html string
		want []string
	}{
		{
			name: "extracts and trims non empty scripts",
			html: `<html>
				<script type="application/ld+json"> {"@type":"Recipe"} </script>
				<script type='application/ld+json'>
				{"@type":"BreadcrumbList"}
				</script>
			</html>`,
			want: []string{`{"@type":"Recipe"}`, `{"@type":"BreadcrumbList"}`},
		},
		{
			name: "ignores empty scripts",
			html: `<script type="application/ld+json">   </script>`,
			want: []string{},
		},
		{
			name: "no scripts",
			html: `<main>No structured data</main>`,
			want: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractJSONLD(tc.html)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("extractJSONLD() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestExtractLinks(t *testing.T) {
	tests := []struct {
		name string
		html string
		want []string
	}{
		{
			name: "cleans text and deduplicates by href",
			html: `<a href="https://example.com/sauce"><span> Tomato </span> Sauce</a>
				<a href="https://example.com/sauce">Duplicate Sauce</a>
				<a href="/bread">Garlic<br>Bread</a>`,
			want: []string{
				"Tomato Sauce [https://example.com/sauce]",
				"Garlic Bread [/bread]",
			},
		},
		{
			name: "ignores short text and fragments",
			html: `<a href="#jump">Jump Link</a>
				<a href="/ok">OK</a>
				<a href="/valid">Valid Recipe</a>`,
			want: []string{"Valid Recipe [/valid]"},
		},
		{
			name: "no links",
			html: `<p>No anchors</p>`,
			want: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractLinks(tc.html)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("extractLinks() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestExtractIngredientGroups(t *testing.T) {
	tests := []struct {
		name string
		html string
		want []extractor.IngredientGroup
	}{
		{
			name: "extracts grouped wprm ingredients",
			html: `<div class="wprm-recipe-ingredient-group">
				<h4 class="wprm-recipe-ingredient-group-name">Cake &amp; Batter</h4>
				<ul>
					<li class="wprm-recipe-ingredient"><span>1 cup</span> flour</li>
					<li class="wprm-recipe-ingredient">2 eggs</li>
				</ul>
			</div>
			<div class="wprm-recipe-ingredient-group">
				<h4 class="wprm-recipe-ingredient-group-name">Glaze</h4>
				<li class="wprm-recipe-ingredient">1 cup sugar</li>
			</div>`,
			want: []extractor.IngredientGroup{
				{Group: "Cake & Batter", Items: []string{"1 cup flour", "2 eggs"}},
				{Group: "Glaze", Items: []string{"1 cup sugar"}},
			},
		},
		{
			name: "skips groups without ingredients",
			html: `<div class="wprm-recipe-ingredient-group">
				<h4 class="wprm-recipe-ingredient-group-name">Empty</h4>
			</div>`,
		},
		{
			name: "returns nil when no groups exist",
			html: `<main>No ingredients</main>`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractIngredientGroups(tc.html)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("extractIngredientGroups() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestIsBlockedAccessResponse(t *testing.T) {
	tests := []struct {
		name   string
		status int
		header string
		body   string
		want   bool
	}{
		{name: "cloudflare challenge header", status: http.StatusOK, header: "challenge", want: true},
		{name: "forbidden javascript challenge", status: http.StatusForbidden, body: "Enable JavaScript and cookies to continue", want: true},
		{name: "too many requests captcha", status: http.StatusTooManyRequests, body: "Captcha required", want: true},
		{name: "forbidden unrelated", status: http.StatusForbidden, body: "No", want: false},
		{name: "ok with challenge text", status: http.StatusOK, body: "captcha", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := &http.Response{StatusCode: tc.status, Header: make(http.Header)}
			if tc.header != "" {
				res.Header.Set("cf-mitigated", tc.header)
			}

			if got := isBlockedAccessResponse(res, tc.body); got != tc.want {
				t.Fatalf("isBlockedAccessResponse() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestArchiveHelpers(t *testing.T) {
	t.Run("SupportsArchivedFallback", func(t *testing.T) {
		tests := []struct {
			name string
			msg  string
			want bool
		}{
			{name: "blocked access", msg: blockedAccessMessage + " (HTTP 403)", want: true},
			{name: "unrelated", msg: "unexpected status code: 500", want: false},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				if got := SupportsArchivedFallback(tc.msg); got != tc.want {
					t.Fatalf("SupportsArchivedFallback() = %v, want %v", got, tc.want)
				}
			})
		}
	})

	t.Run("IsArchivedURL", func(t *testing.T) {
		tests := []struct {
			name string
			url  string
			want bool
		}{
			{name: "archive url", url: "https://web.archive.org/web/20200101/https://example.com", want: true},
			{name: "case insensitive host", url: "https://WEB.ARCHIVE.ORG/web/20200101/https://example.com", want: true},
			{name: "normal url", url: "https://example.com/recipe", want: false},
			{name: "invalid url", url: "://bad", want: false},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				if got := IsArchivedURL(tc.url); got != tc.want {
					t.Fatalf("IsArchivedURL() = %v, want %v", got, tc.want)
				}
			})
		}
	})
}
