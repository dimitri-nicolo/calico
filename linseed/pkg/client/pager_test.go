package client_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client"
)

// setupTest runs common logic before each test, and also returns a function to perform teardown
// after each test.
func setupTest(t *testing.T) func() {
	cancel := logutils.RedirectLogrusToTestingT(t)
	return cancel
}

func TestPager(t *testing.T) {
	// We'll use a dummy ListFunc that returns two pages of results.

	type result struct {
		List  *v1.List[v1.L3Flow]
		Error error
	}

	getListFunc := func(testData []result) client.ListFunc[v1.L3Flow] {
		i := 0
		return func(context.Context, v1.Params) (*v1.List[v1.L3Flow], error) {
			// Use a local function to yield the next item from test data
			// on each call of listFunc.
			res := testData[i]
			t.Logf("PageFunc returning result %d: %+v, %s", i, res.List, res.Error)
			i++
			return res.List, res.Error
		}
	}

	t.Run("should handle a paged list with no errors", func(t *testing.T) {
		defer setupTest(t)()

		// Data to be returned by listFunc for the test.
		testData := []result{
			{
				List: &v1.List[v1.L3Flow]{
					AfterKey: map[string]interface{}{"foo": "bar"},
				},
				Error: nil,
			},
			{
				List: &v1.List[v1.L3Flow]{
					AfterKey: nil, // Indicates this is the last page.
				},
				Error: nil,
			},
		}

		// Perform some paged lists.
		pager := client.NewListPager[v1.L3Flow](&v1.L3FlowParams{})
		listFunc := getListFunc(testData)
		getPage := pager.PageFunc(listFunc)

		var page *v1.List[v1.L3Flow]
		var err error
		var more bool
		for _, expected := range testData {
			page, more, err = getPage()
			require.NoError(t, err)
			require.NotNil(t, page)
			require.Equal(t, expected.List, page)
		}
		require.False(t, more)
	})

	t.Run("should handle a paged list with an error", func(t *testing.T) {
		defer setupTest(t)()

		// Data to be returned by listFunc for the test.
		testData := []result{
			{
				List: &v1.List[v1.L3Flow]{
					AfterKey: map[string]interface{}{"foo": "bar"},
				},
				Error: fmt.Errorf("error in first page call"),
			},
		}

		// Perform some paged lists.
		pager := client.NewListPager[v1.L3Flow](&v1.L3FlowParams{})
		listFunc := getListFunc(testData)
		getPage := pager.PageFunc(listFunc)

		// It should only return a single page due to the error.
		page, more, err := getPage()
		require.Error(t, err)
		require.Nil(t, page)
		require.False(t, more)

		// If we call it again, we should get another error. This time, because
		// the pager has marked itself as complete due to the first error.
		page, more, err = getPage()
		require.Error(t, err)
		require.Nil(t, page)
		require.False(t, more)
	})

	t.Run("should support streaming pages of data", func(t *testing.T) {
		defer setupTest(t)()

		// Data to be returned by listFunc for the test.
		testData := []result{
			{
				List: &v1.List[v1.L3Flow]{
					AfterKey: map[string]interface{}{"foo": "bar"},
				},
				Error: nil,
			},
			{
				List: &v1.List[v1.L3Flow]{
					AfterKey: map[string]interface{}{"whizz": "pop"},
				},
				Error: nil,
			},
			{
				List: &v1.List[v1.L3Flow]{
					AfterKey: map[string]interface{}{"ham": "salad"},
				},
				Error: nil,
			},
			{
				List: &v1.List[v1.L3Flow]{
					AfterKey: nil, // Indicates this is the last page.
				},
				Error: nil,
			},
		}

		// Perform some paged lists.
		pager := client.NewListPager[v1.L3Flow](&v1.L3FlowParams{})
		listFunc := getListFunc(testData)

		var page v1.List[v1.L3Flow]
		var err error

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Create the streamer.
		results, errors := pager.Stream(ctx, listFunc)
		for _, expected := range testData {
			select {
			case err = <-errors:
				require.NoError(t, err)
			case page = <-results:
				require.NotNil(t, page)
				require.Equal(t, *expected.List, page)
			}
		}

		// Assert that the channels have been closed, since we've read all the data.
		require.Empty(t, results)
		require.Empty(t, errors)
	})

	t.Run("should support streaming pages of data with an error", func(t *testing.T) {
		defer setupTest(t)()

		// Data to be returned by listFunc for the test.
		testData := []result{
			{
				List: &v1.List[v1.L3Flow]{
					AfterKey: map[string]interface{}{"foo": "bar"},
				},
				Error: nil,
			},
			{
				List: &v1.List[v1.L3Flow]{
					AfterKey: map[string]interface{}{"whizz": "pop"},
				},
				Error: nil,
			},
			{
				List: &v1.List[v1.L3Flow]{
					AfterKey: map[string]interface{}{"ham": "salad"},
				},
				Error: fmt.Errorf("ham salad is not desirable"),
			},
			{
				List: &v1.List[v1.L3Flow]{
					Items:    []v1.L3Flow{{}},
					AfterKey: nil, // Indicates this is the last page.
				},
				Error: nil,
			},
		}

		// Perform some paged lists.
		pager := client.NewListPager[v1.L3Flow](&v1.L3FlowParams{})
		listFunc := getListFunc(testData)

		var page v1.List[v1.L3Flow]
		var err error

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Create the streamer.
		results, errors := pager.Stream(ctx, listFunc)

		allPages := []v1.List[v1.L3Flow]{}

		// Iterate through test data up until the one that generates the error.
		// Index 2 is the page that generates the error.
		for range testData[0:3] {
			select {
			case err = <-errors:
			case page = <-results:
				require.NotNil(t, page)
				allPages = append(allPages, page)
			}
		}

		// Check the outputs.
		require.Error(t, err)
		for i, p := range allPages {
			require.Equal(t, *testData[i].List, p)
		}

		// Assert that the channels have been closed, since we've read all the data.
		require.Empty(t, errors)
		require.Empty(t, results)
	})
}
