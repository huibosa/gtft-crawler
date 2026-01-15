package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Parser struct {
	verbose bool
}

func NewParser(verbose bool) *Parser {
	return &Parser{
		verbose: verbose,
	}
}

func (p *Parser) Parse(html []byte, url string) (*PaperMetadata, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(html)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	metadata := NewPaperMetadata(url)

	// Extract article ID from URL
	metadata.ID = extractIDFromURL(url)

	// Run all extractors
	extractors := []func(*goquery.Document, *PaperMetadata) error{
		p.extractMetaTags,
		p.extractTitle,
		p.extractAuthors,
		p.extractJournalInfo,
		p.extractPublicationDetails,
		p.extractAbstract,
		p.extractKeywords,
		p.extractMetrics,
		p.extractDates,
		p.extractAdditionalInfo,
	}

	for _, extractor := range extractors {
		if err := extractor(doc, metadata); err != nil && p.verbose {
			fmt.Printf("Warning in extractor: %v\n", err)
		}
	}

	return metadata, nil
}

func (p *Parser) extractMetaTags(doc *goquery.Document, metadata *PaperMetadata) error {
	// Extract Dublin Core metadata
	doc.Find("meta[name^='dc.']").Each(func(i int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		content, _ := s.Attr("content")

		switch name {
		case "dc.title":
			metadata.TitleCN = content
		case "dc.contributor", "dc.creator":
			// These are handled in extractAuthors
		case "dc.date":
			metadata.Date = content
		case "dc.keywords":
			metadata.KeywordsCN = strings.Split(content, ", ")
		case "dc.description":
			metadata.AbstractCN = content
		case "dc.source":
			// Parse journal info from dc.source
			p.parseJournalSource(content, metadata)
		case "dc.publisher":
			metadata.JournalCN = content
		}
	})

	// Extract citation metadata
	doc.Find("meta[name^='citation_']").Each(func(i int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		content, _ := s.Attr("content")

		switch name {
		case "citation_title":
			metadata.TitleCN = content
		case "citation_authors":
			// Parse comma-separated authors
			authors := strings.Split(content, ", ")
			for i, author := range authors {
				metadata.Authors = append(metadata.Authors, Author{
					Name:  strings.TrimSpace(author),
					Order: i + 1,
				})
			}
		case "citation_journal_title":
			metadata.JournalCN = content
		case "citation_journal_abbrev":
			metadata.JournalAbbr = content
		case "citation_issn":
			metadata.ISSN = content
		case "citation_date", "citation_online_date":
			metadata.Date = content
		case "citation_year":
			metadata.Year = content
		case "citation_volume":
			metadata.Volume = content
		case "citation_issue":
			metadata.Issue = content
		case "citation_firstpage":
			if metadata.Pages == "" {
				metadata.Pages = content
			} else {
				metadata.Pages = content + "-" + strings.Split(metadata.Pages, "-")[1]
			}
		case "citation_lastpage":
			if metadata.Pages == "" {
				metadata.Pages = "-" + content
			} else {
				metadata.Pages = strings.Split(metadata.Pages, "-")[0] + "-" + content
			}
		case "citation_doi":
			metadata.DOI = content
		case "citation_keywords":
			metadata.KeywordsCN = strings.Split(content, ", ")
		case "citation_pdf_url":
			metadata.PDFURL = content
		}
	})

	return nil
}

func (p *Parser) parseJournalSource(source string, metadata *PaperMetadata) {
	// Parse format like: "钢铁钒钛, 2003, Vol. 24, Issue 4, Pages: 1-5"
	parts := strings.Split(source, ", ")
	if len(parts) >= 1 {
		metadata.JournalCN = parts[0]
	}
	if len(parts) >= 2 {
		metadata.Year = parts[1]
	}

	// Parse volume and issue
	re := regexp.MustCompile(`Vol\.\s*(\d+)`)
	if matches := re.FindStringSubmatch(source); len(matches) > 1 {
		metadata.Volume = matches[1]
	}

	re = regexp.MustCompile(`Issue\s*(\d+)`)
	if matches := re.FindStringSubmatch(source); len(matches) > 1 {
		metadata.Issue = matches[1]
	}

	// Parse pages
	re = regexp.MustCompile(`Pages:\s*(\d+-\d+)`)
	if matches := re.FindStringSubmatch(source); len(matches) > 1 {
		metadata.Pages = matches[1]
	}
}

func (p *Parser) extractTitle(doc *goquery.Document, metadata *PaperMetadata) error {
	// Try to get title from various selectors
	selectors := []string{
		"h1", "h2", ".article-title", ".title", "title",
		".header-tit", "h2.article-title",
	}

	for _, selector := range selectors {
		title := doc.Find(selector).First().Text()
		if title != "" && metadata.TitleCN == "" {
			metadata.TitleCN = strings.TrimSpace(title)
			break
		}
	}

	return nil
}

