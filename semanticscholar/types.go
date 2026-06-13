package semanticscholar

import (
	"strings"
)

// Paper is the record emitted for paper search results and single-paper fetches.
type Paper struct {
	Rank       int    `json:"rank"`
	ID         string `json:"id"`
	Title      string `json:"title"`
	Year       int    `json:"year"`
	Authors    string `json:"authors"`
	Venue      string `json:"venue"`
	Citations  int    `json:"citations"`
	OpenAccess bool   `json:"open_access"`
	URL        string `json:"url"`
}

// Author is the record emitted for author search results and profile fetches.
type Author struct {
	Rank      int    `json:"rank"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	HIndex    int    `json:"h_index"`
	Citations int    `json:"citations"`
	Papers    int    `json:"papers"`
	URL       string `json:"url"`
}

// ─── wire types from the Semantic Scholar Graph API ──────────────────────────

type apiPaper struct {
	PaperID                  string           `json:"paperId"`
	Title                    string           `json:"title"`
	Abstract                 string           `json:"abstract"`
	Year                     *int             `json:"year"`
	Authors                  []apiAuthorShort `json:"authors"`
	ExternalIDs              map[string]any   `json:"externalIds"`
	CitationCount            int              `json:"citationCount"`
	ReferenceCount           int              `json:"referenceCount"`
	InfluentialCitationCount int              `json:"influentialCitationCount"`
	IsOpenAccess             bool             `json:"isOpenAccess"`
	OpenAccessPdf            *struct {
		URL string `json:"url"`
	} `json:"openAccessPdf"`
	Venue           string `json:"venue"`
	PublicationDate string `json:"publicationDate"`
}

type apiAuthorShort struct {
	AuthorID string `json:"authorId"`
	Name     string `json:"name"`
}

type apiAuthorFull struct {
	AuthorID      string     `json:"authorId"`
	Name          string     `json:"name"`
	HIndex        int        `json:"hIndex"`
	CitationCount int        `json:"citationCount"`
	PaperCount    int        `json:"paperCount"`
	Affiliations  []string   `json:"affiliations"`
	Papers        []apiPaper `json:"papers"`
}

type paperSearchResp struct {
	Data  []apiPaper `json:"data"`
	Total int        `json:"total"`
	Next  *int       `json:"next"`
}

type authorSearchResp struct {
	Data  []apiAuthorFull `json:"data"`
	Total int             `json:"total"`
	Next  *int            `json:"next"`
}

type citationItem struct {
	CitingPaper *apiPaper `json:"citingPaper"`
	CitedPaper  *apiPaper `json:"citedPaper"`
}

type citationsResp struct {
	Data []citationItem `json:"data"`
	Next *int           `json:"next"`
}

type recommendResp struct {
	RecommendedPapers []apiPaper `json:"recommendedPapers"`
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func paperURL(id string) string {
	return "https://www.semanticscholar.org/paper/" + id
}

func authorURL(id string) string {
	return "https://www.semanticscholar.org/author/" + id
}

func joinAuthors(authors []apiAuthorShort) string {
	if len(authors) == 0 {
		return ""
	}
	names := make([]string, len(authors))
	for i, a := range authors {
		names[i] = a.Name
	}
	if len(names) <= 3 {
		return strings.Join(names, ", ")
	}
	return strings.Join(names[:3], ", ") + " et al."
}

func apiPaperToRecord(p apiPaper, rank int) Paper {
	year := 0
	if p.Year != nil {
		year = *p.Year
	}
	return Paper{
		Rank:       rank,
		ID:         p.PaperID,
		Title:      p.Title,
		Year:       year,
		Authors:    joinAuthors(p.Authors),
		Venue:      p.Venue,
		Citations:  p.CitationCount,
		OpenAccess: p.IsOpenAccess,
		URL:        paperURL(p.PaperID),
	}
}

func apiAuthorToRecord(a apiAuthorFull, rank int) Author {
	return Author{
		Rank:      rank,
		ID:        a.AuthorID,
		Name:      a.Name,
		HIndex:    a.HIndex,
		Citations: a.CitationCount,
		Papers:    a.PaperCount,
		URL:       authorURL(a.AuthorID),
	}
}
