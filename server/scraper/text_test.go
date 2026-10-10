package scraper

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestFetchKeepsRecipeAfterLargeScriptsAndStyles(t *testing.T) {
	const structuredData = `{"@context":"https://schema.org","@type":"Article","headline":"Barbacoa Tacos"}`
	const recipeHTML = `<div class="recipe" itemscope itemtype="https://schema.org/Recipe">
		<h4 itemprop="name">Barbacoa Tacos</h4>
		<ul><li itemprop="recipeIngredient">1 tablespoon oil</li>
		<li itemprop="recipeIngredient">2 pounds chuck roast</li>
		<li><a href="https://example.com/sauce">Chipotle Sauce</a></li></ul>
		<ol itemprop="recipeInstructions"><li>Heat the oil and brown the beef.</li></ol>
	</div>`

	for _, tag := range []string{"script", "style", "SCRIPT", "STYLE"} {
		t.Run(tag, func(t *testing.T) {
			// Non-recipe content exceeds the scraper's entire text budget before
			// the recipe, as on the Closet Cooking page that triggered this bug.
			pageHTML := fmt.Sprintf(`<html><head><%s data-example="true">
%s
</%s><script type="application/ld+json">%s</script></head><body>%s</body></html>`,
				tag, strings.Repeat("non-recipe-content ", 5000), tag, structuredData, recipeHTML)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/robots.txt" {
					fmt.Fprint(w, "User-agent: *\nAllow: /\n")
					return
				}
				fmt.Fprint(w, pageHTML)
			}))
			defer server.Close()

			result, err := New(time.Second).Fetch(context.Background(), server.URL+"/barbacoa-tacos/")
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{
				"Barbacoa Tacos",
				"1 tablespoon oil",
				"2 pounds chuck roast",
				"Heat the oil and brown the beef.",
				"Chipotle Sauce [https://example.com/sauce]",
			} {
				if !strings.Contains(result.Text, want) {
					t.Errorf("page text is missing %q", want)
				}
			}
			if strings.Contains(result.Text, "non-recipe-content") || strings.Contains(result.Text, structuredData) {
				t.Error("page text includes script or style content")
			}
			if !reflect.DeepEqual(result.JSONLD, []string{structuredData}) {
				t.Errorf("JSONLD = %#v, want original structured data", result.JSONLD)
			}
			if result.HTML != pageHTML {
				t.Error("original HTML was changed")
			}
		})
	}
}