func (p *Parser) extractAuthors(doc *goquery.Document, metadata *PaperMetadata) error {
	// If we already have authors from meta tags, skip
	if len(metadata.Authors) > 0 {
		return nil
	}

	// Try to find authors in the body
	selectors := []string{
		".article-author", ".authors", ".author-list",
		".article-authors", ".contributors",
	}

	for _, selector := range selectors {
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			s.Find("li, span, a").Each(func(j int, authorSel *goquery.Selection) {
				authorText := strings.TrimSpace(authorSel.Text())
				if authorText != "" && !strings.Contains(authorText, "@") {
					// Clean up author name (remove numbers, punctuation)
					authorText = cleanAuthorName(authorText)

					metadata.Authors = append(metadata.Authors, Author{
						Name:  authorText,
						Order: len(metadata.Authors) + 1,
					})
				}
			})
		})

		if len(metadata.Authors) > 0 {
			break
		}
	}

	return nil
}

func cleanAuthorName(name string) string {
	// Remove numbers, punctuation, and extra whitespace
	name = strings.TrimSpace(name)

	// Remove trailing commas, periods, etc.
	name = strings.TrimRight(name, ",.& ")

	// Remove affiliation numbers like "1,", "2,", etc.
	re := regexp.MustCompile(`^\d+[\.,]?\s*`)
	name = re.ReplaceAllString(name, "")

	return name
}

func (p *Parser) extractJournalInfo(doc *goquery.Document, metadata *PaperMetadata) error {
	// Try to find journal info in navigation or headers
	selectors := []string{
		".journal-name", ".journal-title", ".publication-title",
		"nav a", ".breadcrumb a",
	}

	for _, selector := range selectors {
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if strings.Contains(text, "钢铁钒钛") || strings.Contains(text, "IRON STEEL VANADIUM TITANIUM") {
				metadata.JournalCN = "钢铁钒钛"
				metadata.JournalEN = "IRON STEEL VANADIUM TITANIUM"
			}
		})

		if metadata.JournalCN != "" {
			break
		}
	}

	return nil
}

func (p *Parser) extractPublicationDetails(doc *goquery.Document, metadata *PaperMetadata) error {
	// Look for publication details in the page
	doc.Find("div, span, p").Each(func(i int, s *goquery.Selection) {
		text := s.Text()

		// Look for volume/issue pattern
		re := regexp.MustCompile(`(\d+)\((\d+)\):\s*(\d+-\d+)`)
		if matches := re.FindStringSubmatch(text); len(matches) > 3 {
			if metadata.Volume == "" {
				metadata.Volume = matches[1]
			}
			if metadata.Issue == "" {
				metadata.Issue = matches[2]
			}
			if metadata.Pages == "" {
				metadata.Pages = matches[3]
			}
		}

		// Look for year
		re = regexp.MustCompile(`\b(19|20)\d{2}\b`)
		if matches := re.FindStringSubmatch(text); len(matches) > 0 && metadata.Year == "" {
			metadata.Year = matches[0]
		}
	})

	return nil
}

func (p *Parser) extractAbstract(doc *goquery.Document, metadata *PaperMetadata) error {
	// Look for abstract sections
	selectors := []string{
		"[class*='abstract']", "[id*='abstract']",
		".article-abstract", ".abstract-text",
		"p:contains('摘要')", "div:contains('Abstract')",
	}

	for _, selector := range selectors {
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())

			if strings.Contains(text, "摘要") || strings.Contains(selector, "abstract") {
				// Clean the abstract text
				text = strings.TrimPrefix(text, "摘要:")
				text = strings.TrimPrefix(text, "摘要：")
				text = strings.TrimSpace(text)

				if text != "" && metadata.AbstractCN == "" {
					metadata.AbstractCN = text
				}
			}

			if strings.Contains(text, "Abstract") {
				text = strings.TrimPrefix(text, "Abstract:")
				text = strings.TrimPrefix(text, "Abstract：")
				text = strings.TrimSpace(text)

				if text != "" && metadata.AbstractEN == "" {
					metadata.AbstractEN = text
				}
			}
		})
	}

	return nil
}

