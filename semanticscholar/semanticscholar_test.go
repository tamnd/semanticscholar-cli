package semanticscholar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseID(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"204e3073870fae3d05bcbc2f6a8e263d9b72e776", "204e3073870fae3d05bcbc2f6a8e263d9b72e776"},
		{"10.18653/v1/P16-1162", "DOI:10.18653/v1/P16-1162"},
		{"1706.03762", "ARXIV:1706.03762"},
		{"2005.14165", "ARXIV:2005.14165"},
		{"2005.1416", "ARXIV:2005.1416"},
		{"2005.141650", "2005.141650"},
	}
	for _, tc := range cases {
		got := parseID(tc.input)
		if got != tc.want {
			t.Errorf("parseID(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestSearchPapersReturnsResults(t *testing.T) {
	year := 2017
	body := paperSearchResp{
		Total: 1,
		Data: []apiPaper{
			{
				PaperID:       "abc123",
				Title:         "Attention Is All You Need",
				Year:          &year,
				Authors:       []apiAuthorShort{{AuthorID: "1", Name: "Ashish Vaswani"}},
				CitationCount: 100,
				Venue:         "NeurIPS",
				IsOpenAccess:  true,
			},
		},
	}
	b, _ := json.Marshal(body)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Rate = 0
	c := NewClient(cfg)

	var resp paperSearchResp
	if err := c.getJSON(context.Background(), srv.URL, &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("got %d papers, want 1", len(resp.Data))
	}
	p := apiPaperToRecord(resp.Data[0], 1)
	if p.Title != "Attention Is All You Need" {
		t.Errorf("title = %q", p.Title)
	}
	if p.Year != 2017 {
		t.Errorf("year = %d", p.Year)
	}
	if p.URL != "https://www.semanticscholar.org/paper/abc123" {
		t.Errorf("url = %q", p.URL)
	}
	if !p.OpenAccess {
		t.Error("open_access should be true")
	}
}

func TestPaperByIDMapsFields(t *testing.T) {
	year := 2017
	p := apiPaper{
		PaperID:       "def456",
		Title:         "BERT",
		Year:          &year,
		IsOpenAccess:  false,
		CitationCount: 5000,
		Authors: []apiAuthorShort{
			{AuthorID: "1", Name: "Jacob Devlin"},
			{AuthorID: "2", Name: "Ming-Wei Chang"},
			{AuthorID: "3", Name: "Kenton Lee"},
			{AuthorID: "4", Name: "Kristina Toutanova"},
		},
	}
	b, _ := json.Marshal(p)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Rate = 0
	c := NewClient(cfg)

	var got apiPaper
	if err := c.getJSON(context.Background(), srv.URL, &got); err != nil {
		t.Fatal(err)
	}
	rec := apiPaperToRecord(got, 0)
	if rec.ID != "def456" {
		t.Errorf("id = %q", rec.ID)
	}
	if rec.Authors != "Jacob Devlin, Ming-Wei Chang, Kenton Lee et al." {
		t.Errorf("authors = %q", rec.Authors)
	}
	if rec.Citations != 5000 {
		t.Errorf("citations = %d", rec.Citations)
	}
}

func TestCitationsParsesResponse(t *testing.T) {
	year := 2018
	body := citationsResp{
		Data: []citationItem{
			{CitingPaper: &apiPaper{
				PaperID:       "cite001",
				Title:         "BERT Paper",
				Year:          &year,
				CitationCount: 999,
			}},
		},
	}
	b, _ := json.Marshal(body)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Rate = 0
	c := NewClient(cfg)

	var resp citationsResp
	if err := c.getJSON(context.Background(), srv.URL, &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) == 0 || resp.Data[0].CitingPaper == nil {
		t.Fatal("no citingPaper in response")
	}
	rec := apiPaperToRecord(*resp.Data[0].CitingPaper, 1)
	if rec.ID != "cite001" {
		t.Errorf("id = %q", rec.ID)
	}
	if rec.Year != 2018 {
		t.Errorf("year = %d", rec.Year)
	}
}

func TestSearchAuthorsReturnsResults(t *testing.T) {
	body := authorSearchResp{
		Total: 1,
		Data: []apiAuthorFull{
			{
				AuthorID:      "1741101",
				Name:          "Yoshua Bengio",
				HIndex:        160,
				CitationCount: 450000,
				PaperCount:    620,
				Affiliations:  []string{"University of Montreal"},
			},
		},
	}
	b, _ := json.Marshal(body)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.Rate = 0
	c := NewClient(cfg)

	var resp authorSearchResp
	if err := c.getJSON(context.Background(), srv.URL, &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("got %d authors, want 1", len(resp.Data))
	}
	rec := apiAuthorToRecord(resp.Data[0], 1)
	if rec.HIndex != 160 {
		t.Errorf("h_index = %d", rec.HIndex)
	}
	if rec.Citations != 450000 {
		t.Errorf("citations = %d", rec.Citations)
	}
	if rec.Papers != 620 {
		t.Errorf("papers = %d", rec.Papers)
	}
	if rec.URL != "https://www.semanticscholar.org/author/1741101" {
		t.Errorf("url = %q", rec.URL)
	}
}
