package worker

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/jsness/recipe-extractor/server/extractor"
	"github.com/jsness/recipe-extractor/server/scraper"
	"github.com/jsness/recipe-extractor/server/wayback"
)

const initialSnapshot = "https://web.archive.org/web/20260101120000/https://example.com/recipe"
const olderSnapshot = "https://web.archive.org/web/20250101120000/https://example.com/recipe"
const oldestSnapshot = "https://web.archive.org/web/20240101120000/https://example.com/recipe"

type recoveryFetcher struct {
	calls []string
	fetch func(string) (scraper.Result, error)
}

func (f *recoveryFetcher) Fetch(ctx context.Context, url string) (scraper.Result, error) {
	f.calls = append(f.calls, url)
	if err := ctx.Err(); err != nil {
		return scraper.Result{}, err
	}
	return f.fetch(url)
}

type recoveryArchive struct {
	calls     []string
	snapshots []wayback.Snapshot
	err       error
}

func (a *recoveryArchive) ListSnapshots(ctx context.Context, url string) ([]wayback.Snapshot, error) {
	a.calls = append(a.calls, url)
	return a.snapshots, a.err
}

func TestArchiveRecoveryFindsRecipeAndReturnsActualSource(t *testing.T) {
	fetcher := &recoveryFetcher{fetch: func(url string) (scraper.Result, error) {
		result := scraper.Result{SourceURL: url}
		if url == olderSnapshot {
			result.JSONLD = []string{`[{"@type":"Recipe","name":"Soup","recipeIngredient":["water"],"recipeInstructions":["Boil."]}]`}
		}
		return result, nil
	}}
	archive := &recoveryArchive{snapshots: []wayback.Snapshot{
		{URL: initialSnapshot},
		{URL: "http://web.archive.org/web/20260101120000id_/https://example.com/recipe"},
		{URL: "https://web.archive.org/web/20250101120000/https://example.com/other-recipe"},
		{URL: olderSnapshot, Timestamp: "20250101120000"},
	}}
	w := &Worker{scraper: fetcher, archive: archive, extractor: extractor.NewJSONLDExtractor(nil, testLogger()), logger: testLogger()}
	recipe, source, err := w.extractRecipe(context.Background(), initialSnapshot)
	if err != nil || recipe.Title != "Soup" || source != olderSnapshot {
		t.Fatalf("extractRecipe() = %#v, %q, %v", recipe, source, err)
	}
	if !reflect.DeepEqual(fetcher.calls, []string{initialSnapshot, olderSnapshot}) || !reflect.DeepEqual(archive.calls, []string{"https://example.com/recipe"}) {
		t.Fatalf("fetch calls=%v, archive calls=%v", fetcher.calls, archive.calls)
	}
}

type failingExtractor struct{ err error }

func (f failingExtractor) NormalizeRecipe(context.Context, extractor.Input) (extractor.Recipe, error) {
	return extractor.Recipe{}, f.err
}

func TestArchiveRecoveryDoesNotRepeatProviderFailures(t *testing.T) {
	providerErr := errors.New("provider request failed: status=401")
	fetcher := &recoveryFetcher{fetch: func(url string) (scraper.Result, error) { return scraper.Result{SourceURL: url}, nil }}
	archive := &recoveryArchive{snapshots: []wayback.Snapshot{{URL: olderSnapshot}}}
	w := &Worker{scraper: fetcher, archive: archive, extractor: failingExtractor{providerErr}, logger: testLogger()}
	_, _, err := w.extractRecipe(context.Background(), initialSnapshot)
	if !errors.Is(err, providerErr) || len(fetcher.calls) != 1 || len(archive.calls) != 0 {
		t.Fatalf("error=%v, fetch calls=%v, archive calls=%v", err, fetcher.calls, archive.calls)
	}
}

func TestArchiveRecoveryIsBoundedAndStopsOnRobotsDenial(t *testing.T) {
	for _, robotsDenied := range []bool{false, true} {
		fetcher := &recoveryFetcher{fetch: func(url string) (scraper.Result, error) {
			if robotsDenied {
				return scraper.Result{}, scraper.ErrRobotsDenied
			}
			return scraper.Result{}, &scraper.FetchError{Kind: scraper.FetchErrorKindUnexpectedStatus, StatusCode: 404}
		}}
		archive := &recoveryArchive{snapshots: []wayback.Snapshot{
			{URL: olderSnapshot}, {URL: olderSnapshot}, {URL: oldestSnapshot},
			{URL: "https://web.archive.org/web/20230101120000/https://example.com/recipe"},
		}}
		w := &Worker{scraper: fetcher, archive: archive, logger: testLogger()}
		_, _, err := w.extractRecipe(context.Background(), initialSnapshot)
		wantCalls := 3
		if robotsDenied {
			wantCalls = 1
			if len(archive.calls) != 0 || !errors.Is(err, scraper.ErrRobotsDenied) {
				t.Fatalf("robots denial error=%v, archive calls=%v", err, archive.calls)
			}
		}
		if err == nil || len(fetcher.calls) != wantCalls {
			t.Fatalf("error=%v, fetch calls=%v", err, fetcher.calls)
		}
	}
}

func TestDirectSourceFailureDoesNotAutomaticallyUseArchive(t *testing.T) {
	fetcher := &recoveryFetcher{fetch: func(string) (scraper.Result, error) {
		return scraper.Result{}, &scraper.FetchError{Kind: scraper.FetchErrorKindBlockedAccess, StatusCode: 403}
	}}
	archive := &recoveryArchive{}
	w := &Worker{scraper: fetcher, archive: archive, logger: testLogger()}
	_, _, err := w.extractRecipe(context.Background(), "https://example.com/recipe")
	if !scraper.IsBlockedAccessError(err) || len(archive.calls) != 0 || len(fetcher.calls) != 1 {
		t.Fatalf("error=%v, fetch calls=%v, archive calls=%v", err, fetcher.calls, archive.calls)
	}
}

func TestArchiveRecoveryPropagatesLookupAndCancellationErrors(t *testing.T) {
	lookupErr := errors.New("archive lookup unavailable")
	fetcher := &recoveryFetcher{fetch: func(string) (scraper.Result, error) {
		return scraper.Result{}, &scraper.FetchError{Kind: scraper.FetchErrorKindUnexpectedStatus, StatusCode: 404}
	}}
	archive := &recoveryArchive{err: lookupErr}
	w := &Worker{scraper: fetcher, archive: archive, logger: testLogger()}
	_, _, err := w.extractRecipe(context.Background(), initialSnapshot)
	if !errors.Is(err, lookupErr) {
		t.Fatalf("error=%v, want archive lookup failure", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	archive.calls = nil
	_, _, err = w.extractRecipe(ctx, initialSnapshot)
	if !errors.Is(err, context.Canceled) || len(archive.calls) != 0 {
		t.Fatalf("error=%v, archive calls=%v", err, archive.calls)
	}
}
