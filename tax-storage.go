package main

import (
	"fmt"

	"github.com/dtnp/go/grafana-api/pkg/dashboard"
	"github.com/dtnp/go/grafana-api/pkg/simpletaxonomy"
)

// StoreTaxonomy walks through the simplified taxonomy tags and stores them in the database.
func StoreTaxonomy(db *dashboard.SQLiteRepository, tax simpletaxonomy.SimplifiedTaxonomy) error {
	var newTags []dashboard.Tag

    // Level 1 Tags
	for _, t1 := range tax.Taxonomies {
		newTags = append(newTags, dashboard.Tag{
			Name:       t1.Name,
			Slug:       t1.Slug,
			Definition: t1.Definition,
			Level:      1,
		})

        // Level 2 Tags
		if len(t1.Children) > 0 {
			for _, t2 := range t1.Children {
				newTags = append(newTags, dashboard.Tag{
					Name:       t2.Name,
					Slug:       t2.Slug,
					Definition: t2.Definition,
					Level:      2,
				})
			}
		}
	}

    // Save the tags!
	err := db.InsetTagBulk(newTags)
	if err != nil {
		return fmt.Errorf("StoreTaxonomy: %w", err)
	}

	return nil
}

// StoreDashboards walks through our dashboards and saves them in the database.
func StoreDashboards(db *dashboard.SQLiteRepository, dashboards []dashboardRaw) error {
    var newDashboards []dashboard.Dashboard

    for _, d := range dashboards {
        newDashboards = append(newDashboards, dashboard.Dashboard{
            GrafanaID: int64(d.ID),
            UID: d.UID,
            Title: d.Title,
            Url: d.Url,         
            Slug: d.Slug,
            Description: d.Description,
        })
    }

    err := db.InsertDashboardBulk(newDashboards)
    if err != nil {
        return fmt.Errorf("StoreDashboards: %w", err)
    }

    return nil
}
