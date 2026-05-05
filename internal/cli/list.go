package cli

import (
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/ariefsn/dust/internal/cleaner"
	"github.com/ariefsn/dust/internal/cleaners"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var categoriesOnly bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List every registered cleaner, grouped by category",
		Long: `List every cleaner dust knows about, grouped by category.

Use the IDs shown here as values for 'dust clean --item=...', and the
category names for 'dust clean --category=...'.`,
		Example: `  dust list
  dust list --categories            # only print category names
  dust clean --category=$(dust list --categories | head -1) --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := cleaner.NewRegistry()
			cleaners.RegisterAll(reg)
			byCat := reg.ByCategory()

			cats := make([]string, 0, len(byCat))
			for c := range byCat {
				cats = append(cats, c)
			}
			sort.Strings(cats)

			if categoriesOnly {
				for _, c := range cats {
					cmd.Println(strings.ToLower(c))
				}
				return nil
			}

			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "CATEGORY\tID\tNAME")
			for _, cat := range cats {
				items := byCat[cat]
				sort.Slice(items, func(i, j int) bool { return items[i].ID() < items[j].ID() })
				for i, c := range items {
					catCol := ""
					if i == 0 {
						catCol = strings.ToLower(cat)
					}
					fmt.Fprintf(tw, "%s\t%s\t%s\n", catCol, c.ID(), c.Name())
				}
			}
			tw.Flush()
			cmd.Println()
			cmd.Printf("%d cleaners across %d categories.\n", countCleaners(byCat), len(cats))
			cmd.Println("Tip: pass `--category=<name>` or `--item=<id>` to `dust clean`.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&categoriesOnly, "categories", false, "print just the category names (one per line)")
	return cmd
}

func countCleaners(byCat map[string][]cleaner.Cleaner) int {
	var n int
	for _, v := range byCat {
		n += len(v)
	}
	return n
}
