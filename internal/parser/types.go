package parser

import (
	"time"
)

type Author struct {
	Name        string `json:"name"`
	Affiliation string `json:"affiliation,omitempty"`
	Order       int    `json:"order,omitempty"`
}

type PaperMetadata struct {
	// Core Identification
	ID       string `json:"id"`
	URL      string `json:"url"`
	Language string `json:"language"`

	// Titles
	TitleCN string `json:"title_cn"`
	TitleEN string `json:"title_en,omitempty"`

	// Authors & Affiliations
	Authors []Author `json:"authors"`

	// Journal Information
	JournalCN   string `json:"journal_cn"`
	JournalEN   string `json:"journal_en,omitempty"`
	JournalAbbr string `json:"journal_abbr,omitempty"`
	ISSN        string `json:"issn,omitempty"`

	// Publication Details
	Volume string `json:"volume"`
	Issue  string `json:"issue"`
	Pages  string `json:"pages"`
	Year   string `json:"year"`

	// Dates
	Date       string `json:"date"`
	OnlineDate string `json:"online_date,omitempty"`
	SubmitDate string `json:"submit_date,omitempty"`

	// Content
	AbstractCN string   `json:"abstract_cn"`
	AbstractEN string   `json:"abstract_en,omitempty"`
	KeywordsCN []string `json:"keywords_cn"`
	KeywordsEN []string `json:"keywords_en,omitempty"`

	// Resources
	PDFURL  string `json:"pdf_url,omitempty"`
	PDFSize string `json:"pdf_size,omitempty"`

	// Metrics
	Views     int `json:"views"`
	Downloads int `json:"downloads"`
	Citations int `json:"citations"`

	// Academic Metadata
	DOI         string `json:"doi,omitempty"`
	FundProject string `json:"fund_project,omitempty"`
	CLCCode     string `json:"clc_code,omitempty"`
	License     string `json:"license,omitempty"`

	// Timestamps
	ParsedAt string `json:"parsed_at"`
}

func NewPaperMetadata(url string) *PaperMetadata {
	return &PaperMetadata{
		URL:      url,
		Language: "zh",
		ParsedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

func (p *PaperMetadata) Validate() bool {
	if p.ID == "" || p.TitleCN == "" || len(p.Authors) == 0 || p.JournalCN == "" {
		return false
	}
	return true
}