func (p *Parser) extractKeywords(doc *goquery.Document, metadata *PaperMetadata) error {
	// Look for keyword sections
	selectors := []string{
		"[class*='keyword']", "[id*='keyword']",
		".article-keywords", ".keywords",
		"span:contains('关键词')", "div:contains('Key words')",
	}

	for _, selector := range selectors {
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			text := s.Text()

			if strings.Contains(text, "关键词") {
				// Extract Chinese keywords
				text = strings.TrimPrefix(text, "关键词:")
				text = strings.TrimPrefix(text, "关键词：")
				text = strings.TrimSpace(text)

				if text != "" && len(metadata.KeywordsCN) == 0 {
					// Split by commas, slashes, or Chinese punctuation
					keywords := strings.FieldsFunc(text, func(r rune) bool {
						return r == ',' || r == '，' || r == '/' || r == '、'
					})

					for _, kw := range keywords {
						kw = strings.TrimSpace(kw)
						if kw != "" {
							metadata.KeywordsCN = append(metadata.KeywordsCN, kw)
						}
					}
				}
			}

			if strings.Contains(text, "Key words") {
				// Extract English keywords
				text = strings.TrimPrefix(text, "Key words:")
				text = strings.TrimPrefix(text, "Key words：")
				text = strings.TrimSpace(text)

				if text != "" && len(metadata.KeywordsEN) == 0 {
					keywords := strings.FieldsFunc(text, func(r rune) bool {
						return r == ',' || r == '，' || r == '/' || r == '、'
					})

					for _, kw := range keywords {
						kw = strings.TrimSpace(kw)
						if kw != "" {
							metadata.KeywordsEN = append(metadata.KeywordsEN, kw)
						}
					}
				}
			}
		})
	}

	return nil
}

func (p *Parser) extractMetrics(doc *goquery.Document, metadata *PaperMetadata) error {
	// Look for metrics like views, downloads, citations
	doc.Find("div, span, p").Each(func(i int, s *goquery.Selection) {
		text := s.Text()

		// Look for views count
		if strings.Contains(text, "文章访问数") || strings.Contains(text, "访问数") {
			re := regexp.MustCompile(`(\d+)`)
			if matches := re.FindStringSubmatch(text); len(matches) > 1 {
				if views, err := strconv.Atoi(matches[1]); err == nil {
					metadata.Views = views
				}
			}
		}

		// Look for download count
		if strings.Contains(text, "PDF下载量") || strings.Contains(text, "下载") {
			re := regexp.MustCompile(`(\d+)`)
			if matches := re.FindStringSubmatch(text); len(matches) > 1 {
				if downloads, err := strconv.Atoi(matches[1]); err == nil {
					metadata.Downloads = downloads
				}
			}
		}

		// Look for citation count
		if strings.Contains(text, "被引次数") || strings.Contains(text, "引用") {
			re := regexp.MustCompile(`(\d+)`)
			if matches := re.FindStringSubmatch(text); len(matches) > 1 {
				if citations, err := strconv.Atoi(matches[1]); err == nil {
					metadata.Citations = citations
				}
			}
		}
	})

	return nil
}

func (p *Parser) extractDates(doc *goquery.Document, metadata *PaperMetadata) error {
	// Look for date information
	doc.Find("div, span, p").Each(func(i int, s *goquery.Selection) {
		text := s.Text()

		// Look for submission date
		if strings.Contains(text, "收稿日期") {
			re := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
			if matches := re.FindStringSubmatch(text); len(matches) > 1 && metadata.SubmitDate == "" {
				metadata.SubmitDate = matches[1]
			}
		}

		// Look for online date
		if strings.Contains(text, "网络出版日期") {
			re := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
			if matches := re.FindStringSubmatch(text); len(matches) > 1 && metadata.OnlineDate == "" {
				metadata.OnlineDate = matches[1]
			}
		}

		// Look for publication date
		if strings.Contains(text, "刊出日期") || strings.Contains(text, "出版日期") {
			re := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
			if matches := re.FindStringSubmatch(text); len(matches) > 1 && metadata.Date == "" {
				metadata.Date = matches[1]
			}
		}
	})

	return nil
}

func (p *Parser) extractAdditionalInfo(doc *goquery.Document, metadata *PaperMetadata) error {
	// Look for additional information
	doc.Find("div, span, p").Each(func(i int, s *goquery.Selection) {
		text := s.Text()

		// Look for fund project
		if strings.Contains(text, "基金项目") && metadata.FundProject == "" {
			text = strings.TrimPrefix(text, "基金项目:")
			text = strings.TrimPrefix(text, "基金项目：")
			metadata.FundProject = strings.TrimSpace(text)
		}

		// Look for CLC code
		if strings.Contains(text, "中图分类号") && metadata.CLCCode == "" {
			re := regexp.MustCompile(`[A-Z]+\d+(\.\d+)?`)
			if matches := re.FindStringSubmatch(text); len(matches) > 0 {
				metadata.CLCCode = matches[0]
			}
		}

		// Look for license
		if strings.Contains(text, "creativecommons.org") && metadata.License == "" {
			re := regexp.MustCompile(`https?://[^\s]+`)
			if matches := re.FindStringSubmatch(text); len(matches) > 0 {
				metadata.License = matches[0]
			}
		}
	})

	return nil
}

func extractIDFromURL(url string) string {
	// Extract UUID from URL
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return url
}
