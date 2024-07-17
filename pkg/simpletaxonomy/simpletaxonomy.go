package simpletaxonomy

import (
	"encoding/json"
	"os"
	"strings"
)

type SimplifiedTaxonomy struct {
    Taxonomies []TaxL1 `json:"simplified-taxonomy"`
}

type TaxL1 struct {
	Name       string  `json:"l1-taxonomy"`
	Slug       string  `json:"slug"`
	Definition string  `json:"definition"`
	Children   []TaxL2 `json:"children"`
}

type TaxL2 struct {
	Name       string `json:"l2-taxonomy"`
	Slug       string `json:"slug"`
	Definition string `json:"definition"`
	Squad      []string `json:"squad"`
}

// Parse and return the Simplified Taxonomy structure from a .json file
func ParseFile(filename string) (SimplifiedTaxonomy, error) {
    var taxonomies SimplifiedTaxonomy

    content, err := os.ReadFile(filename)
    if err != nil {
        return taxonomies, err
    }

    err = json.Unmarshal(content, &taxonomies)
    if err != nil {
        return taxonomies, err
    }

    return taxonomies, nil
}

// Lookup L1 Taxonomy name from slug
func (st *SimplifiedTaxonomy) GetL1NameFromSlug (slug string) string {
    for _, tax := range st.Taxonomies {
        if strings.Compare(strings.TrimSpace(slug), tax.Slug) == 0 {
            return tax.Name
        }
    }

    // We didn't find anything pass back the slug
    return slug
}

// Lookup L2 Taxonomy name from slug
func (st *SimplifiedTaxonomy) GetL2NameFromSlug (slug string) string {
    for _, tax1 := range st.Taxonomies {
        for _, tax2 := range tax1.Children {
            if strings.Compare(strings.TrimSpace(slug), tax2.Slug) == 0 {
                return tax2.Name
            }
        }
    }

    // We didn't find anything pass back the slug
    return slug
}

