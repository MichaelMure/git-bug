package query

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MichaelMure/git-bug/bug"
)

func TestParse(t *testing.T) {
	var tests = []struct {
		input  string
		output *Query
	}{
		// KV
		{"status:", nil},
		{":value", nil},

		{"status:open", &Query{
			Filters: Filters{Status: []bug.Status{bug.OpenStatus}},
		}},
		{"status:closed", &Query{
			Filters: Filters{Status: []bug.Status{bug.ClosedStatus}},
		}},
		{"status:unknown", nil},

		{"author:rene", &Query{
			Filters: Filters{Author: []string{"rene"}},
		}},
		{`author:"René Descartes"`, &Query{
			Filters: Filters{Author: []string{"René Descartes"}},
		}},

		{"actor:bernhard", &Query{
			Filters: Filters{Actor: []string{"bernhard"}},
		}},
		{"participant:leonhard", &Query{
			Filters: Filters{Participant: []string{"leonhard"}},
		}},

		{"label:hello", &Query{
			Filters: Filters{Label: []string{"hello"}},
		}},
		{`label:"Good first issue"`, &Query{
			Filters: Filters{Label: []string{"Good first issue"}},
		}},

		{"title:titleOne", &Query{
			Filters: Filters{Title: []string{"titleOne"}},
		}},
		{`title:"Bug titleTwo"`, &Query{
			Filters: Filters{Title: []string{"Bug titleTwo"}},
		}},

		{"no:label", &Query{
			Filters: Filters{NoLabel: true},
		}},

		{"sort:edit", &Query{
			OrderBy: OrderByEdit,
		}},
		{"sort:unknown", nil},

		{"label:\"foo:bar\"", &Query{
			Filters: Filters{Label: []string{"foo:bar"}},
		}},

		// KVV
		{`metadata:key:"https://www.example.com/"`, &Query{
			Filters: Filters{Metadata: []StringPair{{"key", "https://www.example.com/"}}},
		}},

		// Search
		{"search", &Query{
			Search: []string{"search"},
		}},
		{"search \"more terms\"", &Query{
			Search: []string{"search", "more terms"},
		}},

		// Complex
		{`status:open author:"René Descartes" search participant:leonhard label:hello label:"Good first issue" sort:edit-desc "more terms"`,
			&Query{
				Search: []string{"search", "more terms"},
				Filters: Filters{
					Status:      []bug.Status{bug.OpenStatus},
					Author:      []string{"René Descartes"},
					Participant: []string{"leonhard"},
					Label:       []string{"hello", "Good first issue"},
				},
				OrderBy:        OrderByEdit,
				OrderDirection: OrderDescending,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			query, err := Parse(tc.input)
			if tc.output == nil {
				require.Error(t, err)
				require.Nil(t, query)
			} else {
				require.NoError(t, err)
				if tc.output.OrderBy != 0 {
					require.Equal(t, tc.output.OrderBy, query.OrderBy)
				}
				if tc.output.OrderDirection != 0 {
					require.Equal(t, tc.output.OrderDirection, query.OrderDirection)
				}
				require.Equal(t, tc.output.Filters, query.Filters)
			}
		})
	}
}
